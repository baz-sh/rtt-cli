package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme holds all color definitions used throughout the UI.
type Theme struct {
	Title       color.Color
	Error       color.Color
	Muted       color.Color
	Border      color.Color
	Time        color.Color
	Leaving     color.Color
	DepPlatform color.Color
	ArrPlatform color.Color
	Service     color.Color
	Duration    color.Color
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

// currentTheme defaults to dark, updated when Bubble Tea detects background color.
var currentTheme = darkTheme

// SetDarkMode updates the theme based on terminal background.
// Called from Bubble Tea models when they receive tea.BackgroundColorMsg.
func SetDarkMode(isDark bool) {
	if isDark {
		currentTheme = darkTheme
	} else {
		currentTheme = lightTheme
	}
}

// CurrentTheme returns the active theme.
func CurrentTheme() Theme {
	return currentTheme
}
