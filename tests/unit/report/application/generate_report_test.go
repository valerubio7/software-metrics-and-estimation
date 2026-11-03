package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/report/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/report/domain"
)

const (
	reportProjectID = "11111111-1111-4111-8111-111111111111"
	reportSprintID  = "22222222-2222-4222-8222-222222222222"
)

type fakeReader struct {
	calls     int
	projectID string
	sprintID  *string
	snapshot  domain.Snapshot
	err       error
}

func (r *fakeReader) ReadSnapshot(_ context.Context, projectID string, sprintID *string) (domain.Snapshot, error) {
	r.calls++
	r.projectID = projectID
	if sprintID != nil {
		value := *sprintID
		r.sprintID = &value
	}
	return r.snapshot, r.err
}

type fakeRenderer struct {
	calls  int
	report domain.Report
	output []byte
	err    error
}

func (r *fakeRenderer) Render(report domain.Report) ([]byte, error) {
	r.calls++
	r.report = report
	return r.output, r.err
}

func TestGenerateReportRejectsInvalidIdentifiersWithoutReadingOrRendering(t *testing.T) {
	for _, tc := range []struct {
		name  string
		query application.GenerateReportQuery
		field string
	}{
		{name: "invalid project", query: application.GenerateReportQuery{ProjectID: "not-a-uuid"}, field: "project_id"},
		{name: "empty project", query: application.GenerateReportQuery{ProjectID: ""}, field: "project_id"},
		{name: "invalid Sprint", query: application.GenerateReportQuery{ProjectID: reportProjectID, SprintID: stringPointer("bad")}, field: "sprint_id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &fakeReader{}
			renderer := &fakeRenderer{output: []byte("pdf")}
			useCase := application.NewGenerateReportUseCase(reader, renderer, fixedClock)

			document, err := useCase.Execute(context.Background(), tc.query)

			var validation *domain.ValidationError
			if !errors.As(err, &validation) || validation.Fields[tc.field] != "must be a valid UUID" {
				t.Fatalf("Execute() error = %v, want ValidationError for %s", err, tc.field)
			}
			if reader.calls != 0 || renderer.calls != 0 || document != nil {
				t.Errorf("calls = reads %d, renders %d, document %v; want 0/0/nil", reader.calls, renderer.calls, document)
			}
		})
	}
}

func TestGenerateReportReportsBothInvalidIdentifiersTogether(t *testing.T) {
	reader := &fakeReader{}
	renderer := &fakeRenderer{}
	useCase := application.NewGenerateReportUseCase(reader, renderer, fixedClock)

	document, err := useCase.Execute(context.Background(), application.GenerateReportQuery{
		ProjectID: "bad-project",
		SprintID:  stringPointer("bad-sprint"),
	})

	var validation *domain.ValidationError
	if !errors.As(err, &validation) || !reflect.DeepEqual(validation.Fields, map[string]string{
		"project_id": "must be a valid UUID",
		"sprint_id":  "must be a valid UUID",
	}) {
		t.Fatalf("Execute() error = %v, want both invalid UUID fields", err)
	}
	if reader.calls != 0 || renderer.calls != 0 || document != nil {
		t.Errorf("calls = reads %d, renders %d, document %v; want 0/0/nil", reader.calls, renderer.calls, document)
	}
}

