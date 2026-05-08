package ui

import (
	"fmt"
	"math"
	"strings"
	"termophone/config"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ── ASCII Logo ──────────────────────────────────────────────────────────
const termophoneASCII = `
  ______                                __                   
 /_  __/__  _________ ___  ____  ____  / /_  ____  ____  ___ 
  / / / _ \/ ___/ __ \__ \/ __ \/ __ \/ __ \/ __ \/ __ \/ _ \
 / / /  __/ /  / / / / / / /_/ / /_/ / / / / /_/ / / / /  __/
/_/  \___/_/  /_/ /_/ /_/\____/ .___/_/ /_/\____/_/ /_/\___/ 
                             /_/                             `

type Styles struct {
	Border   lipgloss.Style
	Sidebar  lipgloss.Style
	Main     lipgloss.Style
	Title    lipgloss.Style
	Info     lipgloss.Style
	Meter    lipgloss.Style
	Log      lipgloss.Style
	Help     lipgloss.Style
	Online   lipgloss.Style
	Offline  lipgloss.Style
	Dim      lipgloss.Style
	Selected lipgloss.Style
}

func (m Model) getStyles() Styles {
	idx := m.colorScheme
	if idx < 0 || idx >= len(themes) {
		idx = 0
	}
	th := themes[idx]

	return Styles{
		Border:   lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(th.Dim),
		Sidebar:  lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, true, false, false).BorderForeground(th.Dim).Padding(0, 1),
		Main:     lipgloss.NewStyle().Padding(0, 1),
		Title:    lipgloss.NewStyle().Bold(true).Foreground(th.Title),
		Info:     lipgloss.NewStyle().Foreground(th.Info),
		Meter:    lipgloss.NewStyle().Foreground(th.Meter),
		Log:      lipgloss.NewStyle().Foreground(th.Log),
		Help:     lipgloss.NewStyle().Foreground(th.Help),
		Online:   lipgloss.NewStyle().Foreground(th.Online),
		Offline:  lipgloss.NewStyle().Foreground(th.Offline),
		Dim:      lipgloss.NewStyle().Foreground(th.Dim),
		Selected: lipgloss.NewStyle().Background(th.Bg2).Foreground(th.Hi).Bold(true).Width(26),
	}
}

func rmsToDb(rms float64) string {
	if rms <= 0 {
		return "-inf dB"
	}
	db := 20 * math.Log10(rms)
	return fmt.Sprintf("%-6.0fdB", db)
}

// ── UTILITIES ───────────────────────────────────────────────────────────

// cropHeight physically prevents long text from stretching the box downward
func cropHeight(s string, max int) string {
	if max <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) > max {
		return strings.Join(lines[:max], "\n")
	}
	return s
}

// ── CUSTOM BORDER DRAWING (With strict dimensions) ──────────────────────
func (m Model) wrapWithTitle(content string, title string, width int, height int) string {
	st := m.getStyles()

	borderColor := st.Dim.GetForeground()

	targetInnerW := width - 4 // -2 padding, -2 borders
	if targetInnerW < 1 {
		targetInnerW = 1
	}
	targetInnerH := height - 2 // -1 top bar, -1 bottom border
	if targetInnerH < 1 {
		targetInnerH = 1
	}

	// explicit width/height forces the box to occupy space uniformly
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, true, true).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(targetInnerW).
		Height(targetInnerH)

	renderedBox := boxStyle.Render(content)
	actualWidth := lipgloss.Width(renderedBox)

	leftStr := "┌─"
	midStr := title
	if title != "" {
		midStr = " " + title + " "
	} else {
		leftStr = "┌"
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	titleStyle := lipgloss.NewStyle().Foreground(borderColor)

	left := borderStyle.Render(leftStr)
	mid := titleStyle.Render(midStr)

	remLen := actualWidth - lipgloss.Width(leftStr) - lipgloss.Width(midStr) - 1 // 1 for the '┐'
	if remLen < 0 {
		remLen = 0
	}
	right := borderStyle.Render(strings.Repeat("─", remLen) + "┐")
	topBar := left + mid + right

	return lipgloss.JoinVertical(lipgloss.Top, topBar, renderedBox)
}

