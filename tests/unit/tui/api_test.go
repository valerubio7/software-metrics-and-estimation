package tui_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

func TestProjectAPIRequestsAndPropagatesResponses(t *testing.T) {
	var methods, paths []string
	var bodies [][]byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		paths = append(paths, r.URL.Path)
		if r.Method == http.MethodPut {
			body, _ := io.ReadAll(r.Body)
			bodies = append(bodies, body)
		}
		switch r.Method {
		case http.MethodPost:
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode create body: %v", err)
			}
			if body["name"] != "Alpha" || body["start_date"] != "2026-10-01" || body["planned_finish_date"] != "2026-12-01" {
				t.Errorf("create payload = %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"p1","name":"Alpha"}`))
		case http.MethodPut:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"p1","name":"Beta"}`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"p1","name":"Beta","status":"active"}`))
		}
	}))
	defer server.Close()

	client := tui.NewProjectClient(server.URL)
	payload := tui.ProjectInput{Name: "Alpha", StartDate: "2026-10-01", PlannedFinishDate: "2026-12-01"}
	if got, err := client.Create(t.Context(), payload); err != nil || got.StatusCode != http.StatusCreated {
		t.Fatalf("Create() = (%+v, %v), want status 201", got, err)
	}
	if got, err := client.Update(t.Context(), "p1", payload); err != nil || got.StatusCode != http.StatusOK {
		t.Fatalf("Update() = (%+v, %v), want status 200", got, err)
	}
	if got, err := client.Status(t.Context(), "p1"); err != nil || got.StatusCode != http.StatusOK {
		t.Fatalf("Status() = (%+v, %v), want status 200", got, err)
	}
	if strings.Join(methods, ",") != "POST,PUT,GET" || strings.Join(paths, ",") != "/projects,/projects/p1,/projects/p1" {
		t.Fatalf("requests = %v %v", methods, paths)
	}
	var updatePayload map[string]string
	if err := json.Unmarshal(bodies[0], &updatePayload); err != nil {
		t.Fatal(err)
	}
	if updatePayload["name"] != payload.Name || updatePayload["start_date"] != payload.StartDate || updatePayload["planned_finish_date"] != payload.PlannedFinishDate {
		t.Errorf("update payload = %#v", updatePayload)
	}
}

func TestProjectAPIErrorPreserves422And404Body(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/projects" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"error":"validation_failed","message":"invalid dates","fields":{"start_date":"invalid"}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"project_not_found","message":"project not found"}`))
	}))
	defer server.Close()
	client := tui.NewProjectClient(server.URL)

	response, err := client.Create(t.Context(), tui.ProjectInput{})
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(response.Body), "validation_failed") {
		t.Errorf("create error response = %+v", response)
	}
	response, err = client.Status(t.Context(), "missing")
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusNotFound || !strings.Contains(string(response.Body), "project_not_found") {
		t.Errorf("status error response = %+v", response)
	}

}

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
