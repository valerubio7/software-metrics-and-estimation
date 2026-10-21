package application

import (
	"context"

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

// Execute is implemented in the GREEN step.
func (u *ListMembersUseCase) Execute(ctx context.Context, query ListMembersQuery) (ProjectMembers, error) {
	return ProjectMembers{}, nil
}
