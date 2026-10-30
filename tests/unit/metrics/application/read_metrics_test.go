package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/metrics/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/metrics/domain"
)

const (
	metricsProjectID = "11111111-1111-4111-8111-111111111111"
	metricsSprintID  = "22222222-2222-4222-8222-222222222222"
)

type fakeMetricsReader struct {
	calls     int
	projectID string
	sprintID  *string
	input     domain.Input
	err       error
}

func (r *fakeMetricsReader) ReadMetrics(_ context.Context, projectID string, sprintID *string) (domain.Input, error) {
	r.calls++
	r.projectID = projectID
	if sprintID != nil {
		value := *sprintID
		r.sprintID = &value
	}
	return r.input, r.err
}

func TestReadMetricsRejectsInvalidIdentifiersWithoutReading(t *testing.T) {
	for _, tc := range []struct {
		name  string
		query application.ReadMetricsQuery
		field string
	}{
		{name: "invalid project id", query: application.ReadMetricsQuery{ProjectID: "not-a-uuid"}, field: "project_id"},
		{name: "empty project id", query: application.ReadMetricsQuery{ProjectID: ""}, field: "project_id"},
		{name: "invalid sprint id", query: application.ReadMetricsQuery{ProjectID: metricsProjectID, SprintID: stringPointer("bad")}, field: "sprint_id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &fakeMetricsReader{}
			useCase := application.NewReadMetricsUseCase(reader)

			got, err := useCase.Execute(context.Background(), tc.query)

			var validation *domain.ValidationError
			if !errors.As(err, &validation) || validation.Fields[tc.field] != "must be a valid UUID" {
				t.Fatalf("Execute() error = %v, want ValidationError for %s", err, tc.field)
			}
			if reader.calls != 0 {
				t.Errorf("reader calls = %d, want 0", reader.calls)
			}
			if !reflect.DeepEqual(got, domain.Metrics{}) {
				t.Errorf("metrics = %#v, want zero value", got)
			}
		})
	}
}

func TestReadMetricsReportsBothInvalidIdentifiersTogether(t *testing.T) {
	reader := &fakeMetricsReader{}
	useCase := application.NewReadMetricsUseCase(reader)

	got, err := useCase.Execute(context.Background(), application.ReadMetricsQuery{ProjectID: "bad-project", SprintID: stringPointer("bad-sprint")})

	var validation *domain.ValidationError
	if !errors.As(err, &validation) || !reflect.DeepEqual(validation.Fields, map[string]string{
		"project_id": "must be a valid UUID",
		"sprint_id":  "must be a valid UUID",
	}) {
		t.Fatalf("Execute() error = %v, want both invalid UUID fields", err)
	}
	if reader.calls != 0 {
		t.Errorf("reader calls = %d, want 0", reader.calls)
	}
	if !reflect.DeepEqual(got, domain.Metrics{}) {
		t.Errorf("metrics = %#v, want zero value", got)
	}
}

func TestReadMetricsPassesCanonicalScopeAndCalculatesTheResponse(t *testing.T) {
	upperProjectID := "11111111-1111-4111-8111-11111111ABCD"
	upperSprintID := "22222222-2222-4222-8222-22222222ABCD"
	reader := &fakeMetricsReader{input: domain.Input{
		Stories: []domain.StoryInput{{ID: "completed", StoryPoints: intPointer(5), Completed: true}},
	}}
	useCase := application.NewReadMetricsUseCase(reader)

	got, err := useCase.Execute(context.Background(), application.ReadMetricsQuery{ProjectID: upperProjectID, SprintID: stringPointer(upperSprintID)})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if reader.calls != 1 || reader.projectID != "11111111-1111-4111-8111-11111111abcd" || reader.sprintID == nil || *reader.sprintID != "22222222-2222-4222-8222-22222222abcd" {
		t.Errorf("reader scope = %d calls, %q/%v; want one call with canonical UUIDs", reader.calls, reader.projectID, reader.sprintID)
	}
	if got.ProjectID != "11111111-1111-4111-8111-11111111abcd" || got.SprintID == nil || *got.SprintID != "22222222-2222-4222-8222-22222222abcd" {
		t.Errorf("response scope = %q/%v, want canonical UUIDs", got.ProjectID, got.SprintID)
	}
	if got.StoryPointsPlanned == nil || *got.StoryPointsPlanned != 5 || got.StoryPointsCompleted == nil || *got.StoryPointsCompleted != 5 {
		t.Errorf("response points = %v/%v, want 5/5", got.StoryPointsPlanned, got.StoryPointsCompleted)
	}
}

func TestReadMetricsUsesProjectScopeWithoutSprintID(t *testing.T) {
	reader := &fakeMetricsReader{input: domain.Input{
		Stories: []domain.StoryInput{{ID: "pending", StoryPoints: intPointer(3)}},
	}}
	useCase := application.NewReadMetricsUseCase(reader)

	got, err := useCase.Execute(context.Background(), application.ReadMetricsQuery{ProjectID: metricsProjectID})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if reader.calls != 1 || reader.projectID != metricsProjectID || reader.sprintID != nil {
		t.Errorf("reader scope = %d calls, %q/%v, want project scope without Sprint", reader.calls, reader.projectID, reader.sprintID)
	}
	if got.ProjectID != metricsProjectID || got.SprintID != nil {
		t.Errorf("response scope = %q/%v, want project scope without Sprint", got.ProjectID, got.SprintID)
	}
}

func TestReadMetricsPropagatesReadErrorsWithoutPartialMetrics(t *testing.T) {
	readErr := errors.New("database unavailable")
	reader := &fakeMetricsReader{
		input: domain.Input{Stories: []domain.StoryInput{{ID: "partial", StoryPoints: intPointer(13)}}},
		err:   readErr,
	}
	useCase := application.NewReadMetricsUseCase(reader)

	got, err := useCase.Execute(context.Background(), application.ReadMetricsQuery{ProjectID: metricsProjectID})

	if !errors.Is(err, readErr) {
		t.Fatalf("Execute() error = %v, want %v", err, readErr)
	}
	if reader.calls != 1 {
		t.Errorf("reader calls = %d, want 1", reader.calls)
	}
	if !reflect.DeepEqual(got, domain.Metrics{}) {
		t.Errorf("metrics = %#v, want zero value, never partial metrics", got)
	}
}

func intPointer(value int) *int          { return &value }
func stringPointer(value string) *string { return &value }
