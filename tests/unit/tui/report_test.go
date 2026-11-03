package tui_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

func TestReportClientReadsProjectAndSprintPDFsOverGET(t *testing.T) {
	var requests []string
	const body = "%PDF-1.7\nreport"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		requests = append(requests, r.Method+" "+r.URL.EscapedPath())
		if len(raw) != 0 {
			t.Errorf("%s %s sent a body %q, want none", r.Method, r.URL.Path, raw)
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	client := tui.NewReportClient(server.URL)
	project, err := client.Project(t.Context(), "p 1")
	if err != nil {
		t.Fatal(err)
	}
	sprint, err := client.Sprint(t.Context(), "p1", "s 1")
	if err != nil {
		t.Fatal(err)
	}
	if project.StatusCode != http.StatusOK || string(project.Body) != body {
		t.Fatalf("project response = %+v", project)
	}
	if sprint.StatusCode != http.StatusOK || string(sprint.Body) != body {
		t.Fatalf("sprint response = %+v", sprint)
	}
	want := []string{
		"GET /projects/p%201/report.pdf",
		"GET /projects/p1/sprints/s%201/report.pdf",
	}
	if len(requests) != len(want) {
		t.Fatalf("requests = %v, want %v", requests, want)
	}
	for i := range want {
		if requests[i] != want[i] {
			t.Errorf("request[%d] = %q, want %q", i, requests[i], want[i])
		}
	}
}

func TestDashboardDownloadsProjectPDFInCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	pdf := []byte("%PDF-1.7\nproject report")
	rec := &recorder{}
	server := reportServer(t, rec, func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path != "/projects/p1/report.pdf" {
			return false
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(pdf)
		return true
	})

	m := startDashboard(t, server.URL, "p1")
	m, _ = apply(m, tea.WindowSizeMsg{Width: 200, Height: 40})
	m, cmd := apply(m, keyMsg("d"))
	if cmd == nil {
		t.Fatal("d should download the project report")
	}
	if view := stripANSI(m.View()); !strings.Contains(view, "Generando reporte PDF") {
		t.Errorf("dashboard lacks the download progress: %q", view)
	}

	m = run(t, m, cmd)
	view := stripANSI(m.View())
	path := filepath.Join(dir, "reporte-proyecto-p1.pdf")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading downloaded report: %v", err)
	}
	if !bytes.Equal(got, pdf) {
		t.Errorf("saved report = %q, want %q", got, pdf)
	}
	if !strings.Contains(view, "Reporte guardado: "+path) {
		t.Errorf("dashboard lacks the saved path %q: %q", path, view)
	}
	reportRequests := recordedReports(rec.all(), "/projects/p1/report.pdf")
	if len(reportRequests) != 1 {
		t.Fatalf("report requests = %d, want 1", len(reportRequests))
	}
	if got := reportRequests[0]; got.Method != http.MethodGet || len(got.Body) != 0 {
		t.Errorf("report request = %s body=%q, want read-only GET", got.Method, got.Body)
	}
}

func TestSprintsDownloadHighlightedSprintPDFInCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	pdf := []byte("%PDF-1.7\nsprint report")
	rec := &recorder{}
	server := reportServer(t, rec, func(w http.ResponseWriter, r *http.Request) bool {
		switch r.URL.Path {
		case "/projects/p1/sprints/open":
			_, _ = w.Write([]byte(`{"project_id":"p1","sprints":[{"id":"sp1","project_id":"p1","sprint_goal":"First","is_closed":false},{"id":"sp2","project_id":"p1","sprint_goal":"Second","is_closed":false}]}`))
			return true
		case "/projects/p1/sprints/sp2/report.pdf":
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write(pdf)
			return true
		}
		return false
	})

	m := openSprintsWithProject(t, server.URL, "p1")
	m, _ = apply(m, tea.WindowSizeMsg{Width: 200, Height: 40})
	m = press(m, tea.KeyDown)
	m, cmd := apply(m, keyMsg("d"))
	if cmd == nil {
		t.Fatal("d should download the highlighted Sprint report")
	}
	m = run(t, m, cmd)

	path := filepath.Join(dir, "reporte-sprint-sp2.pdf")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading downloaded report: %v", err)
	}
	if !bytes.Equal(got, pdf) {
		t.Errorf("saved report = %q, want %q", got, pdf)
	}
	view := stripANSI(m.View())
	if !strings.Contains(view, "Reporte guardado: "+path) {
		t.Errorf("sprint view lacks the saved path %q: %q", path, view)
	}
	reportRequests := recordedReports(rec.all(), "/projects/p1/sprints/sp2/report.pdf")
	if len(reportRequests) != 1 {
		t.Fatalf("report requests = %d, want 1", len(reportRequests))
	}
	if got := reportRequests[0]; got.Method != http.MethodGet || len(got.Body) != 0 {
		t.Errorf("report request = %s body=%q, want read-only GET", got.Method, got.Body)
	}
}

