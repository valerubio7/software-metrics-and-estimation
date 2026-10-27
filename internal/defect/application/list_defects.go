package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/domain"
)

// DefectLister reads defects belonging to one existing project.
// It returns ErrProjectNotFound for an unknown project; a project without defects is valid.
type DefectLister interface {
	ListByProject(ctx context.Context, projectID string) ([]domain.Defect, error)
}

// ListDefectsQuery identifies the project whose defects are requested.
type ListDefectsQuery struct {
	ProjectID string
}

// ListDefectsUseCase reads a project's defects without modifying them.
type ListDefectsUseCase struct {
	lister DefectLister
}

// NewListDefectsUseCase builds a project-scoped defect listing use case.
func NewListDefectsUseCase(lister DefectLister) *ListDefectsUseCase {
	return &ListDefectsUseCase{lister: lister}
}

// Execute validates and canonicalizes the project identifier before reading defects.
func (u *ListDefectsUseCase) Execute(ctx context.Context, query ListDefectsQuery) ([]domain.Defect, error) {
	projectID, err := uuid.Parse(query.ProjectID)
	if err != nil {
		return nil, &domain.ValidationError{Fields: map[string]string{"project_id": invalidUUIDMessage}}
	}
	defects, err := u.lister.ListByProject(ctx, projectID.String())
	if err != nil {
		return nil, err
	}
	if defects == nil {
		defects = []domain.Defect{}
	}
	return defects, nil
}
