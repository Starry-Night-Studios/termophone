package ui

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
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

type focusPane int

const (
	paneSidebar focusPane = iota
	paneMain
)

type ModelConfig struct {
	Host         host.Host
	PeerCh       <-chan peer.AddrInfo
	StreamCh     <-chan network.Stream
	LogCh        <-chan string
	AudioCh      <-chan MsgAudioLevel
	StatsCh      <-chan MsgStats
	ConnectCh    <-chan MsgPeerConnected
	DisconnCh    <-chan MsgPeerDisconnected
	StatusCh     <-chan string
	LobbyStateCh <-chan string
	LobbyUsersCh <-chan MsgLobbyUsers
	Muted        *atomic.Bool
	Contacts     []config.Contact
	DialCb       func(string) error
	AcceptCb     func(network.Stream) error
	SaveCb       func(config.Contact)
	RemoveCb     func(string)
}

type Model struct {
	state       uiState
	focusedPane focusPane
	h           host.Host

	peers          []peer.AddrInfo
	newPeers       []peer.AddrInfo
	lobbyUsers     []LobbyUser
	contacts       []config.Contact
	cursor         int
	updateCh       <-chan peer.AddrInfo
	streamCh       <-chan network.Stream
	lobbyUsersCh   <-chan MsgLobbyUsers
	incomingStream network.Stream
	selected       *peer.AddrInfo

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
	colorScheme    int
	screenQuality  string

	lobbyState   string
	lobbyStateCh <-chan string

	logCh     <-chan string
	audioCh   <-chan MsgAudioLevel
	statsCh   <-chan MsgStats
	connectCh <-chan MsgPeerConnected
	disconnCh <-chan MsgPeerDisconnected
	statusCh  <-chan string

	dialCb   func(string) error
	acceptCb func(network.Stream) error
	saveCb   func(config.Contact)
	removeCb func(string)

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
	lobbyTI.Placeholder = "ws://localhost:8080/ws"
	lobbyTI.CharLimit = 128
	lobbyTI.SetValue(appCfg.LobbyServer)

	return Model{
		state:         stateBrowsing,
		focusedPane:   paneSidebar,
		h:             cfg.Host,
		peers:         make([]peer.AddrInfo, 0),
		newPeers:      make([]peer.AddrInfo, 0),
		lobbyUsers:    make([]LobbyUser, 0),
		contacts:      cfg.Contacts,
		updateCh:      cfg.PeerCh,
		streamCh:      cfg.StreamCh,
		lobbyUsersCh:  cfg.LobbyUsersCh,
		logCh:         cfg.LogCh,
		audioCh:       cfg.AudioCh,
		statsCh:       cfg.StatsCh,
		connectCh:     cfg.ConnectCh,
		disconnCh:     cfg.DisconnCh,
		statusCh:      cfg.StatusCh,
		lobbyState:    "connecting",
		lobbyStateCh:  cfg.LobbyStateCh,
		muted:         cfg.Muted,
		logs:          make([]string, 0),
		dialCb:        cfg.DialCb,
		acceptCb:      cfg.AcceptCb,
		saveCb:        cfg.SaveCb,
		removeCb:      cfg.RemoveCb,
		debug:         false,
		usernameInput: ti,
		lobbyInput:    lobbyTI,
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
	for {
		select {
		case v := <-ch:
			out = append(out, v)
		default:
			return out
		}
	}
}

func (m Model) filteredLobbyUsers() []LobbyUser {
	myUsername := config.Get().Username
	var out []LobbyUser
	for _, u := range m.lobbyUsers {
		if u.Username != myUsername {
			out = append(out, u)
		}
	}
	return out
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
		if !isContact {
			m.newPeers = append(m.newPeers, p)
		}
	}
	m.clampCursor()
}

func (m *Model) clampCursor() {
	total := m.totalItems()
	if total == 0 {
		m.cursor = 0
	} else if m.cursor >= total {
		m.cursor = total - 1
	}
}

func (m Model) totalItems() int {
	return len(m.contacts) + len(m.filteredLobbyUsers()) + len(m.newPeers)
}

