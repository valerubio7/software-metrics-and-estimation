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

// OpenSprintReader reads the open sprints of an existing project in creation order.
type OpenSprintReader interface {
	ListOpenByProject(ctx context.Context, projectID string) ([]domain.Sprint, error)
}

// ListOpenSprintsQuery identifies the project whose open sprints are requested.
type ListOpenSprintsQuery struct {
	ProjectID string
}

// ListOpenSprintsUseCase reads open sprints through its read-only port.
type ListOpenSprintsUseCase struct {
	reader OpenSprintReader
}

// NewListOpenSprintsUseCase builds an open-sprint listing use case.
func NewListOpenSprintsUseCase(reader OpenSprintReader) *ListOpenSprintsUseCase {
	return &ListOpenSprintsUseCase{reader: reader}
}

// Execute validates the route project identifier before reading open sprints.
func (u *ListOpenSprintsUseCase) Execute(ctx context.Context, query ListOpenSprintsQuery) ([]domain.Sprint, string, error) {
	projectID, err := uuid.Parse(query.ProjectID)
	if err != nil {
		return nil, "", &domain.ValidationError{Fields: map[string]string{"project_id": "must be a valid UUID"}}
	}
	sprints, err := u.reader.ListOpenByProject(ctx, projectID.String())
	if err != nil {
		return nil, "", err
	}
	return sprints, projectID.String(), nil
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
