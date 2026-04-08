# RTT CLI - UK Train Times Terminal App

A beautiful terminal UI for checking UK train times, powered by the Realtime Trains API.

## Features

- 🚂 Search for direct train services between UK stations
- 🔍 Fuzzy search for stations by name or code
- 🎨 Colorful, easy-to-read departure listings with automatic light/dark mode support
- ⚡ Fast and lightweight (single binary)
- 📊 Shows departure times, platforms, service operators, and journey duration

## Installation

```bash
go install github.com/baz-sh/rtt-cli@latest
```

Or build from source:

```bash
git clone https://github.com/baz-sh/rtt-cli.git
cd rtt-cli
go build -o rtt-cli .
```

## Configuration

This app requires an API token from Realtime Trains:

1. Register for a free account at [https://data.rtt.io/](https://data.rtt.io/)
2. On first run, you'll be prompted to enter your API token
3. Your token is stored locally in `~/.config/rtt-cli/config.json`

To reset your credentials, run:

```bash
./rtt-cli --reset
```

## Usage

Simply run the application for interactive mode:

```bash
./rtt-cli
```

Or use station codes directly for quick lookups:

```bash
./rtt-cli EUS MAN    # London Euston to Manchester Piccadilly
./rtt-cli KGX YRK    # London King's Cross to York
./rtt-cli BHM GLC    # Birmingham New Street to Glasgow Central
```

### How it works

1. **Select Departure Station**: Use arrow keys (↑/↓) to browse or type to filter stations
2. **Select Arrival Station**: Same as above for your destination
3. **View Results**: See all upcoming direct trains with:
   - Departure time
   - Time until departure
   - Departure platform (red)
   - Arrival platform (green)
   - Train service operator
   - Journey duration

### Keyboard Shortcuts

- `↑/↓` or `j/k` - Navigate station list
- `/` - Filter/search stations
- `Enter` - Select station
- `Esc` - Clear filter
- `q` or `Ctrl+C` - Quit application

## Example

```
🚂 Trains from London Euston to Manchester Piccadilly

Time      Leaving       Dep Plat  Arr Plat  Service          Duration
────────────────────────────────────────────────────────────────────────
14:30     5min          9         13        Avanti West C... 2hr 8min
14:45     20min         7         14        Avanti West C... 2hr 10min
15:00     35min         8         13        Avanti West C... 2hr 7min
```

## Theming

The application automatically detects whether your terminal has a light or dark background and adjusts its color palette accordingly. No configuration is needed — it works out of the box on most modern terminals (iTerm2, Terminal.app, Windows Terminal, GNOME Terminal, etc.).

## API

This application uses the [Realtime Trains API v2](https://data.rtt.io/) to fetch live train data.

## Tech Stack

- [Go](https://golang.org/)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Style definitions
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components

## License

MIT