// ── TOP LEFT PANE: CONTACTS ─────────────────────────────────────────────
func (m Model) renderContactsPane() string {
	st := m.getStyles()
	b := strings.Builder{}
	b.WriteString("\n")

	for i, c := range m.contacts {
		status := st.Offline.Render("[!]")
		if m.isOnline(c.PeerID) {
			status = st.Online.Render("[O]")
		}

		name := c.Name
		if name == "" {
			if len(c.PeerID) > 8 {
				name = c.PeerID[len(c.PeerID)-8:]
			} else {
				name = c.PeerID
			}
		}

		if m.cursor == i {
			rawStatus := "[!]"
			if m.isOnline(c.PeerID) {
				rawStatus = "[O]"
			}
			rowText := fmt.Sprintf("%s %s", rawStatus, name)
			if len(rowText) > 26 {
				rowText = rowText[:26]
			}
			b.WriteString(st.Selected.Render(rowText) + "\n")
		} else {
			b.WriteString(fmt.Sprintf("%s %s\n", status, name))
		}
	}

	if len(m.contacts) == 0 {
		b.WriteString(st.Dim.Render("(empty)"))
	}

	return b.String()
}

// ── BOTTOM LEFT PANE: ONLINE (Lobby & Local) ────────────────────────────
func (m Model) renderOnlinePane() string {
	st := m.getStyles()
	b := strings.Builder{}
	b.WriteString("\n")

	filtered := m.filteredLobbyUsers()
	contactsLen := len(m.contacts)

	for i, u := range filtered {
		idx := contactsLen + i
		if m.cursor == idx {
			rowText := fmt.Sprintf("[O] %s", u.Username)
			if len(rowText) > 26 {
				rowText = rowText[:26]
			}
			b.WriteString(st.Selected.Render(rowText) + "\n")
		} else {
			b.WriteString(fmt.Sprintf("%s %s\n", st.Online.Render("[O]"), u.Username))
		}
	}

	localOffset := contactsLen + len(filtered)
	for i, p := range m.newPeers {
		idx := localOffset + i
		displayName := m.peerDisplayName(p.ID)
		if m.cursor == idx {
			if len(displayName) > 26 {
				displayName = displayName[:26]
			}
			b.WriteString(st.Selected.Render(displayName) + "\n")
		} else {
			b.WriteString(fmt.Sprintf("%s\n", displayName))
		}
	}

	if len(filtered) == 0 && len(m.newPeers) == 0 {
		b.WriteString(st.Dim.Render("(empty)\n"))
	}

	return b.String()
}

