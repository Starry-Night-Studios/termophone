package ui

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	"termophone/config"
	vnet "termophone/net"
)

type uiState int

const (
	stateBrowsing uiState = iota
	stateInCall
	stateIncoming
	statePostCall
	stateSettings
)

type ModelConfig struct {
	Host          host.Host
	PeerCh        <-chan peer.AddrInfo
	LogCh         <-chan string
	AudioCh       <-chan MsgAudioLevel
	StatsCh       <-chan MsgStats
	ConnectCh     <-chan MsgPeerConnected
	DisconnCh     <-chan MsgPeerDisconnected
	StatusCh      <-chan string
	Muted         *atomic.Bool
	Contacts      []config.Contact
	DialCb        func(string) error
	RespondCb     func(string, bool) error
	SaveCb        func(config.Contact)
	RemoveCb      func(string)
	LobbyIncoming <-chan vnet.IncomingCall
	LobbyErr      <-chan string
}

type Model struct {
	state uiState
	h     host.Host

	peers          []peer.AddrInfo
	newPeers       []peer.AddrInfo
	contacts       []config.Contact
	cursor         int
	updateCh       <-chan peer.AddrInfo
	
	incomingCallerID   string
	incomingCallerName string

	peerName     string
	peerID       string
	statusMsg    string
	lastPeerID   string
	lastPeerName string

	localRMS  float64
	peerRMS   float64
	muted     *atomic.Bool
	loss      float64
	latencyMs int
	logs      []string

	sharingScreen bool
	cancelScreen  context.CancelFunc

	debug     bool
	callStart time.Time

	settingsCursor int
	usernameInput  textinput.Model
	lobbyInput     textinput.Model
	peerIDInput    textinput.Model
	manualDialMode bool
	colorScheme    int
	screenQuality  string

	logCh         <-chan string
	audioCh       <-chan MsgAudioLevel
	statsCh       <-chan MsgStats
	connectCh     <-chan MsgPeerConnected
	disconnCh     <-chan MsgPeerDisconnected
	statusCh      <-chan string
	lobbyIncoming <-chan vnet.IncomingCall
	lobbyErr      <-chan string

	dialCb    func(string) error
	respondCb func(string, bool) error
	saveCb    func(config.Contact)
	removeCb  func(string)

	WindowWidth  int
	WindowHeight int
}

func NewModel(cfg ModelConfig) Model {
	appCfg := config.Get()
	
	ti := textinput.New()
	ti.Placeholder = "Enter new username..."
	ti.SetValue(appCfg.Username)
	ti.CharLimit = 20

	lobbyTI := textinput.New()
	lobbyTI.Placeholder = "http://127.0.0.1:8080"
	lobbyTI.SetValue(appCfg.LobbyURL)
	lobbyTI.CharLimit = 64

	peerTI := textinput.New()
	peerTI.Placeholder = "Paste peer ID (12D3Koo...)"
	peerTI.CharLimit = 128

	return Model{
		state:         stateBrowsing,
		h:             cfg.Host,
		peers:         make([]peer.AddrInfo, 0),
		newPeers:      make([]peer.AddrInfo, 0),
		contacts:      cfg.Contacts,
		updateCh:      cfg.PeerCh,
		logCh:         cfg.LogCh,
		audioCh:       cfg.AudioCh,
		statsCh:       cfg.StatsCh,
		connectCh:     cfg.ConnectCh,
		disconnCh:     cfg.DisconnCh,
		statusCh:      cfg.StatusCh,
		muted:         cfg.Muted,
		logs:          make([]string, 0),
		dialCb:        cfg.DialCb,
		respondCb:     cfg.RespondCb,
		saveCb:        cfg.SaveCb,
		removeCb:      cfg.RemoveCb,
		lobbyIncoming: cfg.LobbyIncoming,
		lobbyErr:      cfg.LobbyErr,
		debug:         false,
		usernameInput: ti,
		lobbyInput:    lobbyTI,
		peerIDInput:   peerTI,
		colorScheme:   appCfg.ColorScheme,
		screenQuality: appCfg.ScreenQuality,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tickCmd())
}

func (m Model) peerDisplayName(id peer.ID) string {
	agent, err := m.h.Peerstore().Get(id, "AgentVersion")
	displayName := id.String()
	if err == nil && agent != nil {
		if str, ok := agent.(string); ok && strings.HasPrefix(str, "termophone/") {
			displayName = strings.TrimPrefix(str, "termophone/")
		}
	} else if len(displayName) > 12 {
		displayName = displayName[len(displayName)-8:]
	}
	return displayName
}

