package main

import (
	"context"
	"log"
	"log/slog"
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

	handler := slog.NewJSONHandler(cw, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slogger := slog.New(handler)
	slog.SetDefault(slogger)

	log.SetOutput(cw)
	log.SetFlags(0)

	cfg := config.Get()
	slog.Info("Starting Termophone Client", "user", cfg.Username)

	ctx, cancel := context.WithCancel(context.Background())

	// 1. Updated SetupHost call (No more DHT or incoming stream channels)
	h, err := vnet.SetupHost(ctx, 0, cfg.Username)
	if err != nil {
		log.Fatal("Failed to setup libp2p host:", err)
	}
	defer h.Close()
	defer cancel()

	// Dummy channel to keep the UI compiler happy until we rebuild the Lobby UI
	incomingStreamCh := make(chan network.Stream)

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

	// 2. startAudio now takes our new RelayClient instead of a libp2p stream
	startAudio := func(relay *vnet.RelayClient, remoteName string, remoteID string) {
		callCtx, callCancel := context.WithCancel(ctx)
		rawCh := make(chan []byte, 8)
		sendCh := make(chan []byte, 8)
		recvCh := make(chan []byte, 16)
		filteredSendCh := make(chan []byte, 8)

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
				relay.Close() // Close the UDP socket
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
			log.Printf("Failed to init audio context: %v", err)
			return
		}

		rb := audio.NewRingBuffer(1024 * 64)

		capturer, err = audio.NewCapturer(mctx, rawCh, freePool)
		if err != nil {
			notifyDisconnected()
			return
		}
		player, err = audio.NewPlayer(mctx, rb)
		if err != nil {
			notifyDisconnected()
			return
		}

		codec, err := audio.NewCodec()
		if err != nil {
			notifyDisconnected()
			return
		}

		aecDelayMs := config.Get().AECTrimOffsetMs

		go audio.NewPipeline(rawCh, sendCh, rb, aecDelayMs, freePool).Run(callCtx)
		go audio.RecvPipeline(recvCh, rb, codec)

		if err := capturer.Start(); err != nil {
			notifyDisconnected()
			return
		}
		if err := player.Start(); err != nil {
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
						}
					}
				}
			}
		}()

		// 3. UDP WRITER: Send filtered mic data to the Relay
		go func() {
			for chunk := range filteredSendCh {
				// TargetMask 255 = Broadcast to all peers in the room
				relay.SendAudio(chunk, 255) 
			}
		}()

		// 4. UDP READER: Listen for incoming audio from the Relay
		relay.StartListening(recvCh)

		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for range ticker.C {
				select {
				case <-callCtx.Done():
					return
				case audioCh <- ui.MsgAudioLevel{Local: 1.0, Peer: 1.0}: 
				default:
				}
			}
		}()

		connectCh <- ui.MsgPeerConnected{Name: remoteName, ID: remoteID}
		log.Printf("UDP Audio transmission started via Relay!")
	}

	dialCb := func(id string) error {
		log.Printf("Dialing via Relay...")
		
		// HACK: Hardcoded local connection for testing!
		// We are pretending the Lobby told us to join Room 777.
		relay, err := vnet.NewRelayClient("127.0.0.1:4000", 777, 111222, 1)
		if err != nil {
			statusCh <- "Relay unreachable"
			return err
		}

		// Spin up the microphone and speaker pipeline
		go startAudio(relay, "Room 777", id)
		return nil
	}

	acceptCb := func(stream network.Stream) error {
		// Deprecated: We don't accept incoming P2P streams anymore.
		return nil
	}

	saveContactCb := func(c config.Contact) { config.SaveContact(c) }
	removeContactCb := func(peerID string) { config.RemoveContact(peerID) }

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