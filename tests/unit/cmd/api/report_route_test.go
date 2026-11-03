package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	reportapplication "github.com/valerubio7/software-metrics-and-estimation/internal/report/application"
	reportdomain "github.com/valerubio7/software-metrics-and-estimation/internal/report/domain"
)

const (
	reportRouteProjectID = "11111111-1111-4111-8111-111111111111"
	reportRouteSprintID  = "22222222-2222-4222-8222-222222222222"
	reportRoutePDF       = "%PDF-1.7 fake report"
)

type reportRouteReader struct {
	calls     int
	projectID string
	sprintID  *string
	snapshot  reportdomain.Snapshot
	err       error
}

func (r *reportRouteReader) ReadSnapshot(_ context.Context, projectID string, sprintID *string) (reportdomain.Snapshot, error) {
	r.calls++
	r.projectID = projectID
	r.sprintID = nil
	if sprintID != nil {
		value := *sprintID
		r.sprintID = &value
	}
	return r.snapshot, r.err
}

type reportRouteRenderer struct {
	calls  int
	report reportdomain.Report
	output []byte
	err    error
}

func (r *reportRouteRenderer) Render(report reportdomain.Report) ([]byte, error) {
	r.calls++
	r.report = report
	return r.output, r.err
}

func reportHTTPHandler(reader reportapplication.SnapshotReader, renderer reportapplication.ReportRenderer) http.Handler {
	return api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return reportRouteProjectID }, api.HTTPDependencies{
		Report: &api.ReportDependencies{Reader: reader, Renderer: renderer, Now: reportRouteClock},
	})
}

func reportRouteSnapshot(sprint bool) reportdomain.Snapshot {
	completedAt := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	snapshot := reportdomain.Snapshot{
		Project: reportdomain.Project{ID: "stale-project", Name: "Proyecto Atlas"},
		Stories: []reportdomain.Story{{
			ID: "story", Title: "Historia", StoryPoints: intPointer(5), Completed: true, CompletedAt: &completedAt,
		}},
	}
	if sprint {
		snapshot.Sprint = &reportdomain.Sprint{ID: "stale-sprint", Goal: "Entrega", Closed: true}
	}
	return snapshot
}

func reportRouteClock() time.Time {
	return time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
}

func TestReportRoutesAreOptionalAndServeBothScopes(t *testing.T) {
	reader := &reportRouteReader{snapshot: reportRouteSnapshot(false)}
	withoutReport := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return reportRouteProjectID }, api.HTTPDependencies{})
	for _, path := range []string{
		"/projects/" + reportRouteProjectID + "/report.pdf",
		"/projects/" + reportRouteProjectID + "/sprints/" + reportRouteSprintID + "/report.pdf",
	} {
		absent := serve(withoutReport, http.MethodGet, path, "")
		if absent.Code != http.StatusNotFound || reader.calls != 0 {
			t.Fatalf("route %s without dependency = %d, reads=%d, want 404/0", path, absent.Code, reader.calls)
		}
	}

	for _, tc := range []struct {
		name       string
		path       string
		snapshot   reportdomain.Snapshot
		sprintID   *string
		filename   string
		sprintGoal string
	}{
		{name: "project", path: "/projects/" + reportRouteProjectID + "/report.pdf", snapshot: reportRouteSnapshot(false), filename: "reporte-proyecto-" + reportRouteProjectID + ".pdf"},
		{name: "Sprint", path: "/projects/" + reportRouteProjectID + "/sprints/" + reportRouteSprintID + "/report.pdf", snapshot: reportRouteSnapshot(true), sprintID: stringPointer(reportRouteSprintID), filename: "reporte-sprint-" + reportRouteSprintID + ".pdf", sprintGoal: "Entrega"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &reportRouteReader{snapshot: tc.snapshot}
			renderer := &reportRouteRenderer{output: []byte(reportRoutePDF)}
			response := serve(reportHTTPHandler(reader, renderer), http.MethodGet, tc.path, "")

			if response.Code != http.StatusOK || response.Body.String() != reportRoutePDF {
				t.Fatalf("response = %d %q, want 200 %q", response.Code, response.Body.String(), reportRoutePDF)
			}
			if got := response.Header().Get("Content-Type"); got != "application/pdf" {
				t.Errorf("Content-Type = %q, want application/pdf", got)
			}
			if got := response.Header().Get("Content-Disposition"); got != `attachment; filename="`+tc.filename+`"` {
				t.Errorf("Content-Disposition = %q, want attachment %q", got, tc.filename)
			}
			if got := response.Header().Get("Content-Length"); got != strconv.Itoa(len(reportRoutePDF)) {
				t.Errorf("Content-Length = %q, want %d", got, len(reportRoutePDF))
			}
			if reader.projectID != reportRouteProjectID || !reflect.DeepEqual(reader.sprintID, tc.sprintID) {
				t.Errorf("reader scope = %q/%v, want %q/%v", reader.projectID, reader.sprintID, reportRouteProjectID, tc.sprintID)
			}
			if renderer.calls != 1 || renderer.report.Project.ID != reportRouteProjectID {
				t.Fatalf("renderer calls/project = %d/%q, want 1/%q", renderer.calls, renderer.report.Project.ID, reportRouteProjectID)
			}
			if tc.sprintID != nil && (renderer.report.Sprint == nil || renderer.report.Sprint.ID != *tc.sprintID || renderer.report.Sprint.Goal != tc.sprintGoal) {
				t.Errorf("rendered Sprint = %+v, want %q with goal %q", renderer.report.Sprint, *tc.sprintID, tc.sprintGoal)
			}
			if tc.sprintID == nil && renderer.report.Sprint != nil {
				t.Errorf("rendered Sprint = %+v, want project scope", renderer.report.Sprint)
			}
			if !renderer.report.GeneratedAt.Equal(reportRouteClock()) {
				t.Errorf("generated at = %v, want %v", renderer.report.GeneratedAt, reportRouteClock())
			}
		})
	}
}

