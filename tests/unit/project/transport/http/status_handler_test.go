package transporthttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/project/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/domain"
	transporthttp "github.com/valerubio7/software-metrics-and-estimation/internal/project/transport/http"
)

type statusReader struct {
	project domain.Project
	err     error
	calls   int
}

func (r *statusReader) GetByID(_ context.Context, _ string) (domain.Project, error) {
	r.calls++
	return r.project, r.err
}

func TestGetProjectStatusHandlerResponseAndErrors(t *testing.T) {
	const id = "dc46073f-51fb-4393-8e80-6b7f84201cc1"
	for _, tc := range []struct {
		name, path string
		err        error
		want       int
		calls      int
	}{
		{"success", id, nil, 200, 1}, {"invalid id", "bad", nil, 422, 0}, {"missing", id, application.ErrProjectNotFound, 404, 1}, {"internal", id, errors.New("secret"), 500, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &statusReader{project: domain.Project{ID: id, Name: "Portal", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), PlannedFinishDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)}, err: tc.err}
			handler := transporthttp.NewGetProjectStatusHandler(application.NewGetProjectStatusUseCase(reader), func() time.Time { return time.Date(2026, 6, 1, 12, 0, 0, 0, time.FixedZone("local", 3600)) })
			request := httptest.NewRequest(http.MethodGet, "/projects/"+tc.path, nil)
			request.SetPathValue("project_id", tc.path)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.want || reader.calls != tc.calls {
				t.Fatalf("status=%d calls=%d body=%s", response.Code, reader.calls, response.Body.String())
			}
			if tc.name == "success" {
				var body map[string]any
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if len(body) != 3 || body["id"] != id || body["name"] != "Portal" || body["status"] != "active" {
					t.Fatalf("body=%v", body)
				}
			}
			if tc.name == "internal" && response.Body.String() == "secret" {
				t.Fatal("internal details leaked")
			}
		})
	}
}