// ── MAIN CONTENT PANE ───────────────────────────────────────────────────
func (m Model) renderMainPane(innerAvailableWidth int) string {
	st := m.getStyles()
	b := strings.Builder{}

	switch m.state {
	case stateSettings:
		b.WriteString("\n      Settings\n\n")
		scm := themeNames[0]
		if m.colorScheme >= 0 && m.colorScheme < len(themeNames) {
			scm = themeNames[m.colorScheme]
		}
		qualityLabel := "Medium"
		switch m.screenQuality {
		case "low":
			qualityLabel = "Low"
		case "high":
			qualityLabel = "High"
		}

		usrStr := fmt.Sprintf("Username : %s", m.usernameInput.View())
		colStr := fmt.Sprintf("Theme    : < %s >", scm)
		qStr := fmt.Sprintf("Quality  : < %s >", qualityLabel)
		lobbyStr := fmt.Sprintf("Lobby    : %s", m.lobbyInput.View())

		// Ensure settings highlight box doesn't stretch past the window
		w := innerAvailableWidth
		if w > 60 {
			w = 60
		}
		settingsSelected := st.Selected.Copy().Width(w).PaddingLeft(6)

		for i, row := range []string{usrStr, colStr, qStr, lobbyStr} {
			if m.settingsCursor == i {
				b.WriteString(settingsSelected.Render(row) + "\n\n")
			} else {
				b.WriteString("      " + row + "\n\n")
			}
		}
		b.WriteString("\n      [Esc] cancel   [Enter] save\n      [Left/Right] change theme/quality\n")

	case stateBrowsing:
		// Responsive ASCII Art
		if innerAvailableWidth >= 65 {
			b.WriteString("\n" + st.Title.Render(termophoneASCII) + "\n\n")
		} else {
			// Fallback text logo for small terminal windows
			b.WriteString("\n" + st.Title.Render("  TERMOPHONE") + "\n\n")
		}

		lobbyURL := m.lobbyInput.Value()
		switch m.lobbyState {
		case "connecting":
			b.WriteString(fmt.Sprintf("      Connecting to lobby : %s...\n", st.Dim.Render(lobbyURL)))
		case "connected":
			b.WriteString(fmt.Sprintf("      Connected to lobby  : %s\n", st.Info.Render(lobbyURL)))
		default:
			b.WriteString(fmt.Sprintf("      %s\n", st.Dim.Render("Lobby disconnected (Local only)")))
		}

		if m.manualDialMode {
			b.WriteString("\n      Paste peer ID and press [Enter]:\n\n")
			b.WriteString("      " + m.peerIDInput.View() + "\n")
			b.WriteString(st.Dim.Render("\n      [Enter] connect   [Esc] cancel\n"))
		} else {
			b.WriteString("\n      Select a peer and press [Enter] to connect, or [P] to paste ID.\n")
		}

		if m.statusMsg != "" {
			b.WriteString(fmt.Sprintf("\n      %s\n", st.Info.Render(m.statusMsg)))
		}

	case statePostCall:
		b.WriteString(fmt.Sprintf("\n      Session ended.\n\n      Unsaved peer: %s\n      Press [S] to save contact,\n      or any key to return.\n", m.lastPeerName))

	case stateIncoming:
		remoteID := m.incomingStream.Conn().RemotePeer()
		displayName := m.peerDisplayName(remoteID)
		b.WriteString(fmt.Sprintf("\n      Incoming connection :\n      %s\n\n      [Y] accept   [N] reject\n", displayName))

	case stateInCall:
		elapsed := time.Since(m.callStart).Round(time.Second)
		durStr := fmt.Sprintf("%02d:%02d:%02d", int(elapsed.Hours()), int(elapsed.Minutes())%60, int(elapsed.Seconds())%60)
		header := fmt.Sprintf(" CONNECTED: %s", m.peerName)
		b.WriteString("\n" + st.Title.Render(header) + "\n\n")

		muteStatus := st.Online.Render("LIVE")
		if m.muted.Load() {
			muteStatus = st.Offline.Render("MUTED")
		}

		b.WriteString(fmt.Sprintf("      Duration : %s\n", st.Info.Render(durStr)))
		b.WriteString(fmt.Sprintf("      Mic      : %s\n", muteStatus))

		videoStatus := st.Dim.Render("OFF")
		if m.sharingScreen {
			videoStatus = st.Online.Render("SHARING")
		}
		qualityLabel := "Medium"
		switch m.screenQuality {
		case "low":
			qualityLabel = "Low"
		case "high":
			qualityLabel = "High"
		}
		b.WriteString(fmt.Sprintf("      Video    : %s (%s)\n", videoStatus, qualityLabel))
		b.WriteString("      Codec    : Opus (48kHz)\n\n")
		b.WriteString(st.Dim.Render("      Controls : [M] mute/unmute   [V] screen share") + "\n\n")

		b.WriteString(st.Title.Render(" STATS ") + "\n")
		b.WriteString(fmt.Sprintf("      Local Peak : %s\n", st.Info.Render(rmsToDb(m.localRMS))))
		b.WriteString(fmt.Sprintf("      Peer Peak  : %s\n", st.Info.Render(rmsToDb(m.peerRMS))))
		b.WriteString(fmt.Sprintf("      Loss       : %s\n", st.Info.Render(fmt.Sprintf("%.1f%%", m.loss))))
		b.WriteString(fmt.Sprintf("      Latency    : %s\n", st.Info.Render(fmt.Sprintf("%dms", m.latencyMs))))
	}

	return b.String()
}

