package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Theme holds all color definitions used throughout the UI.
type Theme struct {
	Title       lipgloss.Color
	Error       lipgloss.Color
	Muted       lipgloss.Color
	Border      lipgloss.Color
	Time        lipgloss.Color
	Leaving     lipgloss.Color
	DepPlatform lipgloss.Color
	ArrPlatform lipgloss.Color
	Service     lipgloss.Color
	Duration    lipgloss.Color
}

// Dark theme - bright/saturated colors for dark terminal backgrounds.
var darkTheme = Theme{
	Title:       lipgloss.Color("205"), // hot pink
	Error:       lipgloss.Color("196"), // bright red
	Muted:       lipgloss.Color("241"), // medium gray
	Border:      lipgloss.Color("238"), // dark gray
	Time:        lipgloss.Color("212"), // light pink
	Leaving:     lipgloss.Color("214"), // orange
	DepPlatform: lipgloss.Color("196"), // bright red
	ArrPlatform: lipgloss.Color("46"),  // bright green
	Service:     lipgloss.Color("201"), // bright magenta
	Duration:    lipgloss.Color("141"), // lavender
}

// Light theme - deeper/darker colors for light terminal backgrounds.
var lightTheme = Theme{
	Title:       lipgloss.Color("125"), // dark magenta
	Error:       lipgloss.Color("160"), // dark red
	Muted:       lipgloss.Color("244"), // medium gray (lighter for contrast)
	Border:      lipgloss.Color("250"), // light gray
	Time:        lipgloss.Color("127"), // dark pink/purple
	Leaving:     lipgloss.Color("172"), // dark orange
	DepPlatform: lipgloss.Color("160"), // dark red
	ArrPlatform: lipgloss.Color("28"),  // dark green
	Service:     lipgloss.Color("90"),  // dark magenta
	Duration:    lipgloss.Color("61"),  // dark lavender/slate blue
}

// currentTheme holds the active theme, set once at init.
var currentTheme Theme

func init() {
	currentTheme = detectTheme()
}

// detectTheme selects the appropriate color theme based on the terminal background.
func detectTheme() Theme {
	if termenv.HasDarkBackground() {
		return darkTheme
	}
	return lightTheme
}

// CurrentTheme returns the active theme.
func CurrentTheme() Theme {
	return currentTheme
}
