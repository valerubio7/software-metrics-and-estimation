package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/domain"
)

const listDefectProjectID = "5c21cbd4-d9a7-42df-9c3a-c0866f058746"

type defectListReader struct {
	projectID string
	defects   []domain.Defect
	err       error
	calls     int
}

func (r *defectListReader) ListByProject(_ context.Context, projectID string) ([]domain.Defect, error) {
	r.calls++
	r.projectID = projectID
	return r.defects, r.err
}

func TestListDefectsReadsProjectScopedDefects(t *testing.T) {
	want := []domain.Defect{
		{ID: "defect-1", ProjectID: listDefectProjectID, StoryID: "story-1", SprintID: "sprint-1", Description: "Open defect", Severity: "alta", Status: "detectado"},
		{ID: "defect-2", ProjectID: listDefectProjectID, StoryID: "story-2", SprintID: "sprint-2", Description: "Resolved defect", Severity: "media", Status: "resuelto", ResolutionSprintID: "sprint-3"},
	}
	reader := &defectListReader{defects: want}
	got, err := application.NewListDefectsUseCase(reader).Execute(context.Background(), application.ListDefectsQuery{ProjectID: listDefectProjectID})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("defects = %#v, want %#v", got, want)
	}
	if reader.calls != 1 || reader.projectID != listDefectProjectID {
		t.Errorf("reader calls/project = %d/%q, want 1/%q", reader.calls, reader.projectID, listDefectProjectID)
	}
}

func TestListDefectsReturnsExplicitEmptyCollection(t *testing.T) {
	reader := &defectListReader{defects: []domain.Defect{}}
	got, err := application.NewListDefectsUseCase(reader).Execute(context.Background(), application.ListDefectsQuery{ProjectID: listDefectProjectID})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("defects = %#v, want non-nil empty slice", got)
	}
}

func TestListDefectsValidatesProjectBeforeReading(t *testing.T) {
	reader := &defectListReader{}
	_, err := application.NewListDefectsUseCase(reader).Execute(context.Background(), application.ListDefectsQuery{ProjectID: "not-a-uuid"})
	var validation *domain.ValidationError
	if !errors.As(err, &validation) || validation.Fields["project_id"] != "must be a valid UUID" {
		t.Fatalf("Execute error = %v, want invalid project_id validation", err)
	}
	if reader.calls != 0 {
		t.Errorf("reader calls = %d, want 0 for invalid project", reader.calls)
	}
}

func TestListDefectsPropagatesReaderError(t *testing.T) {
	want := application.ErrProjectNotFound
	reader := &defectListReader{err: want}
	_, err := application.NewListDefectsUseCase(reader).Execute(context.Background(), application.ListDefectsQuery{ProjectID: listDefectProjectID})
	if !errors.Is(err, want) {
		t.Fatalf("Execute error = %v, want %v", err, want)
	}
}
