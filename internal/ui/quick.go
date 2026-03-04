package ui

import (
	"fmt"

	"github.com/baz-sh/rtt-cli/internal/api"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

type QuickDisplayModel struct {
	fromName   string
	toName     string
	departures []api.Departure
	viewport   viewport.Model
	ready      bool
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
		if m.ready {
			switch msg.String() {
			case "j":
				m.viewport.ScrollDown(1)
			case "k":
				m.viewport.ScrollUp(1)
			}
		}
	case tea.WindowSizeMsg:
		headerHeight := 3 // title + blank line
		footerHeight := 3 // blank line + footer + blank line
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)
			m.viewport.SetContent(m.renderTable())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - headerHeight - footerHeight
			m.viewport.SetContent(m.renderTable())
		}
		return m, nil
	}

	if m.ready {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m QuickDisplayModel) View() string {
	if !m.ready {
		return ""
	}

	theme := CurrentTheme()

	titleStyle := lipgloss.NewStyle().Foreground(theme.Title).Bold(true)
	title := titleStyle.Render(fmt.Sprintf("🚂 Trains from %s to %s", m.fromName, m.toName))

	footerStyle := lipgloss.NewStyle().Foreground(theme.Muted)
	scrollInfo := fmt.Sprintf("%d%%", int(m.viewport.ScrollPercent()*100))
	footer := footerStyle.Render(fmt.Sprintf("↑/↓ scroll • %s • q to quit", scrollInfo))

	return title + "\n\n" + m.viewport.View() + "\n\n" + footer + "\n"
}

func (m QuickDisplayModel) renderTable() string {
	theme := CurrentTheme()

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

	return t.String()
}