func (m Model) renderLogs(maxLines int, maxWidth int) string {
	if maxLines <= 0 {
		return ""
	}
	b := strings.Builder{}
	start := 0
	if len(m.logs) > maxLines {
		start = len(m.logs) - maxLines
	}
	contentWidth := maxWidth - 3 // accommodate " > "
	if contentWidth < 1 {
		contentWidth = 1
	}
	prefixStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#D0B000")).Bold(true)
	lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9AA4B2"))
	for i := start; i < len(m.logs); i++ {
		line := m.logs[i]
		runes := []rune(line)
		if len(runes) > contentWidth {
			line = string(runes[:contentWidth])
		}
		b.WriteString(prefixStyle.Render(" > ") + lineStyle.Render(line))
		if i < len(m.logs)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// ── ROOT RENDER METHOD ──────────────────────────────────────────────────
func (m Model) View() string {
	st := m.getStyles()

	// ── Window Sizing Math ───────────────────────────────────────────────
	sidebarWidth := 34

	mainWidth := m.WindowWidth - sidebarWidth - 2 // -2 creates the visual gap
	if mainWidth < 40 {
		mainWidth = 40
	}
	mainInnerWidth := mainWidth - 4 // minus padding/borders

	contentHeight := m.WindowHeight - 4
	if contentHeight < 15 {
		contentHeight = 15
	}
	innerAvailableHeight := contentHeight - 2

	topHeight := contentHeight / 2
	bottomHeight := contentHeight - topHeight

	// ── Build the Left Panes ────────────────────────────────────────────
	var leftPane string
	if m.state == stateInCall {
		info := fmt.Sprintf("\n  %s\n", st.Info.Render(m.peerName))
		info += st.Dim.Render("\n  Navigation disabled\n  during active session.")
		leftPane = m.wrapWithTitle(info, "In Call", sidebarWidth, contentHeight)
	} else {
		contactsPane := m.wrapWithTitle(m.renderContactsPane(), "Contacts", sidebarWidth, topHeight)
		onlinePane := m.wrapWithTitle(m.renderOnlinePane(), "Online", sidebarWidth, bottomHeight)
		leftPane = lipgloss.JoinVertical(lipgloss.Top, contactsPane, onlinePane)
	}

	// ── Build the Main Pane ─────────────────────────────────────────────
	nowStr := time.Now().Format("02 Jan 2006")
	headerText := fmt.Sprintf("%s   %s", config.Get().Username, nowStr)

	mainContentRaw := m.renderMainPane(mainInnerWidth)

	var rightPaneContent string
	if m.debug {
		mainContentHeight := lipgloss.Height(mainContentRaw)
		maxLogLines := innerAvailableHeight - mainContentHeight - 1 // 1 for divider

		if maxLogLines > 0 {
			divider := st.Dim.Render(strings.Repeat("─", mainInnerWidth))
			logsContent := m.renderLogs(maxLogLines, mainInnerWidth)

			logsBlock := divider
			if logsContent != "" {
				logsBlock += "\n" + logsContent
			}
			rightPaneContent = lipgloss.JoinVertical(lipgloss.Top, mainContentRaw, logsBlock)
		} else {
			// Window too short to show debug logs without breaking UI
			rightPaneContent = mainContentRaw
		}
	} else {
		rightPaneContent = mainContentRaw
	}

	mainTitle := fmt.Sprintf("Main [ %s ]", headerText)
	rightPane := m.wrapWithTitle(rightPaneContent, mainTitle, mainWidth, contentHeight)

	// ── Join Everything (with a clean 2-space gap) ──────────────────────
	split := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, "  ", rightPane)

	// Shortened the keys list to prevent terminal width wrapping
	keysStr := "  [up/down] select  [Enter] connect  [P] paste id  [S] set  [Q] quit"
	if m.state == stateInCall {
		keysStr = "  [M] mute  [V] video  [D] debug  [Q] quit"
	} else if m.state == stateBrowsing && m.manualDialMode {
		keysStr = "  [Paste] peer id  [Enter] connect  [Esc] cancel  [Q] quit"
	}

	// Shortened the Peer ID to 8 chars to prevent terminal width wrapping
	ioStr := "System Default  "
	if m.h != nil {
		peerID := m.h.ID().String()
		if len(peerID) > 8 {
			peerID = peerID[len(peerID)-8:]
		}
		ioStr = fmt.Sprintf("Peer ID: %s  ", peerID)
	}

	footerWidth := m.WindowWidth // Use absolute window width for math
	footerStyle := st.Info.Copy().Bold(true)
	keysRendered := footerStyle.Render(keysStr)
	ioRendered := footerStyle.Render(ioStr)

	padLen := footerWidth - lipgloss.Width(keysRendered) - lipgloss.Width(ioRendered)
	if padLen < 0 {
		padLen = 0
	}

	footer := keysRendered + strings.Repeat(" ", padLen) + ioRendered

	return "\n" + split + "\n" + footer
}
