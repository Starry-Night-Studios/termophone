package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"termophone/audio"
	"termophone/config"
	vnet "termophone/net"
	"termophone/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gen2brain/malgo"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

type chanWriter chan string

func (cw chanWriter) Write(p []byte) (int, error) {
	str := string(p)
	select {
	case cw <- str:
	default:
	}
	return len(p), nil
}

func main() {
	logCh := make(chan string, 32)
	cw := chanWriter(logCh)

	// Create a native structured JSON logger
	handler := slog.NewJSONHandler(cw, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slogger := slog.New(handler)
	slog.SetDefault(slogger)

	// Bridge the standard "log" package to output JSON via slog
	log.SetOutput(cw)
	log.SetFlags(0)

	cfg := config.Get()
	slog.Info("Starting Termophone P2P Node", "user", cfg.Username)

	ctx, cancel := context.WithCancel(context.Background())

	h, kadDHT, incomingStreamCh, err := vnet.SetupHost(ctx, 0, cfg.Username)
	if err != nil {
		log.Fatal("Failed to setup libp2p host:", err)
	}
	defer kadDHT.Close()
	defer h.Close()
	defer cancel()

	peerCh := make(chan peer.AddrInfo, 10)
	if err := vnet.SetupDiscovery(ctx, h, peerCh); err != nil {
		log.Fatal("Failed to start mDNS:", err)
	}

	audioCh := make(chan ui.MsgAudioLevel, 32)
	statsCh := make(chan ui.MsgStats, 10)
	connectCh := make(chan ui.MsgPeerConnected, 2)
	disconnCh := make(chan ui.MsgPeerDisconnected, 2)
	statusCh := make(chan string, 5)
	muted := &atomic.Bool{}

	startAudio := func(stream network.Stream) {
		callCtx, callCancel := context.WithCancel(ctx)
		rawCh := make(chan []byte, 8)
		sendCh := make(chan []byte, 8)
		recvCh := make(chan []byte, 16)
		filteredSendCh := make(chan []byte, 8)

		// Pre-allocated Zero-Allocation audio capture free-list!
		// 32 frames of buffers allows plenty of headroom for the pipeline.
		freePool := make(chan []byte, 32)
		for i := 0; i < 32; i++ {
			freePool <- make([]byte, audio.FrameBytes)
		}

		var disconnectOnce sync.Once
		var mctx *malgo.AllocatedContext
		var capturer *audio.Capturer
		var player *audio.Player

		notifyDisconnected := func() {
			disconnectOnce.Do(func() {
				callCancel()
				if capturer != nil {
					capturer.Stop()
					capturer.Uninit()
				}
				if player != nil {
					player.Stop()
					player.Uninit()
				}
				if mctx != nil {
					mctx.Free()
					mctx = nil
				}
				select {
				case disconnCh <- ui.MsgPeerDisconnected{}:
				default:
				}
			})
		}

		var err error
		mctx, err = malgo.InitContext(nil, malgo.ContextConfig{}, func(msg string) {
			log.Print(msg)
		})
		if err != nil {
			log.Printf("Failed to initialize audio context: %v", err)
			return
		}

		rb := audio.NewRingBuffer(1024 * 64)

		capturer, err = audio.NewCapturer(mctx, rawCh, freePool)
		if err != nil {
			log.Printf("Failed to initialize capturer: %v", err)
			notifyDisconnected()
			return
		}
		player, err = audio.NewPlayer(mctx, rb)
		if err != nil {
			log.Printf("Failed to initialize player: %v", err)
			notifyDisconnected()
			return
		}

		codec, err := audio.NewCodec()
		if err != nil {
			log.Printf("Failed to initialize codec: %v", err)
			notifyDisconnected()
			return
		}

		// Fetch aecDelay parameter from config
		aecDelayMs := config.Get().AECTrimOffsetMs

		go audio.NewPipeline(rawCh, sendCh, rb, aecDelayMs, freePool).Run(callCtx)
		go audio.RecvPipeline(recvCh, rb, codec)

		if err := capturer.Start(); err != nil {
			log.Printf("Failed to start capturer: %v", err)
			notifyDisconnected()
			return
		}
		if err := player.Start(); err != nil {
			log.Printf("Failed to start player: %v", err)
			notifyDisconnected()
			return
		}

		go func() {
			for {
				select {
				case <-callCtx.Done():
					return
				case b, ok := <-sendCh:
					if !ok {
						return
					}
					if !muted.Load() {
						select {
						case filteredSendCh <- b:
						default:
							// drop frame while muted/backpressured
						}
					}
				}
			}
		}()
		go func() {
			vnet.Writer(stream, filteredSendCh)
			notifyDisconnected()
		}()
		go func() {
			vnet.Reader(stream, recvCh)
			notifyDisconnected()
		}()

		go func() {
			// Throttle to 1 FPS for basic UI state updates (duration timer, etc)
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for range ticker.C {
				select {
				case <-callCtx.Done():
					return
				case audioCh <- ui.MsgAudioLevel{Local: 1.0, Peer: 1.0}: // Minimal ping
				default:
				}
			}
		}()

		remoteNamed := stream.Conn().RemotePeer().String()
		agent, err := h.Peerstore().Get(stream.Conn().RemotePeer(), "AgentVersion")
		if err == nil && agent != nil {
			if str, ok := agent.(string); ok && strings.HasPrefix(str, "termophone/") {
				remoteNamed = strings.TrimPrefix(str, "termophone/")
			}
		} else if len(remoteNamed) > 12 {
			remoteNamed = remoteNamed[len(remoteNamed)-8:]
		}

		connectCh <- ui.MsgPeerConnected{Name: remoteNamed, ID: stream.Conn().RemotePeer().String()}

		log.Printf("Audio transmission started!")
	}

	dialCb := func(id string) error {
		log.Printf("Dialing %s...", id)
		pid, err := peer.Decode(id)
		if err != nil {
			return err
		}

		p := h.Peerstore().PeerInfo(pid)
		if len(p.Addrs) == 0 {
			dhtCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			addrInfo, err := kadDHT.FindPeer(dhtCtx, pid)
			if err == nil {
				p = addrInfo
			} else {
				log.Printf("Warning: DHT resolution failed: %v", err)
				return fmt.Errorf("no addresses found locally or via DHT: %w", err)
			}
		}

		if err := h.Connect(ctx, p); err != nil {
			log.Printf("Connect failed: %v", err)
			statusCh <- "Call failed: Unreachable"
			return err
		}
		stream, err := h.NewStream(ctx, p.ID, vnet.ProtocolID)
		if err != nil {
			log.Printf("Stream failed: %v", err)
			statusCh <- "Call failed: Protocol error / Unreachable"
			return err
		}

		// Wait for remote to accept or decline the call
		buf := make([]byte, 1)
		stream.SetReadDeadline(time.Now().Add(30 * time.Second))
		n, err := stream.Read(buf)
		stream.SetReadDeadline(time.Time{})

		if err != nil || n == 0 || buf[0] != 1 {
			stream.Reset()
			statusCh <- "Call declined by peer"
			return fmt.Errorf("call declined")
		}

		go startAudio(stream)
		return nil
	}

	acceptCb := func(stream network.Stream) error {
		log.Printf("Accepted incoming call from %s", stream.Conn().RemotePeer())
		startAudio(stream)
		return nil
	}

	saveContactCb := func(c config.Contact) {
		config.SaveContact(c)
	}

	removeContactCb := func(peerID string) {
		config.RemoveContact(peerID)
	}

	model := ui.NewModel(ui.ModelConfig{
		Host:      h,
		PeerCh:    peerCh,
		StreamCh:  incomingStreamCh,
		LogCh:     logCh,
		AudioCh:   audioCh,
		StatsCh:   statsCh,
		ConnectCh: connectCh,
		DisconnCh: disconnCh,
		StatusCh:  statusCh,
		Muted:     muted,
		Contacts:  cfg.Contacts,
		DialCb:    dialCb,
		AcceptCb:  acceptCb,
		SaveCb:    saveContactCb,
		RemoveCb:  removeContactCb,
	})

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
