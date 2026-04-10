package ui

import (
	"fmt"
	"math"
	"strings"
	"termophone/config"
	"time"

	"github.com/charmbracelet/lipgloss"
)

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
		Border:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Border),
		Sidebar:  lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, true, false, false).BorderForeground(th.Border).Padding(0, 1),
		Main:     lipgloss.NewStyle().Padding(0, 1),
		Title:    lipgloss.NewStyle().Bold(true).Foreground(th.Title),
		Info:     lipgloss.NewStyle().Foreground(th.Info),
		Meter:    lipgloss.NewStyle().Foreground(th.Meter),
		Log:      lipgloss.NewStyle().Foreground(th.Log),
		Help:     lipgloss.NewStyle().Foreground(th.Help),
		Online:   lipgloss.NewStyle().Foreground(th.Online),
		Offline:  lipgloss.NewStyle().Foreground(th.Offline),
		Dim:      lipgloss.NewStyle().Foreground(th.Dim),
		Selected: lipgloss.NewStyle().Background(th.Border).Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Width(26),
	}
}

func rmsToDb(rms float64) string {
	if rms <= 0 {
		return "-inf dB"
	}
	db := 20 * math.Log10(rms)
	return fmt.Sprintf("%-6.0fdB", db)
}

func (m Model) renderSidebar() string {
	st := m.getStyles()
	b := strings.Builder{}
	b.WriteString(st.Title.Render("TERMOPHONE") + "\n\n")

	if m.state == stateInCall {
		b.WriteString(fmt.Sprintf(" CONNECTED:\n  %s\n", st.Info.Render(m.peerName)))
		b.WriteString(st.Dim.Render("\n  Navigation disabled\n  during active session."))
		return b.String()
	}

	b.WriteString(st.Info.Render(" CONTACTS") + "\n")
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
			rowText := fmt.Sprintf("  %s %s", rawStatus, name)
			// Truncate if too long (26 is the width of our sidebar selection)
			if len(rowText) > 26 {
				rowText = rowText[:26]
			}
			b.WriteString(st.Selected.Render(rowText) + "\n")
		} else {
			b.WriteString(fmt.Sprintf("  %s %s\n", status, name))
		}
	}

	b.WriteString("\n" + st.Info.Render(" NEW PEERS") + "\n")
	offset := len(m.contacts)
	for i, p := range m.newPeers {
		displayName := m.peerDisplayName(p.ID)
		if m.cursor == offset+i {
			rowText := fmt.Sprintf("  %s", displayName)
			if len(rowText) > 26 {
				rowText = rowText[:26]
			}
			b.WriteString(st.Selected.Render(rowText) + "\n")
		} else {
			b.WriteString(fmt.Sprintf("  %s\n", displayName))
		}
	}

	if len(m.contacts) == 0 && len(m.newPeers) == 0 {
		b.WriteString(st.Dim.Render("  (empty)"))
	}

	return b.String()
}

func (m Model) renderMainPane() string {
	st := m.getStyles()
	b := strings.Builder{}

	switch m.state {
	case stateSettings:
		b.WriteString("\n      Settings\n\n")
		scm := themeNames[0]
		if m.colorScheme >= 0 && m.colorScheme < len(themeNames) {
			scm = themeNames[m.colorScheme]
		}

		usrStr := fmt.Sprintf("Username : %s", m.usernameInput.View())
		colStr := fmt.Sprintf("Theme    : < %s >", scm)

		// Create a specific selection style for settings
		settingsSelected := st.Selected.Copy().Width(40).PaddingLeft(6)

		if m.settingsCursor == 0 {
			b.WriteString(settingsSelected.Render(usrStr) + "\n\n")
			b.WriteString("      " + colStr + "\n\n")
		} else {
			b.WriteString("      " + usrStr + "\n\n")
			b.WriteString(settingsSelected.Render(colStr) + "\n\n")
		}

		b.WriteString("\n      [Esc] cancel   [Enter] save\n")

	case stateBrowsing:
		b.WriteString("\n      Not connected.\n\n      Select a peer and press\n      [Enter] to connect.\n")
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
		b.WriteString(fmt.Sprintf("      Codec    : Opus (48kHz)\n\n"))

		b.WriteString(st.Title.Render(" STATS ") + "\n")
		b.WriteString(fmt.Sprintf("      Local Peak : %s\n", st.Info.Render(rmsToDb(m.localRMS))))
		b.WriteString(fmt.Sprintf("      Peer Peak  : %s\n", st.Info.Render(rmsToDb(m.peerRMS))))
		b.WriteString(fmt.Sprintf("      Loss       : %s\n", st.Info.Render(fmt.Sprintf("%.1f%%", m.loss))))
		b.WriteString(fmt.Sprintf("      Latency    : %s\n", st.Info.Render(fmt.Sprintf("%dms", m.latencyMs))))
	}

	return b.String()
}

