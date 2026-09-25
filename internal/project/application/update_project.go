package application

import (
	"context"
	"errors"

	"github.com/valerubio7/software-metrics-and-estimation/internal/project/domain"
)

var ErrProjectNotFound = errors.New("project not found")

// UpdateProjectCommand contains a complete replacement for a project's basic data.
type UpdateProjectCommand struct {
	ProjectID         string
	Name              string
	StartDate         string
	PlannedFinishDate string
}

// UpdateProjectUseCase validates and persists a complete project replacement.
type UpdateProjectUseCase struct {
	repository ProjectRepository
}

// NewUpdateProjectUseCase builds an update-project use case.
func NewUpdateProjectUseCase(repository ProjectRepository) *UpdateProjectUseCase {
	return &UpdateProjectUseCase{repository: repository}
}

// Execute validates the complete replacement before changing persistence.
func (u *UpdateProjectUseCase) Execute(ctx context.Context, command UpdateProjectCommand) (domain.Project, error) {
	startDate, startDateError := parseDate(command.StartDate)
	plannedFinishDate, finishDateError := parseDate(command.PlannedFinishDate)
	if startDateError != "" || finishDateError != "" {
		fields := make(map[string]string)
		if startDateError != "" {
			fields["start_date"] = startDateError
		}
		if finishDateError != "" {
			fields["planned_finish_date"] = finishDateError
		}
		return domain.Project{}, &domain.ValidationError{Fields: fields}
	}

	project, err := domain.NewProject(command.ProjectID, command.Name, startDate, plannedFinishDate)
	if err != nil {
		return domain.Project{}, err
	}
	if err := u.repository.Update(ctx, project); err != nil {
		return domain.Project{}, err
	}
	return project, nil
}