func TestReportRoutesSupportHeadAndRejectOtherMethodsWithoutReads(t *testing.T) {
	paths := []struct{ name, path string }{
		{name: "project", path: "/projects/" + reportRouteProjectID + "/report.pdf"},
		{name: "Sprint", path: "/projects/" + reportRouteProjectID + "/sprints/" + reportRouteSprintID + "/report.pdf"},
	}
	for _, route := range paths {
		reader := &reportRouteReader{snapshot: reportRouteSnapshot(route.name == "Sprint")}
		renderer := &reportRouteRenderer{output: []byte(reportRoutePDF)}
		handler := reportHTTPHandler(reader, renderer)
		head := serve(handler, http.MethodHead, route.path, "")
		if head.Code != http.StatusOK || head.Body.Len() != 0 || head.Header().Get("Content-Type") != "application/pdf" {
			t.Errorf("%s HEAD response = %d/%q/%q, want 200, empty body and PDF Content-Type", route.name, head.Code, head.Body.String(), head.Header().Get("Content-Type"))
		}
		if got := head.Header().Get("Content-Length"); got != strconv.Itoa(len(reportRoutePDF)) {
			t.Errorf("%s HEAD Content-Length = %q, want %d", route.name, got, len(reportRoutePDF))
		}
		readsAfterHead := reader.calls
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
			response := serve(handler, method, route.path, `{"ignored":true}`)
			if response.Code != http.StatusMethodNotAllowed || reader.calls != readsAfterHead {
				t.Errorf("%s %s response/read calls = %d/%d, want 405/%d", route.name, method, response.Code, reader.calls, readsAfterHead)
			}
		}
	}
}

