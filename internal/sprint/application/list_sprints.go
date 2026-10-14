package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/domain"
)

// SprintHistoryReader reads the closed sprints of an existing project, from the most recent to the
// oldest by creation order, each with its planned stories in ascending creation order. It returns
// ErrProjectNotFound when the project does not exist; a project without closed sprints is not an error.
type SprintHistoryReader interface {
	ListClosedByProject(ctx context.Context, projectID string) ([]domain.ClosedSprint, error)
}

// ListSprintsQuery identifies the project whose sprint history is requested.
type ListSprintsQuery struct {
	ProjectID string
}

// ListSprintsUseCase reads the sprint history through its read-only port.
type ListSprintsUseCase struct {
	reader SprintHistoryReader
}

// NewListSprintsUseCase builds a list-sprints use case.
func NewListSprintsUseCase(reader SprintHistoryReader) *ListSprintsUseCase {
	return &ListSprintsUseCase{reader: reader}
}

// Execute returns a *domain.ValidationError without touching the port when project_id is not a UUID,
// otherwise the history of the project keyed by its canonical identifier.
func (u *ListSprintsUseCase) Execute(ctx context.Context, query ListSprintsQuery) (domain.SprintHistory, error) {
	projectID, err := uuid.Parse(query.ProjectID)
	if err != nil {
		return domain.SprintHistory{}, &domain.ValidationError{Fields: map[string]string{"project_id": "must be a valid UUID"}}
	}
	sprints, err := u.reader.ListClosedByProject(ctx, projectID.String())
	if err != nil {
		return domain.SprintHistory{}, err
	}
	return domain.SprintHistory{ProjectID: projectID.String(), Sprints: sprints}, nil
}
