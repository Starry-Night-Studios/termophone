package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"sync/atomic"
	"time"

	"termophone/config"
	vnet "termophone/net"
	"termophone/session"
	"termophone/ui"

	tea "github.com/charmbracelet/bubbletea"
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

	h, kadDHT, ds, incomingStreamCh, err := vnet.SetupHost(ctx, 0, cfg.Username)
	if err != nil {
		log.Fatal("Failed to setup libp2p host:", err)
	}
	defer ds.Close()
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
	stateSvc, err := vnet.NewStateService(ctx, h, cfg.Username, muted)
	if err != nil {
		log.Fatal("Failed to initialize state service:", err)
	}
	defer stateSvc.Close()

	audioSession, err := session.NewAudioMeshSession(
		ctx,
		h,
		muted,
		func(id, name string) {
			select {
			case connectCh <- ui.MsgPeerConnected{Name: name, ID: id}:
			default:
			}
		},
		func() {
			select {
			case disconnCh <- ui.MsgPeerDisconnected{}:
			default:
			}
		},
		config.Get().AECTrimOffsetMs,
	)
	if err != nil {
		log.Fatal("Failed to initialize audio session:", err)
	}
	defer audioSession.Close()

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

		audioSession.AddStream(stream)
		return nil
	}

	acceptCb := func(stream network.Stream) error {
		log.Printf("Accepted incoming call from %s", stream.Conn().RemotePeer())
		audioSession.AddStream(stream)
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