func TestReportRoutesMapValidationAndReferenceErrors(t *testing.T) {
	t.Run("both malformed path UUIDs are reported without reading", func(t *testing.T) {
		reader := &reportRouteReader{snapshot: reportRouteSnapshot(true)}
		renderer := &reportRouteRenderer{output: []byte(reportRoutePDF)}
		response := serve(reportHTTPHandler(reader, renderer), http.MethodGet, "/projects/not-a-uuid/sprints/also-bad/report.pdf", "")
		var body struct {
			Error   string            `json:"error"`
			Message string            `json:"message"`
			Fields  map[string]string `json:"fields"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode validation response %q: %v", response.Body.String(), err)
		}
		wantFields := map[string]string{"project_id": "must be a valid UUID", "sprint_id": "must be a valid UUID"}
		if response.Code != http.StatusUnprocessableEntity || body.Error != "validation_failed" || body.Message != "one or more fields are invalid" || !reflect.DeepEqual(body.Fields, wantFields) || reader.calls != 0 || renderer.calls != 0 {
			t.Fatalf("response/read-render calls = %d/%d/%d, want 422 validation_failed with both fields and no work", response.Code, reader.calls, renderer.calls)
		}
		if got := response.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
	})

	for _, tc := range []struct {
		name       string
		path       string
		err        error
		status     int
		code       string
		message    string
		secretLeak string
	}{
		{name: "project missing", path: "/projects/" + reportRouteProjectID + "/report.pdf", err: reportapplication.ErrProjectNotFound, status: http.StatusNotFound, code: "project_not_found", message: "project not found"},
		{name: "Sprint missing", path: "/projects/" + reportRouteProjectID + "/sprints/" + reportRouteSprintID + "/report.pdf", err: reportapplication.ErrSprintNotFound, status: http.StatusNotFound, code: "sprint_not_found", message: "sprint not found"},
		{name: "unexpected read failure is generic", path: "/projects/" + reportRouteProjectID + "/report.pdf", err: errors.New("private connection string"), status: http.StatusInternalServerError, code: "internal_error", message: "an unexpected error occurred", secretLeak: "private connection string"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &reportRouteReader{snapshot: reportRouteSnapshot(false), err: tc.err}
			renderer := &reportRouteRenderer{output: []byte(reportRoutePDF)}
			response := serve(reportHTTPHandler(reader, renderer), http.MethodGet, tc.path, "")
			var body struct {
				Error   string `json:"error"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode error response %q: %v", response.Body.String(), err)
			}
			if response.Code != tc.status || body.Error != tc.code || body.Message != tc.message {
				t.Errorf("response = %d/%+v, want %d/%s/%q", response.Code, body, tc.status, tc.code, tc.message)
			}
			if renderer.calls != 0 {
				t.Errorf("renderer calls = %d, want 0 after a read failure", renderer.calls)
			}
			if tc.secretLeak != "" && strings.Contains(response.Body.String(), tc.secretLeak) {
				t.Errorf("response leaked storage detail: %s", response.Body.String())
			}
		})
	}

	t.Run("Sprint scope without a Sprint snapshot is not found", func(t *testing.T) {
		reader := &reportRouteReader{snapshot: reportRouteSnapshot(false)}
		renderer := &reportRouteRenderer{output: []byte(reportRoutePDF)}
		response := serve(reportHTTPHandler(reader, renderer), http.MethodGet, "/projects/"+reportRouteProjectID+"/sprints/"+reportRouteSprintID+"/report.pdf", "")
		if response.Code != http.StatusNotFound || renderer.calls != 0 {
			t.Errorf("response/render calls = %d/%d, want 404/0", response.Code, renderer.calls)
		}
	})

	t.Run("renderer failure is generic", func(t *testing.T) {
		reader := &reportRouteReader{snapshot: reportRouteSnapshot(false)}
		renderer := &reportRouteRenderer{err: errors.New("font subsystem secret")}
		response := serve(reportHTTPHandler(reader, renderer), http.MethodGet, "/projects/"+reportRouteProjectID+"/report.pdf", "")
		if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "font subsystem secret") {
			t.Errorf("response = %d %q, want generic 500 without renderer detail", response.Code, response.Body.String())
		}
	})
}

func TestReportReadinessGatesRoutesWithoutChangingOtherReadiness(t *testing.T) {
	lookupFailure := errors.New("migration state unavailable")
	for _, tc := range []struct {
		name       string
		version    int
		dirty      bool
		lookup     error
		wantReport bool
	}{
		{name: "v13 clean", version: 13},
		{name: "v14 clean", version: 14, wantReport: true},
		{name: "v14 dirty", version: 14, dirty: true},
		{name: "lookup failed", version: 14, lookup: lookupFailure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			readiness := api.ResolveMigrationReadiness(tc.version, tc.dirty, tc.lookup)
			clean := !tc.dirty && tc.lookup == nil
			if readiness.Report != tc.wantReport || readiness.Projects != true ||
				readiness.Metrics != tc.wantReport || readiness.DefectUpdate != tc.wantReport ||
				readiness.DefectList != tc.wantReport || readiness.Effort != (clean && tc.version >= 12) ||
				readiness.Defect != (clean && tc.version >= 13) || readiness.SprintHistory != (clean && tc.version >= 11) {
				t.Fatalf("readiness changed or Report gate wrong: %+v", readiness)
			}
			reader := &reportRouteReader{snapshot: reportRouteSnapshot(false)}
			renderer := &reportRouteRenderer{output: []byte(reportRoutePDF)}
			dependencies := api.HTTPDependencies{}
			if readiness.Report {
				dependencies.Report = &api.ReportDependencies{Reader: reader, Renderer: renderer}
			}
			handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return reportRouteProjectID }, dependencies)
			response := serve(handler, http.MethodGet, "/projects/"+reportRouteProjectID+"/report.pdf", "")
			wantStatus, wantReads := http.StatusNotFound, 0
			if tc.wantReport {
				wantStatus, wantReads = http.StatusOK, 1
			}
			if response.Code != wantStatus || reader.calls != wantReads {
				t.Errorf("route status/reads = %d/%d, want %d/%d", response.Code, reader.calls, wantStatus, wantReads)
			}
		})
	}
}
