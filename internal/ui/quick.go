package ui

import (
	"fmt"

	"github.com/baz-sh/rtt-cli/internal/api"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

type QuickDisplayModel struct {
	fromName   string
	fromCode   string
	toName     string
	toCode     string
	apiClient  *api.Client
	departures []api.Departure
	spinner    spinner.Model
	viewport   viewport.Model
	loading    bool
	ready      bool
	err        error
}

type quickSearchCompleteMsg struct {
	departures []api.Departure
	err        error
}

func NewQuickDisplayModel(apiClient *api.Client, fromCode, toCode, fromName, toName string) QuickDisplayModel {
	s := spinner.New(
		spinner.WithSpinner(spinner.Points),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("205"))),
	)
	return QuickDisplayModel{
		fromName:  fromName,
		fromCode:  fromCode,
		toName:    toName,
		toCode:    toCode,
		apiClient: apiClient,
		spinner:   s,
		loading:   true,
	}
}

func (m QuickDisplayModel) Init() tea.Cmd {
	return tea.Batch(tea.RequestBackgroundColor, m.spinner.Tick, m.fetchDepartures())
}

func (m QuickDisplayModel) fetchDepartures() tea.Cmd {
	return func() tea.Msg {
		departures, err := m.apiClient.GetDepartures(m.fromCode, m.toCode)
		return quickSearchCompleteMsg{departures: departures, err: err}
	}
}

func (m QuickDisplayModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		SetDarkMode(msg.IsDark())
		if m.ready {
			m.viewport.SetContent(m.renderTable())
		}
		return m, nil

	case tea.KeyPressMsg:
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

	case quickSearchCompleteMsg:
		m.loading = false
		m.departures = msg.departures
		m.err = msg.err
		if m.ready {
			m.viewport.SetContent(m.renderTable())
		}
		return m, nil

	case tea.WindowSizeMsg:
		headerHeight := 3 // title + blank line
		footerHeight := 3 // blank line + footer + blank line
		if !m.ready {
			m.viewport = viewport.New(viewport.WithWidth(msg.Width), viewport.WithHeight(msg.Height-headerHeight-footerHeight))
			m.viewport.SetContent(m.renderTable())
			m.ready = true
		} else {
			m.viewport.SetWidth(msg.Width)
			m.viewport.SetHeight(msg.Height - headerHeight - footerHeight)
			m.viewport.SetContent(m.renderTable())
		}
		return m, nil
	}

	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	if m.ready {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m QuickDisplayModel) View() tea.View {
	v := tea.NewView("")
	theme := CurrentTheme()

	if m.loading {
		style := lipgloss.NewStyle().
			Foreground(theme.Title).
			Bold(true).
			Padding(1, 0)
		v = tea.NewView(style.Render(fmt.Sprintf("%s Searching for trains from %s to %s...",
			m.spinner.View(), m.fromName, m.toName)))
		v.AltScreen = true
		return v
	}

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(theme.Error).
			Bold(true).
			Padding(1, 0)
		v = tea.NewView(errorStyle.Render(fmt.Sprintf("Error: %v\n\nPress q to quit", m.err)))
		v.AltScreen = true
		return v
	}

	if !m.ready || len(m.departures) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(theme.Muted).
			Padding(1, 0)
		v = tea.NewView(emptyStyle.Render("No departures found.\n\nPress q to quit"))
		v.AltScreen = true
		return v
	}

	titleStyle := lipgloss.NewStyle().Foreground(theme.Title).Bold(true)
	title := titleStyle.Render(fmt.Sprintf("Trains from %s to %s", m.fromName, m.toName))

	footerStyle := lipgloss.NewStyle().Foreground(theme.Muted)
	scrollInfo := fmt.Sprintf("%d%%", int(m.viewport.ScrollPercent()*100))
	footer := footerStyle.Render(fmt.Sprintf("↑/↓ scroll • %s • q to quit", scrollInfo))

	v = tea.NewView(title + "\n\n" + m.viewport.View() + "\n\n" + footer + "\n")
	v.AltScreen = true
	return v
}

func (m QuickDisplayModel) renderTable() string {
	theme := CurrentTheme()

	rows := [][]string{}
	for _, dep := range m.departures {
		rows = append(rows, []string{
			dep.BookedDepartureTime,
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
