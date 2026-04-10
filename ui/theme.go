package ui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Border  lipgloss.Color
	Title   lipgloss.Color
	Info    lipgloss.Color
	Meter   lipgloss.Color
	Log     lipgloss.Color
	Help    lipgloss.Color
	Online  lipgloss.Color
	Offline lipgloss.Color
	Dim     lipgloss.Color
}

var themeNames = []string{
	"Lavender", "Sage", "Steel", "Dust", "Rose", "Peach",
	"Neon Purple", "Neon Green", "Neon Blue", "Neon Red",
	"Monochrome", "High Contrast","Ocean", "Ember", "Dusk", "Moss", "Copper",
	"Arctic", "Terminal", "Dracula", "Solarized", "Candy",
}

var themes = []Theme{
	// 0: Lavender (Muted Purple)
	{
		Border:  lipgloss.Color("#9b7bb0"),
		Title:   lipgloss.Color("#bba8cd"),
		Info:    lipgloss.Color("#a794b9"),
		Meter:   lipgloss.Color("#c7b6d9"),
		Log:     lipgloss.Color("#8c799c"),
		Help:    lipgloss.Color("#6f5a7e"),
		Online:  lipgloss.Color("#7ab087"),
		Offline: lipgloss.Color("#b87474"),
		Dim:     lipgloss.Color("#5a5a5a"),
	},
	// 1: Sage (Muted Green)
	{
		Border:  lipgloss.Color("#7fb08c"),
		Title:   lipgloss.Color("#a8cdb6"),
		Info:    lipgloss.Color("#94b9a4"),
		Meter:   lipgloss.Color("#b6d9c6"),
		Log:     lipgloss.Color("#799c87"),
		Help:    lipgloss.Color("#5a7e68"),
		Online:  lipgloss.Color("#7fb08c"),
		Offline: lipgloss.Color("#b87474"),
		Dim:     lipgloss.Color("#5a5a5a"),
	},
	// 2: Steel (Muted Blue)
	{
		Border:  lipgloss.Color("#7894b8"),
		Title:   lipgloss.Color("#a3c1e0"),
		Info:    lipgloss.Color("#89a5c4"),
		Meter:   lipgloss.Color("#b1cced"),
		Log:     lipgloss.Color("#6b85a3"),
		Help:    lipgloss.Color("#4c6685"),
		Online:  lipgloss.Color("#7ab087"),
		Offline: lipgloss.Color("#b87474"),
		Dim:     lipgloss.Color("#5a5a5a"),
	},
	// 3: Dust (Muted Grey/Cyan)
	{
		Border:  lipgloss.Color("#889c9c"),
		Title:   lipgloss.Color("#b3c3c3"),
		Info:    lipgloss.Color("#98abab"),
		Meter:   lipgloss.Color("#bfd0d0"),
		Log:     lipgloss.Color("#768a8a"),
		Help:    lipgloss.Color("#5b6f6f"),
		Online:  lipgloss.Color("#7ab087"),
		Offline: lipgloss.Color("#b87474"),
		Dim:     lipgloss.Color("#5a5a5a"),
	},
	// 4: Rose (Muted Pink/Red)
	{
		Border:  lipgloss.Color("#bd818f"),
		Title:   lipgloss.Color("#dbaabb"),
		Info:    lipgloss.Color("#c494a0"),
		Meter:   lipgloss.Color("#deb6c3"),
		Log:     lipgloss.Color("#a36e7a"),
		Help:    lipgloss.Color("#85525d"),
		Online:  lipgloss.Color("#7ab087"),
		Offline: lipgloss.Color("#c96969"),
		Dim:     lipgloss.Color("#5a5a5a"),
	},
	// 5: Peach (Muted Orange/Pink)
	{
		Border:  lipgloss.Color("#c79a83"),
		Title:   lipgloss.Color("#dec1b0"),
		Info:    lipgloss.Color("#cca897"),
		Meter:   lipgloss.Color("#e6d1c5"),
		Log:     lipgloss.Color("#b3836d"),
		Help:    lipgloss.Color("#8a5d48"),
		Online:  lipgloss.Color("#7ab087"),
		Offline: lipgloss.Color("#c96969"),
		Dim:     lipgloss.Color("#5a5a5a"),
	},
	// 6: Neon Purple
	{
		Border:  lipgloss.Color("#8A2BE2"),
		Title:   lipgloss.Color("#D2A8FF"),
		Info:    lipgloss.Color("#9D72C3"),
		Meter:   lipgloss.Color("#C77DF3"),
		Log:     lipgloss.Color("#7B52AB"),
		Help:    lipgloss.Color("#5B3285"),
		Online:  lipgloss.Color("#4CAF50"),
		Offline: lipgloss.Color("#F44336"),
		Dim:     lipgloss.Color("#555555"),
	},
	// 7: Neon Green
	{
		Border:  lipgloss.Color("#00FF00"),
		Title:   lipgloss.Color("#A8FFD2"),
		Info:    lipgloss.Color("#72C39D"),
		Meter:   lipgloss.Color("#7DF3C7"),
		Log:     lipgloss.Color("#52AB7B"),
		Help:    lipgloss.Color("#32855B"),
		Online:  lipgloss.Color("#00FF00"),
		Offline: lipgloss.Color("#FF0000"),
		Dim:     lipgloss.Color("#555555"),
	},
	// 8: Neon Blue
	{
		Border:  lipgloss.Color("#0000FF"),
		Title:   lipgloss.Color("#A8D2FF"),
		Info:    lipgloss.Color("#729DC3"),
		Meter:   lipgloss.Color("#7DC7F3"),
		Log:     lipgloss.Color("#527BAB"),
		Help:    lipgloss.Color("#325B85"),
		Online:  lipgloss.Color("#4CAF50"),
		Offline: lipgloss.Color("#F44336"),
		Dim:     lipgloss.Color("#555555"),
	},
	// 9: Neon Red
	{
		Border:  lipgloss.Color("#FF0000"),
		Title:   lipgloss.Color("#FFA8A8"),
		Info:    lipgloss.Color("#C37272"),
		Meter:   lipgloss.Color("#F37D7D"),
		Log:     lipgloss.Color("#AB5252"),
		Help:    lipgloss.Color("#853232"),
		Online:  lipgloss.Color("#4CAF50"),
		Offline: lipgloss.Color("#FF0000"),
		Dim:     lipgloss.Color("#555555"),
	},
	// 10: Monochrome
	{
		Border:  lipgloss.Color("#CCCCCC"),
		Title:   lipgloss.Color("#FFFFFF"),
		Info:    lipgloss.Color("#AAAAAA"),
		Meter:   lipgloss.Color("#DDDDDD"),
		Log:     lipgloss.Color("#888888"),
		Help:    lipgloss.Color("#666666"),
		Online:  lipgloss.Color("#AAAAAA"),
		Offline: lipgloss.Color("#666666"),
		Dim:     lipgloss.Color("#444444"),
	},
	// 11: High Contrast
	{
		Border:  lipgloss.Color("#FFFFFF"),
		Title:   lipgloss.Color("#FFFFFF"),
		Info:    lipgloss.Color("#FFFFFF"),
		Meter:   lipgloss.Color("#FFFFFF"),
		Log:     lipgloss.Color("#FFFFFF"),
		Help:    lipgloss.Color("#FFFFFF"),
		Online:  lipgloss.Color("#FFFFFF"),
		Offline: lipgloss.Color("#FFFFFF"),
		Dim:     lipgloss.Color("#FFFFFF"),
	},
	// 12: Ocean (Deep Blue/Teal)
	{
		Border:  lipgloss.Color("#4a9aba"),
		Title:   lipgloss.Color("#7dcce0"),
		Info:    lipgloss.Color("#5db0c8"),
		Meter:   lipgloss.Color("#9dd8e8"),
		Log:     lipgloss.Color("#3a7a94"),
		Help:    lipgloss.Color("#2a5a6e"),
		Online:  lipgloss.Color("#5dbf8a"),
		Offline: lipgloss.Color("#c96969"),
		Dim:     lipgloss.Color("#5a5a5a"),
	},
	// 13: Ember (Dark Orange/Amber)
	{
		Border:  lipgloss.Color("#c87941"),
		Title:   lipgloss.Color("#e8b87a"),
		Info:    lipgloss.Color("#d4955a"),
		Meter:   lipgloss.Color("#f0ce9a"),
		Log:     lipgloss.Color("#a85f2e"),
		Help:    lipgloss.Color("#7a4018"),
		Online:  lipgloss.Color("#7ab087"),
		Offline: lipgloss.Color("#c96969"),
		Dim:     lipgloss.Color("#5a5a5a"),
	},
	// 14: Dusk (Purple/Blue twilight)
	{
		Border:  lipgloss.Color("#6a6aab"),
		Title:   lipgloss.Color("#a0a0d8"),
		Info:    lipgloss.Color("#8080bc"),
		Meter:   lipgloss.Color("#b8b8e8"),
		Log:     lipgloss.Color("#505090"),
		Help:    lipgloss.Color("#383878"),
		Online:  lipgloss.Color("#7ab087"),
		Offline: lipgloss.Color("#b87474"),
		Dim:     lipgloss.Color("#5a5a5a"),
	},
	// 15: Moss (Dark earthy green)
	{
		Border:  lipgloss.Color("#6b8c5a"),
		Title:   lipgloss.Color("#9ab88a"),
		Info:    lipgloss.Color("#7da068"),
		Meter:   lipgloss.Color("#b2cc9a"),
		Log:     lipgloss.Color("#526b42"),
		Help:    lipgloss.Color("#3a4e2e"),
		Online:  lipgloss.Color("#a0c878"),
		Offline: lipgloss.Color("#c96969"),
		Dim:     lipgloss.Color("#5a5a5a"),
	},
	// 16: Copper (Warm metallic)
	{
		Border:  lipgloss.Color("#b87333"),
		Title:   lipgloss.Color("#d4a55a"),
		Info:    lipgloss.Color("#c48840"),
		Meter:   lipgloss.Color("#e8c080"),
		Log:     lipgloss.Color("#8a5520"),
		Help:    lipgloss.Color("#5e3510"),
		Online:  lipgloss.Color("#7ab087"),
		Offline: lipgloss.Color("#c96969"),
		Dim:     lipgloss.Color("#5a5a5a"),
	},
	// 17: Arctic (Icy blue/white)
	{
		Border:  lipgloss.Color("#90c8d8"),
		Title:   lipgloss.Color("#c8e8f0"),
		Info:    lipgloss.Color("#a8d8e8"),
		Meter:   lipgloss.Color("#d8f0f8"),
		Log:     lipgloss.Color("#6aacbc"),
		Help:    lipgloss.Color("#4a8898"),
		Online:  lipgloss.Color("#78cc9a"),
		Offline: lipgloss.Color("#c87878"),
		Dim:     lipgloss.Color("#606060"),
	},
	// 18: Terminal (Classic green-on-black CRT)
	{
		Border:  lipgloss.Color("#00aa00"),
		Title:   lipgloss.Color("#00ff00"),
		Info:    lipgloss.Color("#00cc00"),
		Meter:   lipgloss.Color("#00dd00"),
		Log:     lipgloss.Color("#008800"),
		Help:    lipgloss.Color("#006600"),
		Online:  lipgloss.Color("#00ff00"),
		Offline: lipgloss.Color("#aa0000"),
		Dim:     lipgloss.Color("#004400"),
	},
	// 19: Dracula (Popular dark theme)
	{
		Border:  lipgloss.Color("#bd93f9"),
		Title:   lipgloss.Color("#f8f8f2"),
		Info:    lipgloss.Color("#8be9fd"),
		Meter:   lipgloss.Color("#ff79c6"),
		Log:     lipgloss.Color("#6272a4"),
		Help:    lipgloss.Color("#44475a"),
		Online:  lipgloss.Color("#50fa7b"),
		Offline: lipgloss.Color("#ff5555"),
		Dim:     lipgloss.Color("#44475a"),
	},
	// 20: Solarized (Warm dark)
	{
		Border:  lipgloss.Color("#268bd2"),
		Title:   lipgloss.Color("#2aa198"),
		Info:    lipgloss.Color("#859900"),
		Meter:   lipgloss.Color("#b58900"),
		Log:     lipgloss.Color("#657b83"),
		Help:    lipgloss.Color("#586e75"),
		Online:  lipgloss.Color("#859900"),
		Offline: lipgloss.Color("#dc322f"),
		Dim:     lipgloss.Color("#073642"),
	},
	// 21: Candy (Bright pastels)
	{
		Border:  lipgloss.Color("#ff8fab"),
		Title:   lipgloss.Color("#ffc8dd"),
		Info:    lipgloss.Color("#ffafcc"),
		Meter:   lipgloss.Color("#bde0fe"),
		Log:     lipgloss.Color("#a2d2ff"),
		Help:    lipgloss.Color("#cdb4db"),
		Online:  lipgloss.Color("#a8dadc"),
		Offline: lipgloss.Color("#e76f51"),
		Dim:     lipgloss.Color("#606060"),
	},
}