func (m Model) isOnline(peerID string) bool {
	for _, p := range m.peers {
		if p.ID.String() == peerID {
			return true
		}
	}
	return false
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	skipSettingsInputUpdate := false

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// ── Global shortcuts ──────────────────────
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "m", "M":
			if m.state == stateInCall {
				m.muted.Store(!m.muted.Load())
			}
		case "v", "V":
			if m.state == stateInCall {
				if m.sharingScreen {
					if m.cancelScreen != nil {
						m.cancelScreen()
						m.cancelScreen = nil
					}
					m.sharingScreen = false
				} else {
					targetID, _ := peer.Decode(m.peerID)
					ctx, cancel := context.WithCancel(context.Background())
					m.cancelScreen = cancel
					m.sharingScreen = true
					go func() {
						vnet.StartScreenShare(ctx, m.h, targetID, m.screenQuality)
					}()
				}
			}
		case "d", "D":
			m.debug = !m.debug
		}

		// ── State-specific key handling ───────────────────────────────
		switch m.state {
		case stateBrowsing:
			switch msg.String() {
			case "tab":
				if m.focusedPane == paneSidebar {
					m.focusedPane = paneMain
				} else {
					m.focusedPane = paneSidebar
				}

			case "s", "S":
				m.state = stateSettings
				m.settingsCursor = 0
				m.usernameInput.Focus()
				m.lobbyInput.Blur()
				skipSettingsInputUpdate = true

			case "r", "R":
				m.peers = nil
				m.newPeers = nil
				m.updateNewPeers()
				m.statusMsg = "Peer list cleared (waiting for discovery...)"

			case "up", "k":
				if m.focusedPane == paneSidebar && m.cursor > 0 {
					m.cursor--
				}

			case "down", "j":
				if m.focusedPane == paneSidebar && m.cursor < m.totalItems()-1 {
					m.cursor++
				}

			case "x", "X", "delete", "backspace":
				if m.focusedPane == paneSidebar {
					if m.cursor < len(m.contacts) && len(m.contacts) > 0 {
						removed := m.contacts[m.cursor]
						if m.removeCb != nil {
							m.removeCb(removed.PeerID)
						}
						m.contacts = append(m.contacts[:m.cursor], m.contacts[m.cursor+1:]...)
						m.updateNewPeers()
						if removed.Name != "" {
							m.statusMsg = "Removed contact: " + removed.Name
						} else {
							m.statusMsg = "Removed contact"
						}
					}
				}

			case "enter", " ":
				if m.focusedPane == paneSidebar && m.totalItems() > 0 {
					var selectedID, selectedName string
					filtered := m.filteredLobbyUsers()
					contactsLen := len(m.contacts)
					lobbyLen := len(filtered)

					switch {
					case m.cursor < contactsLen:
						selectedID = m.contacts[m.cursor].PeerID
						selectedName = m.contacts[m.cursor].Name
					case m.cursor < contactsLen+lobbyLen:
						u := filtered[m.cursor-contactsLen]
						selectedID = u.Username
						selectedName = u.Username
					default:
						idx := m.cursor - contactsLen - lobbyLen
						if idx >= 0 && idx < len(m.newPeers) {
							selectedID = m.newPeers[idx].ID.String()
							selectedName = m.peerDisplayName(m.newPeers[idx].ID)
						}
					}

					if selectedID != "" {
						if selectedName == "" {
							selectedName = selectedID
						}
						m.statusMsg = "Connecting to " + selectedName + "..."
						if m.dialCb != nil {
							go m.dialCb(selectedID)
						}
					}
				}
			}

		case stateIncoming:
			switch msg.String() {
			case "y", "Y", "enter":
				if m.acceptCb != nil {
					if m.incomingStream != nil {
						m.incomingStream.Write([]byte{1}) // ACCEPT
					}
					go m.acceptCb(m.incomingStream)
				}
			case "n", "N", "esc":
				if m.incomingStream != nil {
					m.incomingStream.Write([]byte{0}) // DECLINE
					m.incomingStream.Close()
					m.incomingStream.Reset()
					m.incomingStream = nil
				}
				m.statusMsg = "Connection declined"
				m.state = stateBrowsing
			}

		case statePostCall:
			if msg.String() == "s" || msg.String() == "S" {
				if m.saveCb != nil && m.lastPeerID != "" {
					newContact := config.Contact{Name: m.lastPeerName, PeerID: m.lastPeerID}
					m.saveCb(newContact)
					exists := false
					for _, c := range m.contacts {
						if c.PeerID == m.lastPeerID {
							exists = true
							break
						}
					}
					if !exists {
						m.contacts = append(m.contacts, newContact)
						m.updateNewPeers()
					}
				}
			}
			m.state = stateBrowsing

		case stateSettings:
			switch msg.String() {
			case "esc":
				m.state = stateBrowsing
			case "up":
				if m.settingsCursor > 0 {
					m.settingsCursor--
				}
			case "down":
				if m.settingsCursor < 3 {
					m.settingsCursor++
				}
			case "left":
				if m.settingsCursor == 1 {
					m.colorScheme = (m.colorScheme - 1 + len(themes)) % len(themes)
				} else if m.settingsCursor == 2 {
					m.screenQuality = previousQuality(m.screenQuality)
				}
			case "right":
				if m.settingsCursor == 1 {
					m.colorScheme = (m.colorScheme + 1) % len(themes)
				} else if m.settingsCursor == 2 {
					m.screenQuality = nextQuality(m.screenQuality)
				}
			case "enter":
				cfg := config.Get()
				cfg.Username = m.usernameInput.Value()
				cfg.ColorScheme = m.colorScheme
				cfg.ScreenQuality = m.screenQuality
				cfg.LobbyServer = m.lobbyInput.Value()
				config.SaveConfig()
				m.state = stateBrowsing
			}

			if m.state == stateSettings {
				if m.settingsCursor == 0 {
					m.usernameInput.Focus()
					m.lobbyInput.Blur()
				} else if m.settingsCursor == 3 {
					m.lobbyInput.Focus()
					m.usernameInput.Blur()
				} else {
					m.usernameInput.Blur()
					m.lobbyInput.Blur()
				}
			}
		}

	case MsgTick:
		cmds = append(cmds, tickCmd())

		for _, state := range drain(m.lobbyStateCh) {
			m.lobbyState = state
		}

		for _, p := range drain(m.updateCh) {
			exists := false
			for _, ep := range m.peers {
				if ep.ID == p.ID {
					exists = true
					break
				}
			}
			if !exists {
				m.peers = append(m.peers, p)
				m.updateNewPeers()
			}
		}

		if m.lobbyUsersCh != nil {
			for _, lu := range drain(m.lobbyUsersCh) {
				m.lobbyUsers = lu.Users
				m.clampCursor()
			}
		}

		for _, s := range drain(m.streamCh) {
			if m.state == stateBrowsing {
				m.incomingStream = s
				m.state = stateIncoming
			} else {
				s.Reset()
			}
		}

		for _, l := range drain(m.logCh) {
			m.logs = append(m.logs, strings.TrimSpace(l))
			if len(m.logs) > 8 {
				m.logs = m.logs[len(m.logs)-8:]
			}
		}

		for _, a := range drain(m.audioCh) {
			m.localRMS = 0.8*m.localRMS + 0.2*a.Local
			m.peerRMS = 0.8*m.peerRMS + 0.2*a.Peer
		}

		for _, s := range drain(m.statsCh) {
			m.loss = s.LossPercent
			m.latencyMs = s.LatencyMs
		}

		for _, c := range drain(m.connectCh) {
			m.peerName = c.Name
			m.peerID = c.ID
			m.state = stateInCall
			m.callStart = time.Now()
			m.statusMsg = ""
		}

		for _, s := range drain(m.statusCh) {
			if m.state == stateBrowsing {
				m.statusMsg = s
			}
		}

		for range drain(m.disconnCh) {
			if m.sharingScreen {
				if m.cancelScreen != nil {
					m.cancelScreen()
					m.cancelScreen = nil
				}
				m.sharingScreen = false
			}

			isContact := false
			for _, c := range m.contacts {
				if c.PeerID == m.peerID {
					isContact = true
					break
				}
			}

			if !isContact && m.peerID != "" {
				m.lastPeerID = m.peerID
				m.lastPeerName = m.peerName
				m.state = statePostCall
			} else {
				m.state = stateBrowsing
			}
			m.peerName = ""
			m.peerID = ""
			m.incomingStream = nil
			m.selected = nil
		}

	case tea.WindowSizeMsg:
		m.WindowWidth = msg.Width
		m.WindowHeight = msg.Height
	}

	if m.state == stateSettings && !skipSettingsInputUpdate {
		var cmd tea.Cmd
		if m.settingsCursor == 0 {
			m.usernameInput, cmd = m.usernameInput.Update(msg)
			cmds = append(cmds, cmd)
		} else if m.settingsCursor == 3 {
			m.lobbyInput, cmd = m.lobbyInput.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) Cursor() int { return m.cursor }

func previousQuality(current string) string {
	switch current {
	case "high":
		return "medium"
	case "medium":
		return "low"
	default:
		return "high"
	}
}

func nextQuality(current string) string {
	switch current {
	case "low":
		return "medium"
	case "medium":
		return "high"
	default:
		return "low"
	}
}