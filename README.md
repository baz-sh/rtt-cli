# RTT CLI - UK Train Times Terminal App

A terminal UI for checking UK train times, powered by the Realtime Trains API v2.

## Features

- Search for direct train services between UK stations
- Fuzzy search for stations by name or code
- Automatic light/dark mode detection
- Loading animations during search
- Fast and lightweight (single binary)
- Shows departure times, platforms, service operators, and journey duration

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

1. **Select Departure Station**: Browse or type to fuzzy-search stations
2. **Select Arrival Station**: Same as above for your destination
3. **View Results**: See all upcoming direct trains with departure time, time until departure, platforms, service operator, and journey duration

### Keyboard Shortcuts

- `↑/↓` or `j/k` - Navigate station list / scroll results
- `/` - Filter/search stations
- `Enter` - Select station
- `Esc` - Clear filter
- `q` or `Ctrl+C` - Quit

## Example

```
Trains from London Euston to Manchester Piccadilly

┌───────┬──────────┬──────────┬──────────┬──────────────────────┬──────────┐
│ Time  │ Leaving  │ Dep Plat │ Arr Plat │ Service              │ Duration │
├───────┼──────────┼──────────┼──────────┼──────────────────────┼──────────┤
│ 14:30 │ 5min     │ 9        │ 13       │ Avanti West Coast    │ 2hr 8min │
│ 14:45 │ 20min    │ 7        │ 14       │ Avanti West Coast    │ 2hr 10m  │
│ 15:00 │ 35min    │ 8        │ 13       │ Avanti West Coast    │ 2hr 7min │
└───────┴──────────┴──────────┴──────────┴──────────────────────┴──────────┘
```

## Theming

The application automatically detects whether your terminal has a light or dark background and adjusts its color palette accordingly. No configuration needed.

## API

This application uses the [Realtime Trains API v2](https://data.rtt.io/) to fetch live train data. Authentication uses a refresh token exchanged for short-life access tokens.

## Tech Stack

- [Go](https://golang.org/)
- [Bubble Tea v2](https://charm.land/bubbletea/v2) - Terminal UI framework
- [Lip Gloss v2](https://charm.land/lipgloss/v2) - Style definitions
- [Bubbles v2](https://charm.land/bubbles/v2) - TUI components

## License

MIT
