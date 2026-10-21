package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/domain"
)

type memberLister struct {
	calls     int
	projectID string
	members   []domain.Member
	err       error
}

func (l *memberLister) ListByProject(_ context.Context, projectID string) ([]domain.Member, error) {
	l.calls++
	l.projectID = projectID
	return l.members, l.err
}

func TestListMembersRejectsInvalidProjectIDBeforeReading(t *testing.T) {
	lister := &memberLister{}
	useCase := application.NewListMembersUseCase(lister)

	_, err := useCase.Execute(context.Background(), application.ListMembersQuery{ProjectID: "not-a-uuid"})

	var validation *domain.ValidationError
	if !errors.As(err, &validation) || validation.Field != "project_id" {
		t.Fatalf("err = %v, want project_id validation error", err)
	}
	if lister.calls != 0 {
		t.Fatalf("lister called %d times for invalid project id", lister.calls)
	}
}

func TestListMembersReturnsTheProjectAndItsMembers(t *testing.T) {
	projectID := "5C21CBD4-D9A7-42DF-9C3A-C0866F058746"
	email := "ada@example.com"
	lister := &memberLister{members: []domain.Member{{
		ID: "m1", ProjectID: projectID, FullName: "Ada Lovelace", Email: &email,
	}}}
	useCase := application.NewListMembersUseCase(lister)

	got, err := useCase.Execute(context.Background(), application.ListMembersQuery{ProjectID: projectID})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.ProjectID != "5c21cbd4-d9a7-42df-9c3a-c0866f058746" || lister.projectID != got.ProjectID {
		t.Fatalf("project ids = %q / %q, want canonical id", got.ProjectID, lister.projectID)
	}
	if len(got.Members) != 1 || got.Members[0].FullName != "Ada Lovelace" {
		t.Fatalf("members = %#v, want the listed member", got.Members)
	}
}

func TestListMembersPreservesAnEmptyCollectionAndPropagatesErrors(t *testing.T) {
	empty := &memberLister{members: []domain.Member{}}
	got, err := application.NewListMembersUseCase(empty).Execute(context.Background(), application.ListMembersQuery{ProjectID: "5c21cbd4-d9a7-42df-9c3a-c0866f058746"})
	if err != nil || got.Members == nil || len(got.Members) != 0 {
		t.Fatalf("empty listing = %#v, err %v; want non-nil empty slice", got.Members, err)
	}

	failure := errors.New("storage unavailable")
	failing := &memberLister{err: failure}
	if _, err := application.NewListMembersUseCase(failing).Execute(context.Background(), application.ListMembersQuery{ProjectID: "5c21cbd4-d9a7-42df-9c3a-c0866f058746"}); !errors.Is(err, failure) {
		t.Fatalf("err = %v, want the lister failure", err)
	}
}
