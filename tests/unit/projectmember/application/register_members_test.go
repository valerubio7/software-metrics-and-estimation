package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/domain"
)

type repository struct {
	calls   int
	err     error
	members []domain.Member
}

func (r *repository) Register(_ context.Context, _ string, members []domain.Member) error {
	r.calls++
	r.members = members
	return r.err
}

func TestRegisterMembersValidatesWholeBatchBeforeRepository(t *testing.T) {
	repo := &repository{}
	next := 0
	uc := application.NewRegisterMembersUseCase(repo, func() string { next++; return string(rune('0' + next)) })
	for _, batch := range [][]application.MemberInput{nil, {{FullName: "Ada"}, {FullName: " "}}, {{FullName: "Ada", Email: "bad"}}, {{FullName: "Ada"}, {FullName: "Ada"}}} {
		if _, err := uc.Execute(context.Background(), application.RegisterMembersCommand{ProjectID: "p", Members: batch}); err == nil {
			t.Fatalf("expected rejection for %#v", batch)
		}
	}
	if repo.calls != 0 {
		t.Fatalf("repository called %d times for invalid batches", repo.calls)
	}
}

func TestRegisterMembersPersistsAllValidMembersAndReturnsRepositoryErrors(t *testing.T) {
	repo := &repository{}
	uc := application.NewRegisterMembersUseCase(repo, func() string { return "member-id" })
	got, err := uc.Execute(context.Background(), application.RegisterMembersCommand{ProjectID: "project", Members: []application.MemberInput{{FullName: "Ada"}, {FullName: "Grace", Email: "grace@example.com"}}})
	if err != nil || len(got) != 2 || repo.calls != 1 {
		t.Fatalf("got %v, err %v, calls %d", got, err, repo.calls)
	}
	failure := errors.New("storage failure")
	repo.err = failure
	if _, err := uc.Execute(context.Background(), application.RegisterMembersCommand{ProjectID: "project", Members: []application.MemberInput{{FullName: "Lin"}}}); !errors.Is(err, failure) {
		t.Fatalf("expected repository failure, got %v", err)
	}
}
