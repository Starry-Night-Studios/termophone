package ui

import (
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"termophone/config"
)

type uiState int

const (
	stateBrowsing uiState = iota
	stateInCall
	stateIncoming
	statePostCall
)

type ModelConfig struct {
	Host      host.Host
	PeerCh    <-chan peer.AddrInfo
	StreamCh  <-chan network.Stream
	LogCh     <-chan string
	AudioCh   <-chan MsgAudioLevel
	StatsCh   <-chan MsgStats
	ConnectCh <-chan MsgPeerConnected
	DisconnCh <-chan MsgPeerDisconnected
	StatusCh  <-chan string
	Muted     *atomic.Bool
	Contacts  []config.Contact
	DialCb    func(string) error
	AcceptCb  func(network.Stream) error
	SaveCb    func(config.Contact)
	RemoveCb  func(string)
}

type Model struct {
	state uiState
	h     host.Host

	peers          []peer.AddrInfo
	newPeers       []peer.AddrInfo
	contacts       []config.Contact
	cursor         int
	updateCh       <-chan peer.AddrInfo
	streamCh       <-chan network.Stream
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

	debug     bool
	callStart time.Time

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
	return Model{
		state:     stateBrowsing,
		h:         cfg.Host,
		peers:     make([]peer.AddrInfo, 0),
		newPeers:  make([]peer.AddrInfo, 0),
		contacts:  cfg.Contacts,
		updateCh:  cfg.PeerCh,
		streamCh:  cfg.StreamCh,
		logCh:     cfg.LogCh,
		audioCh:   cfg.AudioCh,
		statsCh:   cfg.StatsCh,
		connectCh: cfg.ConnectCh,
		disconnCh: cfg.DisconnCh,
		statusCh:  cfg.StatusCh,
		muted:     cfg.Muted,
		logs:      make([]string, 0),
		dialCb:    cfg.DialCb,
		acceptCb:  cfg.AcceptCb,
		saveCb:    cfg.SaveCb,
		removeCb:  cfg.RemoveCb,
		debug:     false,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd())
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

	total := len(m.contacts) + len(m.newPeers)
	if m.cursor >= total && total > 0 {
		m.cursor = total - 1
	} else if total == 0 {
		m.cursor = 0
	}
}

func (m Model) totalItems() int {
	return len(m.contacts) + len(m.newPeers)
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

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "m", "M":
			if m.state == stateInCall {
				m.muted.Store(!m.muted.Load())
			}
		case "d", "D":
			m.debug = !m.debug
		}

		if m.state == stateBrowsing {
			switch msg.String() {
			case "r", "R":
				m.peers = nil
				m.newPeers = nil
				m.updateNewPeers()
				m.statusMsg = "Peer list cleared (waiting for discovery...)"
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < m.totalItems()-1 {
					m.cursor++
				}
			case "x", "X", "delete", "backspace":
				if m.cursor < len(m.contacts) && len(m.contacts) > 0 {
					removed := m.contacts[m.cursor]
					if m.removeCb != nil {
						m.removeCb(removed.PeerID)
					}
					m.contacts = append(m.contacts[:m.cursor], m.contacts[m.cursor+1:]...)
					m.updateNewPeers()
					if m.cursor >= m.totalItems() && m.cursor > 0 {
						m.cursor--
					}
					if removed.Name != "" {
						m.statusMsg = "Removed contact: " + removed.Name
					} else {
						m.statusMsg = "Removed contact"
					}
				}
			case "enter", " ":
				if m.totalItems() > 0 {
					var selectedID string
					var selectedName string
					if m.cursor < len(m.contacts) {
						selectedID = m.contacts[m.cursor].PeerID
						selectedName = m.contacts[m.cursor].Name
					} else {
						idx := m.cursor - len(m.contacts)
						if idx >= 0 && idx < len(m.newPeers) {
							selectedID = m.newPeers[idx].ID.String()
							selectedName = m.peerDisplayName(m.newPeers[idx].ID)
						}
					}
					if selectedID != "" {
						if selectedName == "" {
							selectedName = selectedID
						}
						m.statusMsg = "Dialing " + selectedName + "..."
						if m.dialCb != nil {
							go m.dialCb(selectedID)
						}
					}
				}
			}
		} else if m.state == stateIncoming {
			switch msg.String() {
			case "y", "Y", "enter":
				if m.acceptCb != nil {
					if m.incomingStream != nil {
						m.incomingStream.Write([]byte{1}) // 1 = ACCEPT
					}
					go m.acceptCb(m.incomingStream)
				}
			case "n", "N", "esc":
				if m.incomingStream != nil {
					m.incomingStream.Write([]byte{0}) // 0 = DECLINE
					m.incomingStream.Close()
					m.incomingStream.Reset()
					m.incomingStream = nil
				}
				m.statusMsg = "Call declined"
				m.state = stateBrowsing
			}
		} else if m.state == statePostCall {
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
				m.state = stateBrowsing
			} else {
				m.state = stateBrowsing
			}
		}

	case MsgTick:
		cmds = append(cmds, tickCmd())

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

	return m, tea.Batch(cmds...)
}
func (m Model) Cursor() int { return m.cursor }
