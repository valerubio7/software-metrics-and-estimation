package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/domain"
)

var (
	ErrProjectNotFound = errors.New("project not found")
	ErrDuplicateMember = errors.New("duplicate member")
)

type MemberInput struct {
	FullName string `json:"full_name"`
	Email    string `json:"email,omitempty"`
}
type RegisterMembersCommand struct {
	ProjectID string
	Members   []MemberInput
}
type MemberRepository interface {
	Register(context.Context, string, []domain.Member) error
}
type IDGenerator func() string

type RegisterMembersUseCase struct {
	repository MemberRepository
	generateID IDGenerator
}

func NewRegisterMembersUseCase(repository MemberRepository, generateID IDGenerator) *RegisterMembersUseCase {
	return &RegisterMembersUseCase{repository: repository, generateID: generateID}
}

func (u *RegisterMembersUseCase) Execute(ctx context.Context, command RegisterMembersCommand) ([]domain.Member, error) {
	if len(command.Members) == 0 {
		return nil, &domain.ValidationError{Field: "members", Message: "must contain at least one member"}
	}
	members := make([]domain.Member, 0, len(command.Members))
	for i, input := range command.Members {
		member, err := domain.NewMember(u.generateID(), command.ProjectID, input.FullName, input.Email)
		if err != nil {
			return nil, fmt.Errorf("members[%d]: %w", i, err)
		}
		for _, prior := range members {
			if member.SameIdentity(prior) {
				return nil, ErrDuplicateMember
			}
		}
		members = append(members, member)
	}
	if err := u.repository.Register(ctx, command.ProjectID, members); err != nil {
		return nil, err
	}
	return members, nil
}
