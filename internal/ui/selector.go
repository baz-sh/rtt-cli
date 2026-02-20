package ui

import (
	"fmt"
	"strings"

	"github.com/barryhall/rtt-cli/internal/api"
	"github.com/barryhall/rtt-cli/internal/stations"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

type stationItem struct {
	name string
	code string
}

func (i stationItem) Title() string       { return i.name }
func (i stationItem) Description() string { return i.code }
func (i stationItem) FilterValue() string { return i.name + " " + i.code }

type selectionStep int

const (
	selectingFrom selectionStep = iota
	selectingTo
	searching
	showingResults
)

type SelectorModel struct {
	step        selectionStep
	fromStation *stationItem
	toStation   *stationItem
	list        list.Model
	textInput   textinput.Model
	apiClient   *api.Client
	departures  []api.Departure
	err         error
	width       int
	height      int
}

type searchCompleteMsg struct {
	departures []api.Departure
	err        error
}

func NewSelectorModel(apiClient *api.Client) SelectorModel {
	// Create station items
	items := make([]list.Item, len(stations.Stations))
	for i, station := range stations.Stations {
		items[i] = stationItem{name: station.Name, code: station.Code}
	}

	// Create list
	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.Title = "Select Departure Station"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	// Create text input for search
	ti := textinput.New()
	ti.Placeholder = "Search stations..."
	ti.Focus()

	return SelectorModel{
		step:      selectingFrom,
		list:      l,
		textInput: ti,
		apiClient: apiClient,
	}
}

func (m SelectorModel) Init() tea.Cmd {
	return nil
}

func (m SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-4)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		switch m.step {
		case selectingFrom, selectingTo:
			if msg.String() == "enter" {
				selectedItem := m.list.SelectedItem()
				if selectedItem == nil {
					return m, nil
				}
				station := selectedItem.(stationItem)

				if m.step == selectingFrom {
					m.fromStation = &station
					m.list.Title = "Select Arrival Station"
					m.list.ResetFilter()
					m.step = selectingTo
					return m, nil
				} else {
					m.toStation = &station
					m.step = searching
					return m, m.searchDepartures()
				}
			}

		case showingResults:
			// Allow user to go back
			if msg.String() == "esc" || msg.String() == "q" {
				return m, tea.Quit
			}
		}

	case searchCompleteMsg:
		m.step = showingResults
		m.departures = msg.departures
		m.err = msg.err
		return m, nil
	}

	var cmd tea.Cmd
	if m.step == selectingFrom || m.step == selectingTo {
		m.list, cmd = m.list.Update(msg)
	}

	return m, cmd
}

func (m SelectorModel) View() string {
	switch m.step {
	case selectingFrom, selectingTo:
		return m.list.View()

	case searching:
		theme := CurrentTheme()
		style := lipgloss.NewStyle().
			Foreground(theme.Title).
			Bold(true).
			Padding(1, 0)
		return style.Render(fmt.Sprintf("🚂 Searching for trains from %s to %s...",
			m.fromStation.name, m.toStation.name))

	case showingResults:
		if m.err != nil {
			theme := CurrentTheme()
			errorStyle := lipgloss.NewStyle().
				Foreground(theme.Error).
				Bold(true).
				Padding(1, 0)
			return errorStyle.Render(fmt.Sprintf("Error: %v\n\nPress q to quit", m.err))
		}

		return m.renderDepartures()
	}

	return ""
}

func (m *SelectorModel) searchDepartures() tea.Cmd {
	return func() tea.Msg {
		departures, err := m.apiClient.GetDepartures(m.fromStation.code, m.toStation.code)
		return searchCompleteMsg{departures: departures, err: err}
	}
}

func (m SelectorModel) renderDepartures() string {
	theme := CurrentTheme()

	if len(m.departures) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(theme.Muted).
			Padding(1, 0)
		return emptyStyle.Render("No departures found.\n\nPress q to quit")
	}

	var sb strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Foreground(theme.Title).Bold(true)
	sb.WriteString(titleStyle.Render(fmt.Sprintf("🚂 Trains from %s to %s", m.fromStation.name, m.toStation.name)))
	sb.WriteString("\n\n")

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

	// Create table with lipgloss table package
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

	sb.WriteString(t.String())
	sb.WriteString("\n\n")

	footerStyle := lipgloss.NewStyle().Foreground(theme.Muted)
	sb.WriteString(footerStyle.Render("Press q to quit") + "\n")

	return sb.String()
}

func formatTime(timeStr string) string {
	if len(timeStr) < 4 {
		return timeStr
	}
	return timeStr[0:2] + ":" + timeStr[2:4]
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
