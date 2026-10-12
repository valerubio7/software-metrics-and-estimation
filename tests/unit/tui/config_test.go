package tui_test

import (
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

func TestLoadConfigDefaultsToLocalhost(t *testing.T) {
	cfg := tui.LoadConfig(func(string) string { return "" })

	if cfg.APIURL != "http://localhost:8080" {
		t.Errorf("APIURL = %q, want %q", cfg.APIURL, "http://localhost:8080")
	}
}

func TestLoadConfigUsesExplicitAPIURL(t *testing.T) {
	cfg := tui.LoadConfig(func(key string) string {
		if key == "API_URL" {
			return "http://api:9000"
		}
		return ""
	})

	if cfg.APIURL != "http://api:9000" {
		t.Errorf("APIURL = %q, want %q", cfg.APIURL, "http://api:9000")
	}
}

func TestLoadConfigTrimsTrailingSlash(t *testing.T) {
	cfg := tui.LoadConfig(func(key string) string {
		if key == "API_URL" {
			return "http://api:9000/"
		}
		return ""
	})

	if cfg.APIURL != "http://api:9000" {
		t.Errorf("APIURL = %q, want %q", cfg.APIURL, "http://api:9000")
	}
}
