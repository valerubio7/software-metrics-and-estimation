package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/domain"
)

// MemberLister reads the members of an existing project in a deterministic order.
type MemberLister interface {
	ListByProject(ctx context.Context, projectID string) ([]domain.Member, error)
}

// ListMembersQuery identifies the project whose members are requested.
type ListMembersQuery struct {
	ProjectID string
}

// ProjectMembers is the project context plus its member collection.
type ProjectMembers struct {
	ProjectID string
	Members   []domain.Member
}

// ListMembersUseCase reads a project's members without modifying anything.
type ListMembersUseCase struct {
	lister MemberLister
}

// NewListMembersUseCase builds a list-members use case with its only dependency.
func NewListMembersUseCase(lister MemberLister) *ListMembersUseCase {
	return &ListMembersUseCase{lister: lister}
}

// Execute validates project_id before reading, returns the port error without a partial
// collection, and always yields a non-nil member slice.
func (u *ListMembersUseCase) Execute(ctx context.Context, query ListMembersQuery) (ProjectMembers, error) {
	projectID, err := uuid.Parse(query.ProjectID)
	if err != nil {
		return ProjectMembers{}, &domain.ValidationError{Field: "project_id", Message: "must be a valid UUID"}
	}

	members, err := u.lister.ListByProject(ctx, projectID.String())
	if err != nil {
		return ProjectMembers{}, err
	}
	if members == nil {
		members = []domain.Member{}
	}
	return ProjectMembers{ProjectID: projectID.String(), Members: members}, nil
}
