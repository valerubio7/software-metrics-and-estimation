package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	metricsapplication "github.com/valerubio7/software-metrics-and-estimation/internal/metrics/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/metrics/domain"
)

const (
	metricsRouteProjectID = "11111111-1111-4111-8111-111111111111"
	metricsRouteSprintID  = "22222222-2222-4222-8222-222222222222"
	metricsRouteTaskID    = "33333333-3333-4333-8333-333333333333"
)

type metricsRouteReader struct {
	calls     int
	projectID string
	sprintID  *string
	input     domain.Input
	err       error
}

func (r *metricsRouteReader) ReadMetrics(_ context.Context, projectID string, sprintID *string) (domain.Input, error) {
	r.calls++
	r.projectID = projectID
	r.sprintID = nil
	if sprintID != nil {
		value := *sprintID
		r.sprintID = &value
	}
	return r.input, r.err
}

func metricsHTTPHandler(reader metricsapplication.MetricsReader) http.Handler {
	return api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return metricsRouteProjectID }, api.HTTPDependencies{
		Metrics: &api.MetricsDependencies{Reader: reader},
	})
}

func metricsRouteInput() domain.Input {
	return domain.Input{
		Stories: []domain.StoryInput{{
			ID: "story", StoryPoints: intPointer(5), Completed: true, EstimatedHours: floatPointer(4),
			Tasks:         []domain.TaskInput{{ID: metricsRouteTaskID, SprintID: metricsRouteSprintID, EstimatedHours: floatPointer(2)}},
			EffortRecords: []domain.EffortRecordInput{{TaskID: stringPointer(metricsRouteTaskID), SprintID: stringPointer(metricsRouteSprintID), Hours: 3}},
		}},
		DefectsDetected: 2,
		DefectsResolved: 1,
	}
}

func TestMetricsRoutesAreOptionalAndReturnTheFlatContract(t *testing.T) {
	reader := &metricsRouteReader{input: metricsRouteInput()}
	withoutMetrics := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return metricsRouteProjectID }, api.HTTPDependencies{})
	for _, path := range []string{
		"/projects/" + metricsRouteProjectID + "/metrics",
		"/projects/" + metricsRouteProjectID + "/sprints/" + metricsRouteSprintID + "/metrics",
	} {
		absent := serve(withoutMetrics, http.MethodGet, path, "")
		if absent.Code != http.StatusNotFound || reader.calls != 0 {
			t.Fatalf("route %s without dependency = %d, reads=%d, want 404/0", path, absent.Code, reader.calls)
		}
	}

	handler := metricsHTTPHandler(reader)
	for _, tc := range []struct {
		name     string
		path     string
		sprintID *string
		velocity *float64
	}{
		{name: "project", path: "/projects/" + metricsRouteProjectID + "/metrics"},
		{name: "Sprint", path: "/projects/" + metricsRouteProjectID + "/sprints/" + metricsRouteSprintID + "/metrics", sprintID: stringPointer(metricsRouteSprintID), velocity: floatPointer(5)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := serve(handler, http.MethodGet, tc.path, "")
			if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("response = %d %q %s, want 200 application/json", response.Code, response.Header().Get("Content-Type"), response.Body.String())
			}
			var got domain.Metrics
			if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode metrics %q: %v", response.Body.String(), err)
			}
			want := domain.Metrics{
				ProjectID: metricsRouteProjectID, SprintID: tc.sprintID,
				StoryPointsPlanned: floatPointer(5), StoryPointsCompleted: floatPointer(5), Velocity: tc.velocity,
				EstimatedHours: floatPointer(2), ActualHours: floatPointer(3), Deviation: floatPointer(1),
				DeviationPercentage: floatPointer(50), CompletedStoriesPercentage: floatPointer(100),
				DefectsDetected: 2, DefectsResolved: 1,
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("metrics = %+v, want %+v", got, want)
			}
			var object map[string]json.RawMessage
			if err := json.Unmarshal(response.Body.Bytes(), &object); err != nil {
				t.Fatalf("decode root object: %v", err)
			}
			gotKeys := make([]string, 0, len(object))
			for key := range object {
				gotKeys = append(gotKeys, key)
			}
			sort.Strings(gotKeys)
			wantKeys := []string{"actual_hours", "completed_stories_percentage", "coverage", "defects_detected", "defects_resolved", "deviation", "deviation_percentage", "estimated_hours", "project_id", "sprint_id", "story_points_completed", "story_points_planned", "velocity"}
			sort.Strings(wantKeys)
			if !reflect.DeepEqual(gotKeys, wantKeys) {
				t.Errorf("root JSON keys = %v, want exactly %v", gotKeys, wantKeys)
			}
			var coverage map[string]json.RawMessage
			if err := json.Unmarshal(object["coverage"], &coverage); err != nil {
				t.Fatalf("decode coverage: %v", err)
			}
			coverageKeys := make([]string, 0, len(coverage))
			for key := range coverage {
				coverageKeys = append(coverageKeys, key)
			}
			sort.Strings(coverageKeys)
			wantCoverageKeys := []string{"actual_hours", "estimated_hours", "story_points_completed", "story_points_planned", "velocity"}
			if !reflect.DeepEqual(coverageKeys, wantCoverageKeys) {
				t.Errorf("coverage keys = %v, want exactly %v", coverageKeys, wantCoverageKeys)
			}
			var actualCoverage map[string]json.RawMessage
			if err := json.Unmarshal(coverage["actual_hours"], &actualCoverage); err != nil {
				t.Fatalf("decode actual-hours coverage: %v", err)
			}
			actualCoverageKeys := make([]string, 0, len(actualCoverage))
			for key := range actualCoverage {
				actualCoverageKeys = append(actualCoverageKeys, key)
			}
			sort.Strings(actualCoverageKeys)
			if want := []string{"stories_without_records", "unattributed_story_level_records"}; !reflect.DeepEqual(actualCoverageKeys, want) {
				t.Errorf("actual-hours coverage keys = %v, want exactly %v", actualCoverageKeys, want)
			}
			if reader.projectID != metricsRouteProjectID || !reflect.DeepEqual(reader.sprintID, tc.sprintID) {
				t.Errorf("reader scope = %q/%v, want %q/%v", reader.projectID, reader.sprintID, metricsRouteProjectID, tc.sprintID)
			}
		})
	}
}

