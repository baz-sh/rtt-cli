package ui

import (
	"fmt"

	"github.com/barryhall/rtt-cli/internal/api"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

type QuickDisplayModel struct {
	fromName   string
	toName     string
	departures []api.Departure
}

func NewQuickDisplayModel(fromName, toName string, departures []api.Departure) QuickDisplayModel {
	return QuickDisplayModel{
		fromName:   fromName,
		toName:     toName,
		departures: departures,
	}
}

func (m QuickDisplayModel) Init() tea.Cmd {
	return nil
}

func (m QuickDisplayModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" || msg.String() == "esc" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m QuickDisplayModel) View() string {
	theme := CurrentTheme()

	// Title
	titleStyle := lipgloss.NewStyle().Foreground(theme.Title).Bold(true)
	title := titleStyle.Render(fmt.Sprintf("🚂 Trains from %s to %s", m.fromName, m.toName))

	// Build table rows
	rows := [][]string{}
	for _, dep := range m.departures {
		formattedTime := formatTime(dep.BookedDepartureTime)
		rows = append(rows, []string{
			formattedTime,
			dep.Leaving,
			dep.DeparturePlatform,
			dep.Platform,
			truncate(dep.Service, 20),
			dep.Duration,
		})
	}

	// Create table
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(theme.Border)).
		Headers("Time", "Leaving", "Dep Plat", "Arr Plat", "Service", "Duration").
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == -1 {
				return lipgloss.NewStyle().
					Foreground(theme.Muted).
					Bold(true).
					Align(lipgloss.Left)
			}
			base := lipgloss.NewStyle().Align(lipgloss.Left)
			switch col {
			case 0: // Time
				return base.Foreground(theme.Time).Bold(true)
			case 1: // Leaving
				return base.Foreground(theme.Leaving)
			case 2: // Dep Platform
				return base.Foreground(theme.DepPlatform)
			case 3: // Arr Platform
				return base.Foreground(theme.ArrPlatform)
			case 4: // Service
				return base.Foreground(theme.Service)
			case 5: // Duration
				return base.Foreground(theme.Duration)
			default:
				return base
			}
		})

	footerStyle := lipgloss.NewStyle().Foreground(theme.Muted)
	footer := footerStyle.Render("Press q to quit")

	return title + "\n\n" + t.String() + "\n\n" + footer + "\n"
}