func TestReportKeysAppearInFooterAndHelp(t *testing.T) {
	server := reportServer(t, &recorder{}, func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path == "/projects/p1/sprints/open" {
			_, _ = w.Write([]byte(`{"project_id":"p1","sprints":[{"id":"sp1","project_id":"p1","sprint_goal":"Ship","is_closed":false}]}`))
			return true
		}
		return false
	})

	m := startDashboard(t, server.URL, "p1")
	view := stripANSI(m.View())
	if !strings.Contains(view, "d  PDF") {
		t.Errorf("dashboard footer lacks the report key: %q", view)
	}
	m, _ = apply(m, keyMsg("?"))
	if help := stripANSI(m.View()); !strings.Contains(help, "Descargar el reporte PDF del ámbito mostrado") {
		t.Errorf("dashboard help lacks the report action: %q", help)
	}

	// The Sprint list footer keeps its five primary keys; d lives in the help.
	m = openSprintsWithProject(t, server.URL, "p1")
	m, _ = apply(m, keyMsg("?"))
	if help := stripANSI(m.View()); !strings.Contains(help, "Descargar el reporte PDF del Sprint resaltado") {
		t.Errorf("sprint help lacks the report action: %q", help)
	}
}

func TestReportDownloadShowsAPIErrorAndWritesNothing(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	server := reportServer(t, &recorder{}, func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path != "/projects/p1/report.pdf" {
			return false
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"project_not_found","message":"project missing"}`))
		return true
	})

	m := startDashboard(t, server.URL, "p1")
	m, cmd := apply(m, keyMsg("d"))
	m = run(t, m, cmd)
	view := stripANSI(m.View())
	if !strings.Contains(view, "404") || !strings.Contains(view, "project missing") {
		t.Errorf("dashboard lacks the API error: %q", view)
	}
	if entries, err := os.ReadDir(dir); err != nil {
		t.Fatal(err)
	} else if len(entries) != 0 {
		t.Errorf("failed download wrote %d files", len(entries))
	}
}

func TestReportDownloadRejectsNonPDFResponse(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	server := reportServer(t, &recorder{}, func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path != "/projects/p1/report.pdf" {
			return false
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>not a report</html>"))
		return true
	})

	m := startDashboard(t, server.URL, "p1")
	m, cmd := apply(m, keyMsg("d"))
	m = run(t, m, cmd)
	view := stripANSI(m.View())
	if !strings.Contains(view, "no devolvió un PDF válido") {
		t.Errorf("dashboard lacks the invalid PDF message: %q", view)
	}
	if entries, err := os.ReadDir(dir); err != nil {
		t.Fatal(err)
	} else if len(entries) != 0 {
		t.Errorf("invalid download wrote %d files", len(entries))
	}
}

func TestReportDownloadDiscardsLateResults(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	server := reportServer(t, &recorder{}, func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path != "/projects/p1/report.pdf" {
			return false
		}
		_, _ = w.Write([]byte("%PDF-1.7\nproject report"))
		return true
	})

	m := startDashboard(t, server.URL, "p1")
	m, _ = apply(m, keyMsg("d"))
	// A result from a previous request must not replace the active download.
	m, _ = apply(m, tui.ReportResultMsg{Scope: "project", Seq: 0, Message: "late report", OK: true})
	view := stripANSI(m.View())
	if !strings.Contains(view, "Generando reporte PDF") || strings.Contains(view, "late report") {
		t.Fatalf("stale result changed the active download: %q", view)
	}
}

// reportServer answers the project picker, the dashboard reads and the routes
// supplied by the caller, recording every request.
func reportServer(t *testing.T, rec *recorder, handler func(w http.ResponseWriter, r *http.Request) bool) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		rec.mu.Lock()
		rec.requests = append(rec.requests, recordedRequest{Method: r.Method, Path: r.URL.EscapedPath(), Body: raw})
		rec.mu.Unlock()
		switch r.URL.Path {
		case "/projects":
			_, _ = w.Write([]byte(projectsBody(t, sampleProjects())))
			return
		case "/projects/p1":
			_, _ = w.Write([]byte(`{"id":"p1","name":"Alpha","status":"active"}`))
			return
		case "/projects/p1/metrics":
			_, _ = w.Write([]byte(dashboardMetricsBody))
			return
		}
		if handler(w, r) {
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(server.Close)
	return server
}

func recordedReports(requests []recordedRequest, path string) []recordedRequest {
	var matches []recordedRequest
	for _, request := range requests {
		if request.Path == path {
			matches = append(matches, request)
		}
	}
	return matches
}
