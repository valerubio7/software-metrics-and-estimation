package tui

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Report scopes identify which PDF endpoint produced a download.
const (
	projectReportScope = "project"
	sprintReportScope  = "sprint"
)

// ReportClient reads the project and Sprint PDF reports.
type ReportClient struct{ api apiClient }

// NewReportClient creates the read-only report client.
func NewReportClient(baseURL string) *ReportClient {
	return &ReportClient{api: newAPIClient(baseURL)}
}

// Project downloads the PDF report of a project.
func (c *ReportClient) Project(ctx context.Context, projectID string) (APIResponse, error) {
	return c.api.do(ctx, http.MethodGet, "/projects/"+url.PathEscape(projectID)+"/report.pdf", nil)
}

// Sprint downloads the PDF report of a Sprint.
func (c *ReportClient) Sprint(ctx context.Context, projectID, sprintID string) (APIResponse, error) {
	return c.api.do(ctx, http.MethodGet, "/projects/"+url.PathEscape(projectID)+"/sprints/"+url.PathEscape(sprintID)+"/report.pdf", nil)
}

// reportState is the download state of the last report requested from a screen.
// The sequence discards a late answer that no longer belongs to the screen.
type reportState struct {
	seq       uint64
	loading   bool
	message   string
	ok        bool
	scope     string
	projectID string
}

// ReportResultMsg is the asynchronous result of a report download.
type ReportResultMsg struct {
	Scope   string
	Seq     uint64
	Message string
	OK      bool
}

// startReport downloads the PDF of the given scope and saves it in the current
// directory. The file name mirrors the API Content-Disposition.
func (m Model) startReport(scope, projectID, sprintID string) (Model, tea.Cmd) {
	if m.reportClient == nil || projectID == "" {
		return m, nil
	}
	if scope == sprintReportScope && sprintID == "" {
		return m, nil
	}
	if m.report.loading && m.report.scope == scope {
		return m, nil
	}

	m.report.seq++
	m.report.loading = true
	m.report.message, m.report.ok = "", false
	m.report.scope, m.report.projectID = scope, projectID

	seq := m.report.seq
	client := m.reportClient
	fileName := reportFileName(scope, projectID, sprintID)
	return m, func() tea.Msg {
		var response APIResponse
		var err error
		if scope == sprintReportScope {
			response, err = client.Sprint(context.Background(), projectID, sprintID)
		} else {
			response, err = client.Project(context.Background(), projectID)
		}

		result := ReportResultMsg{Scope: scope, Seq: seq}
		if message := sectionFailure(err, response); message != "" {
			result.Message = message
			return result
		}
		if !bytes.HasPrefix(response.Body, []byte("%PDF-")) {
			result.Message = "La API no devolvió un PDF válido."
			return result
		}
		if err := os.WriteFile(fileName, response.Body, 0o644); err != nil {
			result.Message = "No se pudo guardar el reporte: " + err.Error()
			return result
		}
		path, err := filepath.Abs(fileName)
		if err != nil {
			path = fileName
		}
		result.Message, result.OK = "Reporte guardado: "+path, true
		return result
	}
}

// applyReportResult stores the answer only when it still belongs to the active
// request of the same scope.
func (m Model) applyReportResult(result ReportResultMsg) (Model, tea.Cmd) {
	if result.Scope != m.report.scope || result.Seq != m.report.seq {
		return m, nil
	}
	m.report.loading = false
	m.report.message, m.report.ok = result.Message, result.OK
	return m, nil
}

// reportView renders the download state for the active scope and project.
func (m Model) reportView(scope, projectID string) string {
	if m.report.scope != scope || m.report.projectID != projectID {
		return ""
	}
	if m.report.loading {
		return statusMutedStyle.Render("Generando reporte PDF…")
	}
	if m.report.message == "" {
		return ""
	}
	return m.resultBlock(m.report.message, m.report.ok)
}

// reportFileName builds a safe local name that matches the API download name.
func reportFileName(scope, projectID, sprintID string) string {
	if scope == sprintReportScope {
		return "reporte-sprint-" + safeReportID(sprintID) + ".pdf"
	}
	return "reporte-proyecto-" + safeReportID(projectID) + ".pdf"
}

func safeReportID(id string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, id)
	if safe == "" {
		return "reporte"
	}
	return safe
}
