package tui

import (
	"net/http"
	"time"
)

// ProbeAPI checks API availability without mutating data.
// Any HTTP response counts as available; only network errors or timeouts report unavailability.
func ProbeAPI(baseURL string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(baseURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