func TestMetricsRoutesSupportHeadAndRejectOtherMethodsWithoutReads(t *testing.T) {
	reader := &metricsRouteReader{input: metricsRouteInput()}
	handler := metricsHTTPHandler(reader)
	paths := []struct{ name, path string }{
		{name: "project", path: "/projects/" + metricsRouteProjectID + "/metrics"},
		{name: "Sprint", path: "/projects/" + metricsRouteProjectID + "/sprints/" + metricsRouteSprintID + "/metrics"},
	}
	for _, route := range paths {
		head := serve(handler, http.MethodHead, route.path, "")
		if head.Code != http.StatusOK || head.Body.Len() != 0 || head.Header().Get("Content-Type") != "application/json" {
			t.Errorf("%s HEAD response = %d/%q/%q, want 200, empty body and JSON Content-Type", route.name, head.Code, head.Body.String(), head.Header().Get("Content-Type"))
		}
		callsAfterHead := reader.calls
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
			response := serve(handler, method, route.path, `{"ignored":true}`)
			if response.Code != http.StatusMethodNotAllowed || reader.calls != callsAfterHead {
				t.Errorf("%s %s response/read calls = %d/%d, want 405/%d", route.name, method, response.Code, reader.calls, callsAfterHead)
			}
		}
	}
}

