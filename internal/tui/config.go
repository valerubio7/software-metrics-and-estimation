// Package tui provides the terminal UI for operating the system via the HTTP API.
package tui

import "strings"

// Config contains the TUI runtime configuration.
type Config struct {
	APIURL string
}

// LoadConfig reads TUI configuration from the supplied environment lookup.
func LoadConfig(getenv func(string) string) Config {
	apiURL := strings.TrimRight(strings.TrimSpace(getenv("API_URL")), "/")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}
	return Config{APIURL: apiURL}
}
