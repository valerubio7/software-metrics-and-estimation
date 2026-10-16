package application

import (
	"context"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/project/domain"
)

// ProjectLister reads every project without changing persistence.
type ProjectLister interface {
	List(ctx context.Context) ([]domain.Project, error)
}

// ProjectSummary pairs a project with its date-derived status.
type ProjectSummary struct {
	Project domain.Project
	Status  domain.ProjectStatus
}

// ListProjectsUseCase reads every project and derives its status for a reference date.
type ListProjectsUseCase struct {
	lister ProjectLister
}

// NewListProjectsUseCase builds a list-projects use case with its only dependency.
func NewListProjectsUseCase(lister ProjectLister) *ListProjectsUseCase {
	return &ListProjectsUseCase{lister: lister}
}

// Execute returns every project in the order reported by the port, each paired with the
// status derived from referenceDate. An empty collection is not an error.
func (u *ListProjectsUseCase) Execute(ctx context.Context, referenceDate time.Time) ([]ProjectSummary, error) {
	projects, err := u.lister.List(ctx)
	if err != nil {
		return nil, err
	}
	summaries := make([]ProjectSummary, 0, len(projects))
	for _, project := range projects {
		summaries = append(summaries, ProjectSummary{Project: project, Status: project.DeriveStatus(referenceDate)})
	}
	return summaries, nil
}
