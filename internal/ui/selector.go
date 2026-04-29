package ui

import (
	"fmt"
	"io"

	"github.com/baz-sh/rtt-cli/internal/api"
	"github.com/baz-sh/rtt-cli/internal/stations"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/x/ansi"
)

type stationItem struct {
	name string
	code string
}

func (i stationItem) Title() string       { return i.name }
func (i stationItem) Description() string { return i.code }
func (i stationItem) FilterValue() string { return i.name + " " + i.code }

// stationDelegate renders station items as compact single-line entries.
type stationDelegate struct {
	normalStyle   lipgloss.Style
	selectedStyle lipgloss.Style
	codeStyle     lipgloss.Style
	dimmedStyle   lipgloss.Style
	filterMatch   lipgloss.Style
}

func newStationDelegate(isDark bool) stationDelegate {
	ld := lipgloss.LightDark(isDark)

	return stationDelegate{
		normalStyle: lipgloss.NewStyle().
			Foreground(ld(lipgloss.Color("#1a1a1a"), lipgloss.Color("#dddddd"))).
			PaddingLeft(2),
		selectedStyle: lipgloss.NewStyle().
			Foreground(ld(lipgloss.Color("#EE6FF8"), lipgloss.Color("#EE6FF8"))).
			Bold(true).
			PaddingLeft(1).
			SetString("▸ "),
		codeStyle: lipgloss.NewStyle().
			Foreground(ld(lipgloss.Color("#9B9B9B"), lipgloss.Color("#5C5C5C"))),
		dimmedStyle: lipgloss.NewStyle().
			Foreground(ld(lipgloss.Color("#A49FA5"), lipgloss.Color("#777777"))).
			PaddingLeft(2),
		filterMatch: lipgloss.NewStyle().Underline(true),
	}
}

func (d stationDelegate) Height() int                             { return 1 }
func (d stationDelegate) Spacing() int                            { return 0 }
func (d stationDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d stationDelegate) ShortHelp() []key.Binding                { return nil }
func (d stationDelegate) FullHelp() [][]key.Binding               { return nil }

func (d stationDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	si, ok := item.(stationItem)
	if !ok || m.Width() <= 0 {
		return
	}

	isSelected := index == m.Index()
	emptyFilter := m.FilterState() == list.Filtering && m.FilterValue() == ""
	isFiltered := m.FilterState() == list.Filtering || m.FilterState() == list.FilterApplied

	code := d.codeStyle.Render(si.code)
	maxNameWidth := m.Width() - lipgloss.Width(code) - 5 // padding + gap
	name := ansi.Truncate(si.name, maxNameWidth, "…")

	if emptyFilter {
		fmt.Fprintf(w, "%s %s", d.dimmedStyle.Render(name), d.codeStyle.Render(si.code))
	} else if isSelected && m.FilterState() != list.Filtering {
		if isFiltered {
			matched := m.MatchesForItem(index)
			unmatched := d.selectedStyle.Inline(true).UnsetString()
			matchStyle := unmatched.Inherit(d.filterMatch)
			name = lipgloss.StyleRunes(name, matched, matchStyle, unmatched)
		}
		fmt.Fprintf(w, "%s%s %s", d.selectedStyle.Render(""), name, code)
	} else {
		if isFiltered {
			matched := m.MatchesForItem(index)
			unmatched := d.normalStyle.Inline(true)
			matchStyle := unmatched.Inherit(d.filterMatch)
			name = lipgloss.StyleRunes(name, matched, matchStyle, unmatched)
		}
		fmt.Fprintf(w, "%s %s", d.normalStyle.Render(name), code)
	}
}

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
	spinner     spinner.Model
	apiClient   *api.Client
	departures  []api.Departure
	err         error
	width       int
	height      int
	viewport    viewport.Model
}

type searchCompleteMsg struct {
	departures []api.Departure
	err        error
}

func NewSelectorModel(apiClient *api.Client) SelectorModel {
	items := make([]list.Item, len(stations.Stations))
	for i, station := range stations.Stations {
		items[i] = stationItem{name: station.Name, code: station.Code}
	}

	delegate := newStationDelegate(true)
	l := list.New(items, delegate, 0, 0)
	l.Title = "Select Departure Station"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	s := spinner.New(
		spinner.WithSpinner(spinner.Points),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("205"))),
	)

	return SelectorModel{
		step:      selectingFrom,
		list:      l,
		spinner:   s,
		apiClient: apiClient,
	}
}

