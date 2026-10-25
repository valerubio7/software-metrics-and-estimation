package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/domain"
)

const updateDefectID = "f47ac10b-58cc-4372-a567-0e02b2c3d479"

type updateRepository struct {
	command application.UpdateDefectCommand
	calls   int
	err     error
}

func (r *updateRepository) Create(context.Context, domain.Defect) error { return nil }

func (r *updateRepository) Update(_ context.Context, command application.UpdateDefectCommand) (domain.Defect, error) {
	r.calls++
	r.command = command
	if r.err != nil {
		return domain.Defect{}, r.err
	}
	return domain.Defect{ID: command.DefectID, ProjectID: command.ProjectID, Status: command.Status, ResolutionSprintID: command.ResolutionSprintID}, nil
}

func TestUpdateDefectValidatesResolutionState(t *testing.T) {
	cases := []struct {
		name             string
		status           string
		resolutionSprint string
		wantField        string
		wantMessage      string
	}{
		{name: "resolved requires sprint", status: "resuelto", wantField: "resolution_sprint_id", wantMessage: "is required when status is resuelto"},
		{name: "other status rejects sprint", status: "detectado", resolutionSprint: "b2a6a455-06e2-41ee-b011-7462c71375a0", wantField: "resolution_sprint_id", wantMessage: "must be omitted unless status is resuelto"},
		{name: "unknown status", status: "abierto", wantField: "status", wantMessage: domain.StatusMessage},
		{name: "invalid sprint uuid", status: "resuelto", resolutionSprint: "not-a-uuid", wantField: "resolution_sprint_id", wantMessage: "must be a valid UUID"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repository := &updateRepository{}
			useCase := application.NewUpdateDefectUseCase(repository)
			_, err := useCase.Execute(context.Background(), application.UpdateDefectCommand{
				ProjectID: projectID, DefectID: updateDefectID, Status: tc.status, ResolutionSprintID: tc.resolutionSprint,
			})
			var validation *domain.ValidationError
			if !errors.As(err, &validation) || validation.Fields[tc.wantField] != tc.wantMessage {
				t.Fatalf("Execute error = %v, want %s: %s", err, tc.wantField, tc.wantMessage)
			}
			if repository.calls != 0 {
				t.Fatalf("repository calls = %d, want 0 for invalid input", repository.calls)
			}
		})
	}
}

func TestUpdateDefectPersistsResolvedAndReopenedStates(t *testing.T) {
	for _, tc := range []struct {
		name             string
		status           string
		resolutionSprint string
		wantSprint       string
	}{
		{name: "resolve", status: "resuelto", resolutionSprint: "B2A6A455-06E2-41EE-B011-7462C71375A0", wantSprint: "b2a6a455-06e2-41ee-b011-7462c71375a0"},
		{name: "reopen", status: "en_correccion", wantSprint: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repository := &updateRepository{}
			useCase := application.NewUpdateDefectUseCase(repository)
			updated, err := useCase.Execute(context.Background(), application.UpdateDefectCommand{
				ProjectID: projectID, DefectID: updateDefectID, Status: tc.status, ResolutionSprintID: tc.resolutionSprint,
			})
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if repository.calls != 1 || updated.Status != tc.status || updated.ResolutionSprintID != tc.wantSprint || repository.command.ResolutionSprintID != tc.wantSprint {
				t.Fatalf("updated=%+v command=%+v calls=%d", updated, repository.command, repository.calls)
			}
		})
	}
}

func TestUpdateDefectPropagatesRepositoryErrors(t *testing.T) {
	want := application.ErrDefectNotFound
	repository := &updateRepository{err: want}
	useCase := application.NewUpdateDefectUseCase(repository)
	_, err := useCase.Execute(context.Background(), application.UpdateDefectCommand{
		ProjectID: projectID, DefectID: updateDefectID, Status: "detectado",
	})
	if !errors.Is(err, want) {
		t.Fatalf("Execute error = %v, want %v", err, want)
	}
}
