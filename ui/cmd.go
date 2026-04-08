package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg {
		return MsgTick(t)
	})
}
