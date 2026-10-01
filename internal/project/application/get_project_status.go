package application

import (
	"context"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/project/domain"
)

// ProjectReader retrieves a project without changing persistence.
type ProjectReader interface {
	GetByID(ctx context.Context, id string) (domain.Project, error)
}

// GetProjectStatusUseCase reads a project and derives its status for a calendar date.
type GetProjectStatusUseCase struct {
	reader ProjectReader
}

// NewGetProjectStatusUseCase creates the read-only status query.
func NewGetProjectStatusUseCase(reader ProjectReader) *GetProjectStatusUseCase {
	return &GetProjectStatusUseCase{reader: reader}
}

// Execute retrieves the identified project and derives its status from referenceDate.
func (u *GetProjectStatusUseCase) Execute(ctx context.Context, id string, referenceDate time.Time) (domain.Project, domain.ProjectStatus, error) {
	project, err := u.reader.GetByID(ctx, id)
	if err != nil {
		return domain.Project{}, "", err
	}
	return project, project.DeriveStatus(referenceDate), nil
}