func drain[T any](ch <-chan T) []T {
	var out []T
	for { select { case v := <-ch: out = append(out, v); default: return out } }
}

func (m *Model) updateNewPeers() {
	m.newPeers = make([]peer.AddrInfo, 0)
	for _, p := range m.peers {
		isContact := false
		for _, c := range m.contacts {
			if c.PeerID == p.ID.String() {
				isContact = true
				break
			}
		}
		if !isContact { m.newPeers = append(m.newPeers, p) }
	}
	total := len(m.contacts) + len(m.newPeers)
	if m.cursor >= total && total > 0 {
		m.cursor = total - 1
	} else if total == 0 {
		m.cursor = 0
	}
}

func (m Model) totalItems() int { return len(m.contacts) + len(m.newPeers) }

func (m Model) isOnline(peerID string) bool {
	for _, p := range m.peers {
		if p.ID.String() == peerID { return true }
	}
	return false
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	skipSettingsInputUpdate := false

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q": return m, tea.Quit
		case "m", "M":
			if m.state == stateInCall { m.muted.Store(!m.muted.Load()) }
		case "v", "V":
			if m.state == stateInCall {
				if m.sharingScreen {
					if m.cancelScreen != nil { m.cancelScreen(); m.cancelScreen = nil }
					m.sharingScreen = false
				} else {
					targetID, _ := peer.Decode(m.peerID)
					ctx, cancel := context.WithCancel(context.Background())
					m.cancelScreen = cancel
					m.sharingScreen = true
					go func() { vnet.StartScreenShare(ctx, m.h, targetID, m.screenQuality) }()
				}
			}
		case "d", "D": m.debug = !m.debug
		}

		if m.state == stateBrowsing {
			if m.manualDialMode {
				switch msg.String() {
				case "esc":
					m.manualDialMode = false; m.peerIDInput.Blur(); m.peerIDInput.SetValue("")
				case "enter":
					peerID := strings.TrimSpace(m.peerIDInput.Value())
					if peerID != "" {
						m.statusMsg = "Connecting to pasted peer ID..."
						if m.dialCb != nil { go m.dialCb(peerID) }
					} else { m.statusMsg = "Paste a valid peer ID first" }
					m.manualDialMode = false; m.peerIDInput.Blur(); m.peerIDInput.SetValue("")
				default:
					var cmd tea.Cmd
					m.peerIDInput, cmd = m.peerIDInput.Update(msg)
					cmds = append(cmds, cmd)
				}
				break
			}

			switch msg.String() {
			case "s", "S":
				m.state = stateSettings
				m.usernameInput.Focus(); m.lobbyInput.Blur()
				m.settingsCursor = 0; skipSettingsInputUpdate = true
			case "p", "P":
				m.manualDialMode = true; m.peerIDInput.Focus(); m.peerIDInput.SetValue("")
			case "r", "R":
				m.peers = nil; m.newPeers = nil; m.updateNewPeers()
				m.statusMsg = "Peer list cleared (waiting for discovery...)"
			case "up", "k": if m.cursor > 0 { m.cursor-- }
			case "down", "j": if m.cursor < m.totalItems()-1 { m.cursor++ }
			case "x", "X", "delete", "backspace":
				if m.cursor < len(m.contacts) && len(m.contacts) > 0 {
					removed := m.contacts[m.cursor]
					if m.removeCb != nil { m.removeCb(removed.PeerID) }
					m.contacts = append(m.contacts[:m.cursor], m.contacts[m.cursor+1:]...)
					m.updateNewPeers()
					if m.cursor >= m.totalItems() && m.cursor > 0 { m.cursor-- }
					m.statusMsg = "Removed contact"
				}
			case "enter", " ":
				if m.totalItems() > 0 {
					var selectedID string
					if m.cursor < len(m.contacts) {
						selectedID = m.contacts[m.cursor].PeerID
					} else {
						idx := m.cursor - len(m.contacts)
						if idx >= 0 && idx < len(m.newPeers) { selectedID = m.newPeers[idx].ID.String() }
					}
					if selectedID != "" {
						m.statusMsg = "Ringing..."
						if m.dialCb != nil { go m.dialCb(selectedID) }
					}
				}
			}
		} else if m.state == stateIncoming {
			switch msg.String() {
			case "y", "Y", "enter":
				if m.respondCb != nil { go m.respondCb(m.incomingCallerID, true) }
				m.statusMsg = "Connecting to call..."
				m.state = stateBrowsing
			case "n", "N", "esc":
				if m.respondCb != nil { go m.respondCb(m.incomingCallerID, false) }
				m.statusMsg = "Connection declined"
				m.state = stateBrowsing
			}
		} else if m.state == statePostCall {
			if msg.String() == "s" || msg.String() == "S" {
				if m.saveCb != nil && m.lastPeerID != "" {
					newContact := config.Contact{Name: m.lastPeerName, PeerID: m.lastPeerID}
					m.saveCb(newContact)
					m.contacts = append(m.contacts, newContact)
					m.updateNewPeers()
				}
			}
			m.state = stateBrowsing
		} else if m.state == stateSettings {
			switch msg.String() {
			case "esc": m.state = stateBrowsing
			case "up": if m.settingsCursor > 0 { m.settingsCursor-- }
			case "down": if m.settingsCursor < 3 { m.settingsCursor++ }
			case "left":
				if m.settingsCursor == 2 { m.colorScheme = (m.colorScheme - 1 + len(themes)) % len(themes) } else if m.settingsCursor == 3 { m.screenQuality = previousQuality(m.screenQuality) }
			case "right":
				if m.settingsCursor == 2 { m.colorScheme = (m.colorScheme + 1) % len(themes) } else if m.settingsCursor == 3 { m.screenQuality = nextQuality(m.screenQuality) }
			case "enter":
				cfg := config.Get()
				cfg.Username = m.usernameInput.Value(); cfg.LobbyURL = m.lobbyInput.Value()
				cfg.ColorScheme = m.colorScheme; cfg.ScreenQuality = m.screenQuality
				config.SaveConfig()
				m.state = stateBrowsing
			}
			if m.state == stateSettings {
				if m.settingsCursor == 0 { m.usernameInput.Focus(); m.lobbyInput.Blur()
				} else if m.settingsCursor == 1 { m.usernameInput.Blur(); m.lobbyInput.Focus()
				} else { m.usernameInput.Blur(); m.lobbyInput.Blur() }
			}
		}

	case MsgTick:
		cmds = append(cmds, tickCmd())

		// Process Lobby Events
		for _, call := range drain(m.lobbyIncoming) {
			if m.state == stateBrowsing {
				m.incomingCallerID = call.CallerID
				m.incomingCallerName = call.CallerName
				m.state = stateIncoming
			}
		}
		for _, errStr := range drain(m.lobbyErr) {
			m.statusMsg = "Lobby Error: " + errStr
		}

		for _, p := range drain(m.updateCh) {
			exists := false
			for _, ep := range m.peers { if ep.ID == p.ID { exists = true; break } }
			if !exists { m.peers = append(m.peers, p); m.updateNewPeers() }
		}

		for _, l := range drain(m.logCh) {
			m.logs = append(m.logs, strings.TrimSpace(l))
			if len(m.logs) > 8 { m.logs = m.logs[len(m.logs)-8:] }
		}

		for _, a := range drain(m.audioCh) {
			m.localRMS = 0.8*m.localRMS + 0.2*a.Local; m.peerRMS = 0.8*m.peerRMS + 0.2*a.Peer
		}

		for _, s := range drain(m.statsCh) {
			m.loss = s.LossPercent; m.latencyMs = s.LatencyMs
		}

		for _, c := range drain(m.connectCh) {
			m.peerName = c.Name; m.peerID = c.ID
			m.state = stateInCall; m.callStart = time.Now(); m.statusMsg = ""
		}

		for _, s := range drain(m.statusCh) { if m.state == stateBrowsing { m.statusMsg = s } }

		for range drain(m.disconnCh) {
			if m.sharingScreen {
				if m.cancelScreen != nil { m.cancelScreen(); m.cancelScreen = nil }
				m.sharingScreen = false
			}
			m.state = stateBrowsing
			m.peerName = ""; m.peerID = ""
		}

	case tea.WindowSizeMsg:
		m.WindowWidth = msg.Width; m.WindowHeight = msg.Height
	}

	if m.state == stateSettings && !skipSettingsInputUpdate {
		var cmd1, cmd2 tea.Cmd
		m.usernameInput, cmd1 = m.usernameInput.Update(msg)
		m.lobbyInput, cmd2 = m.lobbyInput.Update(msg)
		cmds = append(cmds, cmd1, cmd2)
	}

	return m, tea.Batch(cmds...)
}
func (m Model) Cursor() int { return m.cursor }
func previousQuality(current string) string { if current == "high" { return "medium" }; if current == "medium" { return "low" }; return "high" }
func nextQuality(current string) string { if current == "low" { return "medium" }; if current == "medium" { return "high" }; return "low" }