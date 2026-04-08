package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/term"
)

type Config struct {
	Token string `json:"token"`
}

// isValid returns true if the config has the required fields.
func (c *Config) isValid() bool {
	return c.Token != ""
}

func configPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "rtt-cli", "config.json"), nil
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No config exists
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Handle stale configs (e.g. old username/password format)
	if !cfg.isValid() {
		return nil, nil
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func Reset() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil // Already doesn't exist
	}
	return err
}

func PromptForCredentials() (*Config, error) {
	fmt.Println("🔐 RTT CLI Configuration")
	fmt.Println("Get an API token at: https://data.rtt.io/")
	fmt.Println()

	fmt.Print("API Token: ")
	tokenBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return nil, err
	}
	fmt.Println() // New line after hidden input

	cfg := &Config{
		Token: strings.TrimSpace(string(tokenBytes)),
	}

	if err := Save(cfg); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("✓ Token saved to ~/.config/rtt-cli/config.json")
	fmt.Println()

	return cfg, nil
}
