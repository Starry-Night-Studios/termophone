package ui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Bg         lipgloss.Color
	Bg2        lipgloss.Color
	Dim        lipgloss.Color
	Text       lipgloss.Color
	Hi         lipgloss.Color
	Border     lipgloss.Color
	Title      lipgloss.Color
	Info       lipgloss.Color
	Meter      lipgloss.Color
	Log        lipgloss.Color
	Help       lipgloss.Color
	Online     lipgloss.Color
	Offline    lipgloss.Color
	Selected   lipgloss.Color
	SelectedFg lipgloss.Color
}

var themeNames = []string{
	"System24",
	"S24 Blue",
	"S24 Red",
	"S24 Green",
	"S24 Amber",
	"Phosphor",
	"Teal Term",
}

var themes = []Theme{
	// 1. System24 (default, grayscale + purple accent)
	{
		Bg:         lipgloss.Color("#1e1e1e"),
		Bg2:        lipgloss.Color("#2a2a2a"),
		Dim:        lipgloss.Color("#4a4a4a"),
		Text:       lipgloss.Color("#c2c2c2"),
		Hi:         lipgloss.Color("#f0f0f0"),
		Border:     lipgloss.Color("#4a4a4a"),
		Title:      lipgloss.Color("#b07cc6"),
		Info:       lipgloss.Color("#b07cc6"),
		Meter:      lipgloss.Color("#b07cc6"),
		Log:        lipgloss.Color("#b07cc6"),
		Help:       lipgloss.Color("#4a4a4a"),
		Online:     lipgloss.Color("#6db88a"),
		Offline:    lipgloss.Color("#4a4a4a"),
		Selected:   lipgloss.Color("#2a2a2a"),
		SelectedFg: lipgloss.Color("#f0f0f0"),
	},
	// 2. S24 Blue
	{
		Bg:         lipgloss.Color("#1e1e1e"),
		Bg2:        lipgloss.Color("#2a2a2a"),
		Dim:        lipgloss.Color("#4a4a4a"),
		Text:       lipgloss.Color("#c2c2c2"),
		Hi:         lipgloss.Color("#f0f0f0"),
		Border:     lipgloss.Color("#7aaac8"),
		Title:      lipgloss.Color("#7aaac8"),
		Info:       lipgloss.Color("#7aaac8"),
		Meter:      lipgloss.Color("#7aaac8"),
		Log:        lipgloss.Color("#7aaac8"),
		Help:       lipgloss.Color("#4a4a4a"),
		Online:     lipgloss.Color("#6db88a"),
		Offline:    lipgloss.Color("#4a4a4a"),
		Selected:   lipgloss.Color("#2a2a2a"),
		SelectedFg: lipgloss.Color("#f0f0f0"),
	},
	// 3. S24 Red
	{
		Bg:         lipgloss.Color("#1e1e1e"),
		Bg2:        lipgloss.Color("#2a2a2a"),
		Dim:        lipgloss.Color("#4a4a4a"),
		Text:       lipgloss.Color("#c2c2c2"),
		Hi:         lipgloss.Color("#f0f0f0"),
		Border:     lipgloss.Color("#c87a7a"),
		Title:      lipgloss.Color("#c87a7a"),
		Info:       lipgloss.Color("#c87a7a"),
		Meter:      lipgloss.Color("#c87a7a"),
		Log:        lipgloss.Color("#c87a7a"),
		Help:       lipgloss.Color("#4a4a4a"),
		Online:     lipgloss.Color("#6db88a"),
		Offline:    lipgloss.Color("#4a4a4a"),
		Selected:   lipgloss.Color("#2a2a2a"),
		SelectedFg: lipgloss.Color("#f0f0f0"),
	},
	// 4. S24 Green
	{
		Bg:         lipgloss.Color("#1e1e1e"),
		Bg2:        lipgloss.Color("#2a2a2a"),
		Dim:        lipgloss.Color("#4a4a4a"),
		Text:       lipgloss.Color("#c2c2c2"),
		Hi:         lipgloss.Color("#f0f0f0"),
		Border:     lipgloss.Color("#6db88a"),
		Title:      lipgloss.Color("#6db88a"),
		Info:       lipgloss.Color("#6db88a"),
		Meter:      lipgloss.Color("#6db88a"),
		Log:        lipgloss.Color("#6db88a"),
		Help:       lipgloss.Color("#4a4a4a"),
		Online:     lipgloss.Color("#6db88a"),
		Offline:    lipgloss.Color("#4a4a4a"),
		Selected:   lipgloss.Color("#2a2a2a"),
		SelectedFg: lipgloss.Color("#f0f0f0"),
	},
	// 5. S24 Amber 
	{
		Bg:         lipgloss.Color("#1c1a14"),
		Bg2:        lipgloss.Color("#262414"),
		Dim:        lipgloss.Color("#5a5440"),
		Text:       lipgloss.Color("#c8bc96"),
		Hi:         lipgloss.Color("#f0e8c0"),
		Border:     lipgloss.Color("#c8b46d"),
		Title:      lipgloss.Color("#c8b46d"),
		Info:       lipgloss.Color("#c8b46d"),
		Meter:      lipgloss.Color("#c8b46d"),
		Log:        lipgloss.Color("#c8b46d"),
		Help:       lipgloss.Color("#5a5440"),
		Online:     lipgloss.Color("#6db88a"),
		Offline:    lipgloss.Color("#5a5440"),
		Selected:   lipgloss.Color("#262414"),
		SelectedFg: lipgloss.Color("#f0e8c0"),
	},
	// 6. Phosphor
	{
		Bg:         lipgloss.Color("#0a0f0a"),
		Bg2:        lipgloss.Color("#0f160f"),
		Dim:        lipgloss.Color("#2a4a2a"),
		Text:       lipgloss.Color("#7abf7a"),
		Hi:         lipgloss.Color("#b0ffb0"),
		Border:     lipgloss.Color("#2a4a2a"),
		Title:      lipgloss.Color("#40ff40"),
		Info:       lipgloss.Color("#40ff40"),
		Meter:      lipgloss.Color("#40ff40"),
		Log:        lipgloss.Color("#40ff40"),
		Help:       lipgloss.Color("#2a4a2a"),
		Online:     lipgloss.Color("#40ff40"),
		Offline:    lipgloss.Color("#2a4a2a"),
		Selected:   lipgloss.Color("#0f160f"),
		SelectedFg: lipgloss.Color("#b0ffb0"),
	},
	// 7. Teal Term
	{
		Bg:         lipgloss.Color("#0d1117"),
		Bg2:        lipgloss.Color("#2a2a2a"),
		Dim:        lipgloss.Color("#3a5060"),
		Text:       lipgloss.Color("#a0c8d0"),
		Hi:         lipgloss.Color("#f0f0f0"),
		Border:     lipgloss.Color("#6dbcb8"),
		Title:      lipgloss.Color("#6dbcb8"),
		Info:       lipgloss.Color("#6dbcb8"),
		Meter:      lipgloss.Color("#6dbcb8"),
		Log:        lipgloss.Color("#6dbcb8"),
		Help:       lipgloss.Color("#3a5060"),
		Online:     lipgloss.Color("#6dbcb8"),
		Offline:    lipgloss.Color("#3a5060"),
		Selected:   lipgloss.Color("#161b22"),
		SelectedFg: lipgloss.Color("#f0f0f0"),
	},
}
