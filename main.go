package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/barryhall/rtt-cli/internal/api"
	"github.com/barryhall/rtt-cli/internal/config"
	"github.com/barryhall/rtt-cli/internal/stations"
	"github.com/barryhall/rtt-cli/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Handle --reset flag
	if len(os.Args) == 2 && os.Args[1] == "--reset" {
		if err := config.Reset(); err != nil {
			fmt.Printf("Error resetting credentials: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Credentials have been reset.")
		fmt.Println("Run the application again to enter new credentials.")
		return
	}

	// Load or prompt for credentials
	cfg, err := loadOrPromptCredentials()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Create API client with credentials
	client := api.NewClient(cfg.Username, cfg.Password)

	// Check for command-line arguments: rtt-cli FROM TO
	if len(os.Args) == 3 {
		fromCode := strings.ToUpper(os.Args[1])
		toCode := strings.ToUpper(os.Args[2])

		// Validate station codes
		fromStation := findStation(fromCode)
		toStation := findStation(toCode)

		if fromStation == nil {
			fmt.Printf("Error: Unknown station code '%s'\n", fromCode)
			os.Exit(1)
		}
		if toStation == nil {
			fmt.Printf("Error: Unknown station code '%s'\n", toCode)
			os.Exit(1)
		}

		// Quick mode - fetch and display directly
		runQuickMode(client, fromCode, toCode, fromStation.Name, toStation.Name)
		return
	}

	// Interactive mode
	p := tea.NewProgram(ui.NewSelectorModel(client), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func loadOrPromptCredentials() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if cfg == nil {
		// No config exists, prompt for credentials
		cfg, err = config.PromptForCredentials()
		if err != nil {
			return nil, fmt.Errorf("failed to get credentials: %w", err)
		}
	}

	return cfg, nil
}

func findStation(code string) *stations.Station {
	for _, s := range stations.Stations {
		if s.Code == code {
			return &s
		}
	}
	return nil
}

func runQuickMode(client *api.Client, fromCode, toCode, fromName, toName string) {
	fmt.Printf("🚂 Searching for trains from %s to %s...\n\n", fromName, toName)

	departures, err := client.GetDepartures(fromCode, toCode)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if len(departures) == 0 {
		fmt.Println("No departures found.")
		os.Exit(0)
	}

	// Use the QuickDisplay model to render results
	m := ui.NewQuickDisplayModel(fromName, toName, departures)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
