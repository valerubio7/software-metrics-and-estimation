package tui_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

func errUnavailableStub() error {
	return errors.New("connection refused")
}

func TestProbeAPIReachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if err := tui.ProbeAPI(server.URL); err != nil {
		t.Errorf("ProbeAPI() error = %v, want nil", err)
	}
}

func TestProbeAPITreatsHTTPResponseAsAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	if err := tui.ProbeAPI(server.URL); err != nil {
		t.Errorf("ProbeAPI() error = %v, want nil for any HTTP response", err)
	}
}

func TestProbeAPIUnreachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	if err := tui.ProbeAPI(url); err == nil {
		t.Errorf("ProbeAPI() error = nil, want non-nil for closed server")
	}
}

func TestViewShowsAPIErrWithoutCrashing(t *testing.T) {
	m := tui.NewModel("http://localhost:8080")
	m.APIErr = errUnavailableStub()

	view := m.View()
	if !strings.Contains(strings.ToLower(view), "no disponible") {
		t.Errorf("View() = %q, want api-unavailable message", view)
	}
}