func TestMetricsRoutesMapValidationAndReferenceErrors(t *testing.T) {
	t.Run("both malformed path UUIDs are reported without reading", func(t *testing.T) {
		reader := &metricsRouteReader{input: metricsRouteInput()}
		handler := metricsHTTPHandler(reader)
		response := serve(handler, http.MethodGet, "/projects/not-a-uuid/sprints/also-bad/metrics", "")
		var body struct {
			Error   string            `json:"error"`
			Message string            `json:"message"`
			Fields  map[string]string `json:"fields"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode validation response %q: %v", response.Body.String(), err)
		}
		wantFields := map[string]string{"project_id": "must be a valid UUID", "sprint_id": "must be a valid UUID"}
		if response.Code != http.StatusUnprocessableEntity || body.Error != "validation_failed" || body.Message != "one or more fields are invalid" || !reflect.DeepEqual(body.Fields, wantFields) || reader.calls != 0 {
			t.Fatalf("response/read calls = %d/%+v/%d, want 422 validation_failed with both fields and no read", response.Code, body, reader.calls)
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
		{name: "project missing", path: "/projects/" + metricsRouteProjectID + "/metrics", err: metricsapplication.ErrProjectNotFound, status: http.StatusNotFound, code: "project_not_found", message: "project not found"},
		{name: "Sprint missing", path: "/projects/" + metricsRouteProjectID + "/sprints/" + metricsRouteSprintID + "/metrics", err: metricsapplication.ErrSprintNotFound, status: http.StatusNotFound, code: "sprint_not_found", message: "sprint not found"},
		{name: "unexpected read failure is generic", path: "/projects/" + metricsRouteProjectID + "/metrics", err: errors.New("private connection string"), status: http.StatusInternalServerError, code: "internal_error", message: "an unexpected error occurred", secretLeak: "private connection string"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &metricsRouteReader{input: metricsRouteInput(), err: tc.err}
			response := serve(metricsHTTPHandler(reader), http.MethodGet, tc.path, "")
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
			if tc.secretLeak != "" && strings.Contains(response.Body.String(), tc.secretLeak) {
				t.Errorf("response leaked storage detail: %s", response.Body.String())
			}
		})
	}
}

func TestMetricsReadinessGatesRoutesWithoutChangingExistingReadiness(t *testing.T) {
	lookupFailure := errors.New("migration state unavailable")
	for _, tc := range []struct {
		name             string
		version          int
		dirty            bool
		lookup           error
		wantMetrics      bool
		wantDefectUpdate bool
	}{
		{name: "v13 clean", version: 13},
		{name: "v14 clean", version: 14, wantMetrics: true, wantDefectUpdate: true},
		{name: "v14 dirty", version: 14, dirty: true},
		{name: "lookup failed", version: 14, lookup: lookupFailure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			readiness := api.ResolveMigrationReadiness(tc.version, tc.dirty, tc.lookup)
			clean := !tc.dirty && tc.lookup == nil
			if readiness.Metrics != tc.wantMetrics || readiness.Projects != true ||
				readiness.Stories != (clean && tc.version >= 2) || readiness.StoryPoints != (clean && tc.version >= 2) ||
				readiness.Sprints != (clean && tc.version >= 5) || readiness.Members != (clean && tc.version >= 6) ||
				readiness.Assignment != (clean && tc.version >= 8) || readiness.SprintClose != (clean && tc.version >= 8) ||
				readiness.Tasks != (clean && tc.version >= 9) || readiness.Completion != (clean && tc.version >= 10) ||
				readiness.SprintHistory != (clean && tc.version >= 11) || readiness.Effort != (clean && tc.version >= 12) ||
				readiness.Defect != (clean && tc.version >= 13) || readiness.DefectUpdate != tc.wantDefectUpdate {
				t.Fatalf("readiness changed or Metrics gate wrong: %+v", readiness)
			}
			reader := &metricsRouteReader{input: metricsRouteInput()}
			dependencies := api.HTTPDependencies{}
			if readiness.Metrics {
				dependencies.Metrics = &api.MetricsDependencies{Reader: reader}
			}
			handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return metricsRouteProjectID }, dependencies)
			response := serve(handler, http.MethodGet, "/projects/"+metricsRouteProjectID+"/metrics", "")
			wantStatus := http.StatusNotFound
			wantReads := 0
			if tc.wantMetrics {
				wantStatus, wantReads = http.StatusOK, 1
			}
			if response.Code != wantStatus || reader.calls != wantReads {
				t.Errorf("route status/reads = %d/%d, want %d/%d", response.Code, reader.calls, wantStatus, wantReads)
			}
		})
	}
}

func intPointer(value int) *int           { return &value }
func floatPointer(value float64) *float64 { return &value }
func stringPointer(value string) *string  { return &value }