func (m SelectorModel) Init() tea.Cmd {
	return tea.RequestBackgroundColor
}

func (m SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		isDark := msg.IsDark()
		SetDarkMode(isDark)
		m.list.Styles = list.DefaultStyles(isDark)
		m.list.SetDelegate(newStationDelegate(isDark))
		if m.step == showingResults {
			m.initResultsViewport()
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-2)
		if m.step == showingResults {
			m.initResultsViewport()
		}
		return m, nil

	case tea.KeyPressMsg:
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
					m.list.Select(0)
					m.step = selectingTo
					return m, nil
				} else {
					m.toStation = &station
					m.step = searching
					return m, tea.Batch(m.spinner.Tick, m.searchDepartures())
				}
			}

		case showingResults:
			if msg.String() == "esc" || msg.String() == "q" {
				return m, tea.Quit
			}
			switch msg.String() {
			case "j":
				m.viewport.ScrollDown(1)
			case "k":
				m.viewport.ScrollUp(1)
			}
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

	case searchCompleteMsg:
		m.step = showingResults
		m.departures = msg.departures
		m.err = msg.err
		m.initResultsViewport()
		return m, nil
	}

	var cmd tea.Cmd
	switch m.step {
	case selectingFrom, selectingTo:
		m.list, cmd = m.list.Update(msg)
	case searching:
		m.spinner, cmd = m.spinner.Update(msg)
	}

	return m, cmd
}

func (m SelectorModel) View() tea.View {
	v := tea.NewView("")

	switch m.step {
	case selectingFrom, selectingTo:
		v = tea.NewView(m.list.View())

	case searching:
		theme := CurrentTheme()
		style := lipgloss.NewStyle().
			Foreground(theme.Title).
			Bold(true).
			Padding(1, 0)
		v = tea.NewView(style.Render(fmt.Sprintf("%s Searching for trains from %s to %s...",
			m.spinner.View(), m.fromStation.name, m.toStation.name)))

	case showingResults:
		if m.err != nil {
			theme := CurrentTheme()
			errorStyle := lipgloss.NewStyle().
				Foreground(theme.Error).
				Bold(true).
				Padding(1, 0)
			v = tea.NewView(errorStyle.Render(fmt.Sprintf("Error: %v\n\nPress q to quit", m.err)))
		} else {
			v = tea.NewView(m.renderDepartures())
		}
	}

	v.AltScreen = true
	return v
}

func (m *SelectorModel) searchDepartures() tea.Cmd {
	return func() tea.Msg {
		departures, err := m.apiClient.GetDepartures(m.fromStation.code, m.toStation.code)
		return searchCompleteMsg{departures: departures, err: err}
	}
}

func (m *SelectorModel) initResultsViewport() {
	headerHeight := 3 // title + blank line
	footerHeight := 3 // blank line + footer + blank line
	m.viewport = viewport.New(viewport.WithWidth(m.width), viewport.WithHeight(m.height-headerHeight-footerHeight))
	m.viewport.SetContent(m.renderTable())
}

func (m SelectorModel) renderDepartures() string {
	theme := CurrentTheme()

	if len(m.departures) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(theme.Muted).
			Padding(1, 0)
		return emptyStyle.Render("No departures found.\n\nPress q to quit")
	}

	titleStyle := lipgloss.NewStyle().Foreground(theme.Title).Bold(true)
	title := titleStyle.Render(fmt.Sprintf("Trains from %s to %s", m.fromStation.name, m.toStation.name))

	footerStyle := lipgloss.NewStyle().Foreground(theme.Muted)
	scrollInfo := fmt.Sprintf("%d%%", int(m.viewport.ScrollPercent()*100))
	footer := footerStyle.Render(fmt.Sprintf("↑/↓ scroll • %s • q to quit", scrollInfo))

	return title + "\n\n" + m.viewport.View() + "\n\n" + footer + "\n"
}

func (m SelectorModel) renderTable() string {
	theme := CurrentTheme()

	rows := [][]string{}
	addedSeparator := false
	for _, dep := range m.departures {
		if dep.NextDay && !addedSeparator {
			rows = append(rows, []string{"── Tomorrow ──", "", "", "", "", ""})
			addedSeparator = true
		}
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
			if row >= 0 && row < len(rows) && rows[row][0] == "── Tomorrow ──" {
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

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