func TestGenerateReportCanonicalizesScopeAndRendersTheBuiltReport(t *testing.T) {
	upperProjectID := "11111111-1111-4111-8111-11111111ABCD"
	upperSprintID := "22222222-2222-4222-8222-22222222ABCD"
	reader := &fakeReader{snapshot: domain.Snapshot{
		Project: domain.Project{ID: "stale-project", Name: "Proyecto"},
		Sprint:  &domain.Sprint{ID: "stale-sprint", Goal: "Entrega"},
		Stories: []domain.Story{{
			ID: "story", Title: "Historia", StoryPoints: intPointer(5), Completed: true,
		}},
	}}
	renderer := &fakeRenderer{output: []byte("%PDF-1.7")}
	useCase := application.NewGenerateReportUseCase(reader, renderer, fixedClock)

	document, err := useCase.Execute(context.Background(), application.GenerateReportQuery{
		ProjectID: upperProjectID,
		SprintID:  stringPointer(upperSprintID),
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if string(document) != "%PDF-1.7" {
		t.Errorf("document = %q, want renderer output", document)
	}
	if reader.calls != 1 || reader.projectID != "11111111-1111-4111-8111-11111111abcd" ||
		reader.sprintID == nil || *reader.sprintID != "22222222-2222-4222-8222-22222222abcd" {
		t.Errorf("reader scope = %d calls, %q/%v; want canonical UUIDs", reader.calls, reader.projectID, reader.sprintID)
	}
	if renderer.calls != 1 {
		t.Fatalf("renderer calls = %d, want 1", renderer.calls)
	}
	if renderer.report.Project.ID != "11111111-1111-4111-8111-11111111abcd" ||
		renderer.report.Sprint == nil || renderer.report.Sprint.ID != "22222222-2222-4222-8222-22222222abcd" {
		t.Errorf("rendered scope = %q/%+v, want canonical UUIDs", renderer.report.Project.ID, renderer.report.Sprint)
	}
	if !renderer.report.GeneratedAt.Equal(fixedTime) {
		t.Errorf("generated at = %v, want %v", renderer.report.GeneratedAt, fixedTime)
	}
	if renderer.report.Metrics.Velocity == nil || *renderer.report.Metrics.Velocity != 5 {
		t.Errorf("Sprint velocity = %v, want 5", renderer.report.Metrics.Velocity)
	}
}

func TestGenerateReportUsesProjectScopeWithoutSprint(t *testing.T) {
	reader := &fakeReader{snapshot: domain.Snapshot{
		Project: domain.Project{ID: "project", Name: "Proyecto"},
		Stories: []domain.Story{{ID: "story", StoryPoints: intPointer(3)}},
	}}
	renderer := &fakeRenderer{output: []byte("pdf")}
	useCase := application.NewGenerateReportUseCase(reader, renderer, fixedClock)

	document, err := useCase.Execute(context.Background(), application.GenerateReportQuery{ProjectID: reportProjectID})

	if err != nil || document == nil {
		t.Fatalf("Execute() = (%v, %v), want document", document, err)
	}
	if reader.calls != 1 || reader.projectID != reportProjectID || reader.sprintID != nil {
		t.Errorf("reader scope = %d calls, %q/%v; want project scope without Sprint", reader.calls, reader.projectID, reader.sprintID)
	}
	if renderer.report.Sprint != nil {
		t.Errorf("rendered Sprint = %+v, want nil", renderer.report.Sprint)
	}
}

func TestGenerateReportPropagatesReaderErrorsWithoutRendering(t *testing.T) {
	readErr := errors.New("database unavailable")
	reader := &fakeReader{err: readErr}
	renderer := &fakeRenderer{output: []byte("partial")}
	useCase := application.NewGenerateReportUseCase(reader, renderer, fixedClock)

	document, err := useCase.Execute(context.Background(), application.GenerateReportQuery{ProjectID: reportProjectID})

	if !errors.Is(err, readErr) {
		t.Fatalf("Execute() error = %v, want %v", err, readErr)
	}
	if document != nil || renderer.calls != 0 {
		t.Errorf("document/renders = %v/%d, want no partial document and no render", document, renderer.calls)
	}
}

func TestGenerateReportPropagatesRendererErrors(t *testing.T) {
	renderErr := errors.New("font failure")
	reader := &fakeReader{snapshot: domain.Snapshot{Project: domain.Project{ID: "project"}}}
	renderer := &fakeRenderer{err: renderErr}
	useCase := application.NewGenerateReportUseCase(reader, renderer, fixedClock)

	document, err := useCase.Execute(context.Background(), application.GenerateReportQuery{ProjectID: reportProjectID})

	if !errors.Is(err, renderErr) {
		t.Fatalf("Execute() error = %v, want %v", err, renderErr)
	}
	if document != nil || renderer.calls != 1 {
		t.Errorf("document/renders = %v/%d, want no bytes and one render", document, renderer.calls)
	}
}

func TestGenerateReportRejectsSprintScopeWithoutSprintSnapshot(t *testing.T) {
	reader := &fakeReader{snapshot: domain.Snapshot{Project: domain.Project{ID: "project"}}}
	renderer := &fakeRenderer{output: []byte("pdf")}
	useCase := application.NewGenerateReportUseCase(reader, renderer, fixedClock)

	document, err := useCase.Execute(context.Background(), application.GenerateReportQuery{
		ProjectID: reportProjectID,
		SprintID:  stringPointer(reportSprintID),
	})

	if !errors.Is(err, application.ErrSprintNotFound) {
		t.Fatalf("Execute() error = %v, want ErrSprintNotFound", err)
	}
	if document != nil || renderer.calls != 0 {
		t.Errorf("document/renders = %v/%d, want no bytes and no render", document, renderer.calls)
	}
}

var fixedTime = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

func fixedClock() time.Time { return fixedTime }

func intPointer(value int) *int          { return &value }
func stringPointer(value string) *string { return &value }
