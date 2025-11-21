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
	// Title
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
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
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("238"))).
		Headers("Time", "Leaving", "Dep Plat", "Arr Plat", "Service", "Duration").
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == -1 {
				return lipgloss.NewStyle().
					Foreground(lipgloss.Color("241")).
					Bold(true).
					Align(lipgloss.Left)
			}
			base := lipgloss.NewStyle().Align(lipgloss.Left)
			switch col {
			case 0: // Time
				return base.Foreground(lipgloss.Color("212")).Bold(true)
			case 1: // Leaving
				return base.Foreground(lipgloss.Color("214"))
			case 2: // Dep Platform
				return base.Foreground(lipgloss.Color("196"))
			case 3: // Arr Platform
				return base.Foreground(lipgloss.Color("46"))
			case 4: // Service
				return base.Foreground(lipgloss.Color("201"))
			case 5: // Duration
				return base.Foreground(lipgloss.Color("141"))
			default:
				return base
			}
		})

	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	footer := footerStyle.Render("Press q to quit")

	return title + "\n\n" + t.String() + "\n\n" + footer + "\n"
}
