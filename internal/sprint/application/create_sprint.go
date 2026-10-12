package application

import (
	"context"
	"errors"

	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/domain"
)

// ErrProjectNotFound identifies a missing project when the repository inserts a sprint.
var (
	ErrProjectNotFound = errors.New("project not found")
	ErrSprintNotFound  = errors.New("sprint not found")
	ErrSprintClosed    = errors.New("sprint is already closed")
)

// SprintCloser closes one sprint scoped to its owning project.
type SprintCloser interface {
	CloseSprint(ctx context.Context, projectID, sprintID string) error
}

// CloseSprintCommand identifies the sprint and owning project to close.
type CloseSprintCommand struct {
	ProjectID string
	SprintID  string
}

// CloseSprintUseCase closes a sprint through its project-scoped persistence port.
type CloseSprintUseCase struct {
	closer SprintCloser
}

// NewCloseSprintUseCase builds a close-sprint use case.
func NewCloseSprintUseCase(closer SprintCloser) *CloseSprintUseCase {
	return &CloseSprintUseCase{closer: closer}
}

// Execute delegates one project-scoped close operation.
func (u *CloseSprintUseCase) Execute(ctx context.Context, command CloseSprintCommand) error {
	return u.closer.CloseSprint(ctx, command.ProjectID, command.SprintID)
}

// CreateSprintCommand contains only values supplied to create a sprint.
type CreateSprintCommand struct {
	ProjectID  string
	SprintGoal string
}

// SprintRepository persists a validated sprint and ensures its project association.
type SprintRepository interface {
	Create(ctx context.Context, sprint domain.Sprint) error
}

// IDGenerator creates identifiers for new sprints.
type IDGenerator func() string

// CreateSprintUseCase creates a sprint with one repository write.
type CreateSprintUseCase struct {
	repository SprintRepository
	generateID IDGenerator
}

// NewCreateSprintUseCase builds a create-sprint use case with its dependencies.
func NewCreateSprintUseCase(repository SprintRepository, generateID IDGenerator) *CreateSprintUseCase {
	return &CreateSprintUseCase{repository: repository, generateID: generateID}
}

// Execute validates and inserts a sprint, returning it only after persistence succeeds.
func (u *CreateSprintUseCase) Execute(ctx context.Context, command CreateSprintCommand) (domain.Sprint, error) {
	sprint, err := domain.NewSprint("", command.ProjectID, command.SprintGoal)
	if err != nil {
		return domain.Sprint{}, err
	}
	sprint.ID = u.generateID()
	if err := u.repository.Create(ctx, sprint); err != nil {
		return domain.Sprint{}, err
	}
	return sprint, nil
}