func (m Model) renderLogs(maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	st := m.getStyles()
	b := strings.Builder{}
	start := 0
	if len(m.logs) > maxLines {
		start = len(m.logs) - maxLines
	}
	for i := start; i < len(m.logs); i++ {
		b.WriteString(st.Log.Render("  > " + m.logs[i]))
		if i < len(m.logs)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (m Model) View() string {
	st := m.getStyles()
	const sidebarWidth = 28
	mainWidth := m.WindowWidth - sidebarWidth - 6
	if mainWidth < 30 {
		mainWidth = 30
	}

	sidebarHeight := m.WindowHeight - 4
	if sidebarHeight < 10 {
		sidebarHeight = 10
	}

	leftPane := st.Sidebar.Width(sidebarWidth).Height(sidebarHeight).Render(m.renderSidebar())

	nowStr := time.Now().Format("02 Jan 2006")
	headerText := fmt.Sprintf("%s   %s", config.Get().Username, nowStr)
	// st.Main has padding of 1 on left and right, so the inner width is mainWidth - 2
	headerRendered := lipgloss.NewStyle().Width(mainWidth - 2).Align(lipgloss.Right).Foreground(st.Info.GetForeground()).Render(headerText)

	mainContent := headerRendered + "\n" + m.renderMainPane()

	var rightPane string
	if m.debug {
		divider := st.Dim.Render(strings.Repeat("─", mainWidth-2))

		// Ensure we don't overflow the UI height by displaying infinite logs
		mainContentHeight := lipgloss.Height(mainContent)
		maxLogLines := sidebarHeight - mainContentHeight - 2 // -2 for the divider and spacer
		if maxLogLines < 0 {
			maxLogLines = 0
		}

		logsContent := m.renderLogs(maxLogLines)
		logsBlock := divider
		if logsContent != "" {
			logsBlock += "\n" + logsContent
		}

		logsHeight := lipgloss.Height(logsBlock)
		topBlockHeight := sidebarHeight - logsHeight
		if topBlockHeight < 0 {
			topBlockHeight = 0
		}

		topPane := lipgloss.NewStyle().Width(mainWidth - 2).Height(topBlockHeight).Render(mainContent)
		rightPane = st.Main.Width(mainWidth).Height(sidebarHeight).Render(
			lipgloss.JoinVertical(lipgloss.Top, topPane, logsBlock),
		)
	} else {
		rightPane = st.Main.Width(mainWidth).Height(sidebarHeight).Render(mainContent)
	}

	split := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	full := st.Border.Render(split)

	keysStr := "  [up/down] navigate  [Enter] connect  [S] settings  [R] reload  [X] remove  [M] mute  [Q] quit"
	ioStr := "System Default  "

	// Create a responsive footer that aligns the IO text to the right
	footerWidth := lipgloss.Width(full)
	keysRendered := st.Help.Render(keysStr)
	ioRendered := st.Help.Render(ioStr)

	padLen := footerWidth - lipgloss.Width(keysRendered) - lipgloss.Width(ioRendered)
	if padLen < 0 {
		padLen = 0
	}

	footer := keysRendered + strings.Repeat(" ", padLen) + ioRendered

	return "\n" + full + "\n" + footer
}
