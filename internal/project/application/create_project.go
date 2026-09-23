package application

import (
	"context"
	"time"

	"github.com/valerubio7/software-metrics-estimation/internal/project/domain"
)

const dateLayout = "2006-01-02"

// CreateProjectCommand contains the values required to create a project.
type CreateProjectCommand struct {
	Name               string
	StartDate          string
	PlannedFinishDate string
}

// ProjectRepository persists valid projects.
type ProjectRepository interface {
	Create(ctx context.Context, project domain.Project) error
}

// IDGenerator creates identifiers for new projects.
type IDGenerator func() string

// CreateProjectUseCase validates and persists a new project.
type CreateProjectUseCase struct {
	repository  ProjectRepository
	generateID IDGenerator
}

// NewCreateProjectUseCase builds a create-project use case with its dependencies.
func NewCreateProjectUseCase(repository ProjectRepository, generateID IDGenerator) *CreateProjectUseCase {
	return &CreateProjectUseCase{
		repository:  repository,
		generateID: generateID,
	}
}

// Execute creates and persists a valid project.
func (u *CreateProjectUseCase) Execute(ctx context.Context, command CreateProjectCommand) (domain.Project, error) {
	startDate, startDateError := parseDate(command.StartDate)
	plannedFinishDate, plannedFinishDateError := parseDate(command.PlannedFinishDate)
	if startDateError != "" || plannedFinishDateError != "" {
		fields := make(map[string]string)
		if startDateError != "" {
			fields["start_date"] = startDateError
		}
		if plannedFinishDateError != "" {
			fields["planned_finish_date"] = plannedFinishDateError
		}
		return domain.Project{}, &domain.ValidationError{Fields: fields}
	}

	project, err := domain.NewProject(u.generateID(), command.Name, startDate, plannedFinishDate)
	if err != nil {
		return domain.Project{}, err
	}
	if err := u.repository.Create(ctx, project); err != nil {
		return domain.Project{}, err
	}
	return project, nil
}

func parseDate(value string) (time.Time, string) {
	if value == "" {
		return time.Time{}, "is required"
	}

	date, err := time.Parse(dateLayout, value)
	if err != nil || date.Format(dateLayout) != value {
		return time.Time{}, "must be a valid YYYY-MM-DD date"
	}
	return date, ""
}
