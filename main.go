package main

import (
	"context"
	"fmt"
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
	slog.SetDefault(slog.New(slog.NewJSONHandler(cw, &slog.HandlerOptions{Level: slog.LevelDebug})))
	log.SetOutput(cw)
	log.SetFlags(0)

	cfg := config.Get()
	slog.Info("Starting Termophone Client", "user", cfg.Username)
	ctx, cancel := context.WithCancel(context.Background())

	// 1. Setup Local Identity
	h, err := vnet.SetupHost(ctx, 0, cfg.Username)
	if err != nil {
		log.Fatal(err)
	}
	defer h.Close()
	defer cancel()

	peerCh := make(chan peer.AddrInfo, 10)
	vnet.SetupDiscovery(ctx, h, peerCh)

	// 2. Extract our Ed25519 Private Key to authenticate with the Lobby
	privKey := h.Peerstore().PrivKey(h.ID())
	if privKey == nil {
		log.Fatal("Could not retrieve private key from peerstore")
	}
	lobby, err := vnet.NewLobbyClient(cfg.LobbyURL, cfg.Username, privKey, h.ID())
	if err != nil {
		log.Printf("Warning: Failed to connect to Lobby at %s: %v", cfg.LobbyURL, err)
	}

	audioCh := make(chan ui.MsgAudioLevel, 32)
	statsCh := make(chan ui.MsgStats, 10)
	connectCh := make(chan ui.MsgPeerConnected, 2)
	disconnCh := make(chan ui.MsgPeerDisconnected, 2)
	statusCh := make(chan string, 5)
	muted := &atomic.Bool{}
	startAudio := func(room vnet.RoomReady) {
		callCtx, callCancel := context.WithCancel(ctx)
		rawCh, sendCh, recvCh, filteredSendCh := make(chan []byte, 8), make(chan []byte, 8), make(chan []byte, 16), make(chan []byte, 8)

		freePool := make(chan []byte, 32)
		for i := 0; i < 32; i++ {
			freePool <- make([]byte, audio.FrameBytes)
		}

		relayAddr := fmt.Sprintf("%s:%d", room.RelayIP, room.RelayPort)
		relay, err := vnet.NewRelayClient(relayAddr, room.RoomID, room.SecretKey, room.MyID)
		if err != nil {
			log.Println("Relay connection failed:", err)
			callCancel()
			return
		}

		var disconnectOnce sync.Once
		var mctx *malgo.AllocatedContext
		var capturer *audio.Capturer
		var player *audio.Player

		notifyDisconnected := func() {
			disconnectOnce.Do(func() {
				callCancel()
				relay.Close()
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
				}
				select {
				case disconnCh <- ui.MsgPeerDisconnected{}:
				default:
				}
			})
		}

		go func() {
			<-callCtx.Done()
			notifyDisconnected()
		}()

		// Proper error handling for malgo.InitContext
		mctx, err = malgo.InitContext(nil, malgo.ContextConfig{}, func(msg string) { log.Print(msg) })
		if err != nil {
			log.Printf("malgo.InitContext failed: %v", err)
			notifyDisconnected()
			return
		}

		rb := audio.NewRingBuffer(1024 * 64)

		capturer, err = audio.NewCapturer(mctx, rawCh, freePool)
		if err != nil {
			log.Printf("audio.NewCapturer failed: %v", err)
			notifyDisconnected()
			return
		}

		player, err = audio.NewPlayer(mctx, rb)
		if err != nil {
			log.Printf("audio.NewPlayer failed: %v", err)
			notifyDisconnected()
			return
		}

		codec, err := audio.NewCodec()
		if err != nil {
			log.Printf("audio.NewCodec failed: %v", err)
			notifyDisconnected()
			return
		}

		go audio.NewPipeline(rawCh, sendCh, rb, config.Get().AECTrimOffsetMs, freePool).Run(callCtx)
		go audio.RecvPipeline(recvCh, rb, codec)

		capturer.Start()
		player.Start()

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

		// Fire UDP packets at the Relay
		go func() {
			for chunk := range filteredSendCh {
				targetMask := uint8(0xFF ^ (1 << room.MyID))
				relay.SendAudio(chunk, targetMask)
			}
		}()
		relay.StartListening(recvCh)

		// (Removed stray go func with relay.SendAudio(chunk, 255) that caused syntax error)

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

		// Tell the UI we are connected!
		connectCh <- ui.MsgPeerConnected{Name: room.PeerName, ID: ""}
		log.Printf("UDP Session Live on Room %d!", room.RoomID)
	}

	// 4. Background Lobby Event Listener
	if lobby != nil {
		go func() {
			for room := range lobby.RoomReadyCh {
				go startAudio(room)
			}
		}()
	}

	dialCb := func(id string) error {
		if lobby != nil {
			log.Printf("Ringing %s via Lobby...", id)
			lobby.Dial(id)
		} else {
			statusCh <- "Lobby offline"
		}
		return nil
	}

	respondCb := func(callerID string, accept bool) error {
		if lobby != nil {
			lobby.RespondToCall(callerID, accept)
		}
		return nil
	}

	// Safely extract Lobby channels (or leave them nil if offline)
	var lobbyIncoming <-chan vnet.IncomingCall
	var lobbyErr <-chan string
	if lobby != nil {
		lobbyIncoming = lobby.IncomingCallCh
		lobbyErr = lobby.ErrorCh
	}

	model := ui.NewModel(ui.ModelConfig{
		Host:          h,
		PeerCh:        peerCh,
		LogCh:         logCh,
		AudioCh:       audioCh,
		StatsCh:       statsCh,
		ConnectCh:     connectCh,
		DisconnCh:     disconnCh,
		StatusCh:      statusCh,
		Muted:         muted,
		Contacts:      cfg.Contacts,
		DialCb:        dialCb,
		RespondCb:     respondCb,
		SaveCb:        func(c config.Contact) { config.SaveContact(c) },
		RemoveCb:      func(id string) { config.RemoveContact(id) },
		LobbyIncoming: lobbyIncoming, // Use the safe variables
		LobbyErr:      lobbyErr,      // Use the safe variables
	})

	if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
		log.Fatal(err)
	}
}
