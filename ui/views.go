package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	borderStyle    = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#8A2BE2"))
	sidebarStyle   = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, true, false, false).BorderForeground(lipgloss.Color("#8A2BE2")).Padding(0, 1)
	mainRightStyle = lipgloss.NewStyle().Padding(0, 1)
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#D2A8FF"))
	infoStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#9D72C3"))
	meterStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#C77DF3"))
	logStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#7B52AB"))
	helpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#5B3285"))
	onlineStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#4CAF50"))
	offlineStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F44336"))
	dimStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
)

func rmsToDb(rms float64) string {
	if rms <= 0 {
		return "-inf dB"
	}
	db := 20 * math.Log10(rms)
	return fmt.Sprintf("%-6.0fdB", db)
}

func (m Model) renderSidebar() string {
	b := strings.Builder{}
	b.WriteString(titleStyle.Render("  termophone") + "\n\n")

	if m.state == stateInCall {
		b.WriteString(fmt.Sprintf(" IN CALL:\n  %s\n", infoStyle.Render(m.peerName)))
		b.WriteString(dimStyle.Render("\n  Navigation disabled\n  during active call."))
		return b.String()
	}

	b.WriteString(infoStyle.Render(" CONTACTS") + "\n")
	for i, c := range m.contacts {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}
		status := offlineStyle.Render("[!]")
		if m.isOnline(c.PeerID) {
			status = onlineStyle.Render("[O]")
		}

		name := c.Name
		if name == "" {
			if len(c.PeerID) > 8 {
				name = c.PeerID[len(c.PeerID)-8:]
			} else {
				name = c.PeerID
			}
		}

		b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, status, name))
	}

	b.WriteString("\n" + infoStyle.Render(" NEW PEERS") + "\n")
	offset := len(m.contacts)
	for i, p := range m.newPeers {
		cursor := "  "
		if m.cursor == offset+i {
			cursor = "> "
		}

		displayName := m.peerDisplayName(p.ID)
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, displayName))
	}

	if len(m.contacts) == 0 && len(m.newPeers) == 0 {
		b.WriteString(dimStyle.Render("  (empty)"))
	}

	return b.String()
}

func (m Model) renderMainPane() string {
	b := strings.Builder{}

	switch m.state {
	case stateBrowsing:
		b.WriteString("\n      ◇ No active call.\n\n      Select a peer and press\n      [Enter] to dial.\n")
		if m.statusMsg != "" {
			b.WriteString(fmt.Sprintf("\n      %s\n", infoStyle.Render(m.statusMsg)))
		}
	case statePostCall:
		b.WriteString(fmt.Sprintf("\n      ◇ Call ended.\n\n      Unsaved peer: %s\n      Press [S] to save contact,\n      or any key to return.\n", m.lastPeerName))
	case stateIncoming:
		remoteID := m.incomingStream.Conn().RemotePeer()
		displayName := m.peerDisplayName(remoteID)
		b.WriteString(fmt.Sprintf("\n      ◇ Incoming call :\n      %s\n\n      [Y] accept   [N] reject\n", displayName))
	case stateInCall:
		elapsed := time.Since(m.callStart).Round(time.Second)
		durStr := fmt.Sprintf("%02d:%02d:%02d", int(elapsed.Hours()), int(elapsed.Minutes())%60, int(elapsed.Seconds())%60)
		header := fmt.Sprintf(" IN CALL: %s", m.peerName)
		b.WriteString("\n" + titleStyle.Render(header) + "\n\n")

		muteStatus := onlineStyle.Render("LIVE")
		if m.muted.Load() {
			muteStatus = offlineStyle.Render("MUTED")
		}

		b.WriteString(fmt.Sprintf("      Duration : %s\n", infoStyle.Render(durStr)))
		b.WriteString(fmt.Sprintf("      Mic      : %s\n", muteStatus))
		b.WriteString(fmt.Sprintf("      Codec    : Opus (48kHz)\n\n"))

		b.WriteString(titleStyle.Render(" STATS ") + "\n")
		b.WriteString(fmt.Sprintf("      Local Peak : %s\n", infoStyle.Render(rmsToDb(m.localRMS))))
		b.WriteString(fmt.Sprintf("      Peer Peak  : %s\n", infoStyle.Render(rmsToDb(m.peerRMS))))
		b.WriteString(fmt.Sprintf("      Loss       : %s\n", infoStyle.Render(fmt.Sprintf("%.1f%%", m.loss))))
		b.WriteString(fmt.Sprintf("      Latency    : %s\n", infoStyle.Render(fmt.Sprintf("%dms", m.latencyMs))))
	}

	return b.String()
}

func (m Model) renderLogs() string {
	b := strings.Builder{}
	for _, l := range m.logs {
		b.WriteString(logStyle.Render("  > "+l) + "\n")
	}
	return b.String()
}

func (m Model) View() string {
	const sidebarWidth = 28
	mainWidth := m.WindowWidth - sidebarWidth - 6
	if mainWidth < 30 {
		mainWidth = 30
	}

	sidebarHeight := m.WindowHeight - 4
	if sidebarHeight < 10 {
		sidebarHeight = 10
	}

	leftPane := sidebarStyle.Width(sidebarWidth).Height(sidebarHeight).Render(m.renderSidebar())

	mainContent := m.renderMainPane()

	if m.debug {
		mainContent += "\n\n" + strings.Repeat("-", mainWidth) + "\n" + m.renderLogs()
	}

	rightPane := mainRightStyle.Width(mainWidth).Height(sidebarHeight).Render(mainContent)

	split := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	full := borderStyle.Render(split)

	footer := helpStyle.Render("  [up/down] navigate  [Enter] dial  [R] reload  [X] remove  [M] mute  [D] debug  [Q] quit")

	return "\n" + full + "\n" + footer
}
