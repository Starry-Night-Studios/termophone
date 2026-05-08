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
	"Muted Blue",
	"Muted Green",
	"Muted Red",
	"Muted Gold",
	"Muted Purple",
	"One Dark",
	"Dracula",
	"Kanagawa",
	"Catppuccin",
	"Nord",
}

var themes = []Theme{
	// 1. Muted Blue (Slate/Navy)
	{
		Bg:         lipgloss.Color("#111827"),
		Bg2:        lipgloss.Color("#1F2937"),
		Dim:        lipgloss.Color("#6B7280"), // Highly readable gray
		Text:       lipgloss.Color("#D1D5DB"),
		Hi:         lipgloss.Color("#F9FAFB"),
		Border:     lipgloss.Color("#6B7280"),
		Title:      lipgloss.Color("#7CA1D2"), // Muted Blue Accent
		Info:       lipgloss.Color("#7CA1D2"),
		Meter:      lipgloss.Color("#7CA1D2"),
		Log:        lipgloss.Color("#9CA3AF"),
		Help:       lipgloss.Color("#6B7280"),
		Online:     lipgloss.Color("#86B38A"),
		Offline:    lipgloss.Color("#C77D7D"),
		Selected:   lipgloss.Color("#374151"),
		SelectedFg: lipgloss.Color("#F9FAFB"),
	},
	// 2. Muted Green (Forest/Sage)
	{
		Bg:         lipgloss.Color("#141A14"),
		Bg2:        lipgloss.Color("#1E261E"),
		Dim:        lipgloss.Color("#6B7A6B"),
		Text:       lipgloss.Color("#D1DBD1"),
		Hi:         lipgloss.Color("#F4FBF4"),
		Border:     lipgloss.Color("#6B7A6B"),
		Title:      lipgloss.Color("#84A98C"), // Muted Green Accent
		Info:       lipgloss.Color("#84A98C"),
		Meter:      lipgloss.Color("#84A98C"),
		Log:        lipgloss.Color("#A0AFA0"),
		Help:       lipgloss.Color("#6B7A6B"),
		Online:     lipgloss.Color("#84A98C"),
		Offline:    lipgloss.Color("#C77D7D"),
		Selected:   lipgloss.Color("#2D382D"),
		SelectedFg: lipgloss.Color("#F4FBF4"),
	},
	// 3. Muted Red (Brick/Rose)
	{
		Bg:         lipgloss.Color("#1A1414"),
		Bg2:        lipgloss.Color("#261E1E"),
		Dim:        lipgloss.Color("#806363"),
		Text:       lipgloss.Color("#E0D4D4"),
		Hi:         lipgloss.Color("#FFF2F2"),
		Border:     lipgloss.Color("#806363"),
		Title:      lipgloss.Color("#CD7B7B"), // Muted Red Accent
		Info:       lipgloss.Color("#CD7B7B"),
		Meter:      lipgloss.Color("#CD7B7B"),
		Log:        lipgloss.Color("#B59A9A"),
		Help:       lipgloss.Color("#806363"),
		Online:     lipgloss.Color("#86B38A"),
		Offline:    lipgloss.Color("#CD7B7B"),
		Selected:   lipgloss.Color("#382A2A"),
		SelectedFg: lipgloss.Color("#FFF2F2"),
	},
	// 4. Muted Gold (Ochre/Sand)
	{
		Bg:         lipgloss.Color("#1A1814"),
		Bg2:        lipgloss.Color("#26231E"),
		Dim:        lipgloss.Color("#807863"),
		Text:       lipgloss.Color("#E0DCD4"),
		Hi:         lipgloss.Color("#FFFBF2"),
		Border:     lipgloss.Color("#807863"),
		Title:      lipgloss.Color("#D2B471"), // Muted Gold Accent
		Info:       lipgloss.Color("#D2B471"),
		Meter:      lipgloss.Color("#D2B471"),
		Log:        lipgloss.Color("#B5AFA0"),
		Help:       lipgloss.Color("#807863"),
		Online:     lipgloss.Color("#86B38A"),
		Offline:    lipgloss.Color("#C77D7D"),
		Selected:   lipgloss.Color("#38332A"),
		SelectedFg: lipgloss.Color("#FFFBF2"),
	},
	// 5. Muted Purple (Plum/Lavender)
	{
		Bg:         lipgloss.Color("#17141A"),
		Bg2:        lipgloss.Color("#221E26"),
		Dim:        lipgloss.Color("#746380"),
		Text:       lipgloss.Color("#DBD4E0"),
		Hi:         lipgloss.Color("#F8F2FF"),
		Border:     lipgloss.Color("#746380"),
		Title:      lipgloss.Color("#A889C2"), // Muted Purple Accent
		Info:       lipgloss.Color("#A889C2"),
		Meter:      lipgloss.Color("#A889C2"),
		Log:        lipgloss.Color("#AEA0B5"),
		Help:       lipgloss.Color("#746380"),
		Online:     lipgloss.Color("#86B38A"),
		Offline:    lipgloss.Color("#C77D7D"),
		Selected:   lipgloss.Color("#322A38"),
		SelectedFg: lipgloss.Color("#F8F2FF"),
	},
	// 6. One Dark
	{
		Bg:         lipgloss.Color("#282C34"),
		Bg2:        lipgloss.Color("#3E4452"),
		Dim:        lipgloss.Color("#7F848E"), // Brightened from standard comment gray for border readability
		Text:       lipgloss.Color("#ABB2BF"),
		Hi:         lipgloss.Color("#FFFFFF"),
		Border:     lipgloss.Color("#7F848E"),
		Title:      lipgloss.Color("#61AFEF"), // Blue
		Info:       lipgloss.Color("#56B6C2"), // Cyan
		Meter:      lipgloss.Color("#C678DD"),
		Log:        lipgloss.Color("#ABB2BF"),
		Help:       lipgloss.Color("#7F848E"),
		Online:     lipgloss.Color("#98C379"),
		Offline:    lipgloss.Color("#E06C75"),
		Selected:   lipgloss.Color("#3E4452"),
		SelectedFg: lipgloss.Color("#FFFFFF"),
	},
	// 7. Dracula
	{
		Bg:         lipgloss.Color("#282A36"),
		Bg2:        lipgloss.Color("#44475A"),
		Dim:        lipgloss.Color("#7483B5"), // Brightened comment color for crisp borders
		Text:       lipgloss.Color("#F8F8F2"),
		Hi:         lipgloss.Color("#FFFFFF"),
		Border:     lipgloss.Color("#7483B5"),
		Title:      lipgloss.Color("#BD93F9"), // Purple
		Info:       lipgloss.Color("#8BE9FD"), // Cyan
		Meter:      lipgloss.Color("#FF79C6"),
		Log:        lipgloss.Color("#F8F8F2"),
		Help:       lipgloss.Color("#7483B5"),
		Online:     lipgloss.Color("#50FA7B"),
		Offline:    lipgloss.Color("#FF5555"),
		Selected:   lipgloss.Color("#44475A"),
		SelectedFg: lipgloss.Color("#FFFFFF"),
	},
	// 8. Kanagawa
	{
		Bg:         lipgloss.Color("#1F1F28"),
		Bg2:        lipgloss.Color("#2A2A37"),
		Dim:        lipgloss.Color("#7E7D73"), // Fujinami Spring
		Text:       lipgloss.Color("#DCD7BA"),
		Hi:         lipgloss.Color("#C8C093"),
		Border:     lipgloss.Color("#7E7D73"),
		Title:      lipgloss.Color("#7E9CD8"), // Crystal Blue
		Info:       lipgloss.Color("#957FB8"), // Oni Violet
		Meter:      lipgloss.Color("#FFA066"),
		Log:        lipgloss.Color("#DCD7BA"),
		Help:       lipgloss.Color("#7E7D73"),
		Online:     lipgloss.Color("#76946A"),
		Offline:    lipgloss.Color("#C34043"),
		Selected:   lipgloss.Color("#2A2A37"),
		SelectedFg: lipgloss.Color("#C8C093"),
	},
	// 9. Catppuccin Mocha
	{
		Bg:         lipgloss.Color("#1E1E2E"),
		Bg2:        lipgloss.Color("#313244"),
		Dim:        lipgloss.Color("#7F849C"), // Overlay 1 (highly readable dim)
		Text:       lipgloss.Color("#CDD6F4"),
		Hi:         lipgloss.Color("#B4BEFE"),
		Border:     lipgloss.Color("#7F849C"),
		Title:      lipgloss.Color("#CBA6F7"), // Mauve
		Info:       lipgloss.Color("#89B4FA"), // Blue
		Meter:      lipgloss.Color("#F38BA8"),
		Log:        lipgloss.Color("#CDD6F4"),
		Help:       lipgloss.Color("#7F849C"),
		Online:     lipgloss.Color("#A6E3A1"),
		Offline:    lipgloss.Color("#F38BA8"),
		Selected:   lipgloss.Color("#45475A"),
		SelectedFg: lipgloss.Color("#CDD6F4"),
	},
	// 10. Nord
	{
		Bg:         lipgloss.Color("#2E3440"),
		Bg2:        lipgloss.Color("#3B4252"),
		Dim:        lipgloss.Color("#616E88"), // Blended Polar Night for crisp unselected frames
		Text:       lipgloss.Color("#D8DEE9"),
		Hi:         lipgloss.Color("#ECEFF4"),
		Border:     lipgloss.Color("#616E88"),
		Title:      lipgloss.Color("#88C0D0"), // Frost
		Info:       lipgloss.Color("#81A1C1"),
		Meter:      lipgloss.Color("#B48EAD"),
		Log:        lipgloss.Color("#E5E9F0"),
		Help:       lipgloss.Color("#616E88"),
		Online:     lipgloss.Color("#A3BE8C"),
		Offline:    lipgloss.Color("#BF616A"),
		Selected:   lipgloss.Color("#434C5E"),
		SelectedFg: lipgloss.Color("#ECEFF4"),
	},
}