package main

import (
	"context"
	"fmt"
	"io"
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
	"github.com/gorilla/websocket"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// ── Logging bridge ────────────────────────────────────────────────────────────

type chanWriter chan string

func (cw chanWriter) Write(p []byte) (int, error) {
	str := string(p)
	select {
	case cw <- str:
	default:
	}
	return len(p), nil
}

// ── WebSocket relay stream wrapper ────────────────────────────────────────────

// wsReadWriter wraps a WebSocket connection as an io.ReadWriteCloser so it can
// be passed directly to vnet.Writer / vnet.Reader, which work with any RWC.
// WebSocket is message-framed, so we buffer leftover bytes from each message.
type wsReadWriter struct {
	conn *websocket.Conn
	mu   sync.Mutex // guards writes; reads are single-goroutine
	buf  []byte
}

func newWsRWC(conn *websocket.Conn) *wsReadWriter {
	return &wsReadWriter{conn: conn}
}

func (w *wsReadWriter) Read(p []byte) (int, error) {
	// Drain any leftover bytes from the previous WebSocket message first.
	for len(w.buf) == 0 {
		_, data, err := w.conn.ReadMessage()
		if err != nil {
			return 0, err
		}
		w.buf = data
	}
	n := copy(p, w.buf)
	w.buf = w.buf[n:]
	return n, nil
}

func (w *wsReadWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	// Copy p so the caller can reuse the buffer immediately.
	frame := make([]byte, len(p))
	copy(frame, p)
	if err := w.conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *wsReadWriter) Close() error {
	return w.conn.Close()
}

// ── Peer name helper ──────────────────────────────────────────────────────────

func derivePeerName(h host.Host, pid peer.ID) string {
	name := pid.String()
	agent, err := h.Peerstore().Get(pid, "AgentVersion")
	if err == nil && agent != nil {
		if str, ok := agent.(string); ok && strings.HasPrefix(str, "termophone/") {
			return strings.TrimPrefix(str, "termophone/")
		}
	}
	if len(name) > 12 {
		name = name[len(name)-8:]
	}
	return name
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	logCh := make(chan string, 32)
	cw := chanWriter(logCh)

	handler := slog.NewJSONHandler(cw, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))
	log.SetOutput(cw)
	log.SetFlags(0)

	cfg := config.Get()
	slog.Info("Starting Termophone P2P Node", "user", cfg.Username)

	ctx, cancel := context.WithCancel(context.Background())

	h, incomingStreamCh, err := vnet.SetupHost(ctx, 0, cfg.Username)
	if err != nil {
		log.Fatal("Failed to setup libp2p host:", err)
	}
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
	lobbyUsersCh := make(chan ui.MsgLobbyUsers, 4)
	routingCh := make(chan vnet.RoutingInfo, 2)
	incomingLobbyCallCh := make(chan vnet.IncomingCall, 4)
	muted := &atomic.Bool{}

	// ── startAudio ────────────────────────────────────────────────────────────
	//
	// Accepts any io.ReadWriteCloser so it works with both libp2p streams
	// (direct/LAN) and wsReadWriter (relay). peerName / peerID are shown in
	// the UI; for relay calls both are the remote username.
	startAudio := func(rwc io.ReadWriteCloser, peerName, peerID string) {
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

		var initErr error
		mctx, initErr = malgo.InitContext(nil, malgo.ContextConfig{}, func(msg string) {
			log.Print(msg)
		})
		if initErr != nil {
			log.Printf("Failed to initialize audio context: %v", initErr)
			rwc.Close()
			return
		}

		rb := audio.NewRingBuffer(1024 * 64)

		capturer, initErr = audio.NewCapturer(mctx, rawCh, freePool)
		if initErr != nil {
			log.Printf("Failed to initialize capturer: %v", initErr)
			notifyDisconnected()
			return
		}
		player, initErr = audio.NewPlayer(mctx, rb)
		if initErr != nil {
			log.Printf("Failed to initialize player: %v", initErr)
			notifyDisconnected()
			return
		}

		codec, initErr := audio.NewCodec()
		if initErr != nil {
			log.Printf("Failed to initialize codec: %v", initErr)
			notifyDisconnected()
			return
		}

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

		// Mute gate: only forward frames to the network when not muted.
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

		go func() {
			vnet.Writer(rwc, filteredSendCh)
			notifyDisconnected()
		}()
		go func() {
			vnet.Reader(rwc, recvCh)
			notifyDisconnected()
		}()

		// Heartbeat so the UI duration timer keeps ticking even during silence.
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

		connectCh <- ui.MsgPeerConnected{Name: peerName, ID: peerID}
		log.Printf("Audio transmission started with %s", peerName)
	}

	// ── Lobby client setup ───────────────────────────────────────────────────

	lobbyStateCh := make(chan string, 2)

	var lobbyClient *vnet.LobbyClient
	var lobbyMu sync.RWMutex

	defer func() {
		lobbyMu.RLock()
		if lobbyClient != nil {
			lobbyClient.Close()
		}
		lobbyMu.RUnlock()
	}()

	connectLobby := func(url string) {
		lobbyMu.Lock()
		if lobbyClient != nil {
			lobbyClient.Close()
			lobbyClient = nil
		}
		lobbyMu.Unlock()

		localIPs := vnet.GetLocalIPs()
		lobbyStateCh <- "connecting"

		go func() {
			lc, err := vnet.NewLobbyClient(url, cfg.Username, localIPs)
			if err != nil {
				log.Printf("Lobby unavailable (%v) — running in local-only mode", err)
				lobbyStateCh <- "failed"
				return
			}

			lobbyMu.Lock()
			lobbyClient = lc
			lobbyMu.Unlock()

			lobbyStateCh <- "connected"

			lc.OnClients = func(users []vnet.LobbyUser) {
				out := make([]ui.LobbyUser, len(users))
				for i, u := range users {
					out[i] = ui.LobbyUser{Username: u.Username, PublicIP: u.PublicIP}
				}
				select {
				case lobbyUsersCh <- ui.MsgLobbyUsers{Users: out}:
				default:
				}
			}

			lc.OnRouting = func(r vnet.RoutingInfo) {
				select {
				case routingCh <- r:
				default:
				}
			}

			lc.OnIncoming = func(ic vnet.IncomingCall) {
				select {
				case incomingLobbyCallCh <- ic:
				default:
				}
			}

			lc.OnError = func(err error) {
				log.Printf("Lobby connection error: %v", err)
				lobbyStateCh <- "failed"
				lobbyMu.Lock()
				lobbyClient = nil
				lobbyMu.Unlock()
			}
		}()
	}

	disconnectLobby := func() {
		lobbyMu.Lock()
		if lobbyClient != nil {
			lobbyClient.Close()
			lobbyClient = nil
		}
		lobbyMu.Unlock()
		lobbyStateCh <- "disconnected"
		select {
		case lobbyUsersCh <- ui.MsgLobbyUsers{Users: []ui.LobbyUser{}}:
		default:
		}
	}

	connectLobby(cfg.LobbyServer)

	// ── Incoming lobby call handler ───────────────────────────────────────────
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ic := <-incomingLobbyCallCh:
				log.Printf("Incoming lobby call from %s via %s", ic.CallerUsername, ic.RouteType)
				if ic.RouteType == "relay" {
					go func(ic vnet.IncomingCall) {
						relayURL := ic.RelayAddress + "?id=" + ic.SessionID
						wsConn, _, err := websocket.DefaultDialer.Dial(relayURL, nil)
						if err != nil {
							log.Printf("Relay connect failed for incoming call: %v", err)
							return
						}
						startAudio(newWsRWC(wsConn), ic.CallerUsername, ic.CallerUsername)
					}(ic)
				}
				// For LAN route: caller connects via libp2p → incomingStreamCh fires.
			}
		}
	}()

	// ── dialCb ───────────────────────────────────────────────────────────────
	dialCb := func(id string) error {
		log.Printf("Dialing %s...", id)

		if strings.HasPrefix(id, "12D3") || strings.HasPrefix(id, "Qm") {
			pid, err := peer.Decode(id)
			if err != nil {
				return err
			}

			p := h.Peerstore().PeerInfo(pid)
			if len(p.Addrs) == 0 {
				return fmt.Errorf("no addresses found locally for peer")
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

			buf := make([]byte, 1)
			stream.SetReadDeadline(time.Now().Add(30 * time.Second))
			n, err := stream.Read(buf)
			stream.SetReadDeadline(time.Time{})
			if err != nil || n == 0 || buf[0] != 1 {
				stream.Reset()
				statusCh <- "Call declined by peer"
				return fmt.Errorf("call declined")
			}

			peerName := derivePeerName(h, p.ID)
			go startAudio(stream, peerName, p.ID.String())
			return nil
		}

		lobbyMu.RLock()
		client := lobbyClient
		lobbyMu.RUnlock()

		if client == nil {
			statusCh <- "Lobby not connected"
			return fmt.Errorf("lobby unavailable")
		}

		if err := client.Call(id); err != nil {
			statusCh <- "Call signal failed"
			return err
		}

		select {
		case routing := <-routingCh:
			log.Printf("Routing received: type=%s session=%s", routing.RouteType, routing.SessionID)

			relayURL := routing.RelayAddress + "?id=" + routing.SessionID
			wsConn, _, err := websocket.DefaultDialer.Dial(relayURL, nil)
			if err != nil {
				statusCh <- "Relay connection failed"
				return fmt.Errorf("relay dial: %w", err)
			}
			go startAudio(newWsRWC(wsConn), routing.TargetUsername, routing.TargetUsername)

		case <-time.After(30 * time.Second):
			statusCh <- "Call timed out"
			return fmt.Errorf("routing timeout")
		}

		return nil
	}

	// ── acceptCb (libp2p incoming) ────────────────────────────────────────────
	acceptCb := func(stream network.Stream) error {
		remotePeer := stream.Conn().RemotePeer()
		peerName := derivePeerName(h, remotePeer)
		log.Printf("Accepted incoming call from %s", peerName)
		startAudio(stream, peerName, remotePeer.String())
		return nil
	}

	saveContactCb := func(c config.Contact) {
		config.SaveContact(c)
	}

	removeContactCb := func(peerID string) {
		config.RemoveContact(peerID)
	}

	// ── Build and run the UI ──────────────────────────────────────────────────
	model := ui.NewModel(ui.ModelConfig{
		Host:              h,
		PeerCh:            peerCh,
		StreamCh:          incomingStreamCh,
		LogCh:             logCh,
		AudioCh:           audioCh,
		StatsCh:           statsCh,
		LobbyStateCh:      lobbyStateCh,
		ConnectCh:         connectCh,
		DisconnCh:         disconnCh,
		StatusCh:          statusCh,
		LobbyUsersCh:      lobbyUsersCh,
		ConnectLobbyCb:    connectLobby,
		DisconnectLobbyCb: disconnectLobby,
		Muted:             muted,
		Contacts:          cfg.Contacts,
		DialCb:            dialCb,
		AcceptCb:          acceptCb,
		SaveCb:            saveContactCb,
		RemoveCb:          removeContactCb,
	})

	prog := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		log.Fatal(err)
	}
}
