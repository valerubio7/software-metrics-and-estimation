package api_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/api"
	memberdomain "github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/domain"
)

// listableMemberRepository stands in for the PostgreSQL repository, which reads the collection.
type listableMemberRepository struct {
	*fakeMemberRepository
	listed    []memberdomain.Member
	listErr   error
	listCalls int
}

func (r *listableMemberRepository) ListByProject(_ context.Context, projectID string) ([]memberdomain.Member, error) {
	r.listCalls++
	if r.listErr != nil {
		return nil, r.listErr
	}
	members := make([]memberdomain.Member, 0, len(r.listed))
	for _, member := range r.listed {
		if member.ProjectID == projectID {
			members = append(members, member)
		}
	}
	return members, nil
}

func memberDependencies(repository *listableMemberRepository) api.HTTPDependencies {
	return api.HTTPDependencies{Members: &api.MemberDependencies{
		Repository: repository,
		GenerateID: func() string { return "5c21cbd4-d9a7-42df-9c3a-c0866f058746" },
		Lister:     repository,
	}}
}

func TestMemberCollectionRouteIsComposedOnlyForListableRepository(t *testing.T) {
	const projectID = "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
	email := "ada@example.com"
	repository := &listableMemberRepository{
		fakeMemberRepository: &fakeMemberRepository{},
		listed: []memberdomain.Member{
			{ID: "m1", ProjectID: projectID, FullName: "Ada Lovelace", Email: &email},
			{ID: "m2", ProjectID: projectID, FullName: "Grace Hopper"},
			{ID: "m3", ProjectID: "another-project", FullName: "Alan Turing"},
		},
	}
	handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return projectID }, memberDependencies(repository))

	response := serve(handler, http.MethodGet, "/projects/"+projectID+"/members", "")
	if response.Code != http.StatusOK || repository.listCalls != 1 {
		t.Fatalf("GET members = %d (reads %d) body=%s", response.Code, repository.listCalls, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, `"project_id":"`+projectID+`"`) || !strings.Contains(body, `"full_name":"Ada Lovelace"`) || !strings.Contains(body, `"email":null`) {
		t.Fatalf("member collection body = %s", body)
	}
	if strings.Contains(body, "Alan Turing") {
		t.Fatalf("member collection leaked another project: %s", body)
	}

	// A member module without the listing port keeps the collection unreadable: GET stays 405.
	withoutLister := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return projectID }, api.HTTPDependencies{
		Members: &api.MemberDependencies{Repository: &fakeMemberRepository{}, GenerateID: func() string { return projectID }},
	})
	if got := serve(withoutLister, http.MethodGet, "/projects/"+projectID+"/members", ""); got.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET members without lister = %d, want 405", got.Code)
	}
}

func TestMemberCollectionRouteSerializesEmptyArrayAndHidesInternalErrors(t *testing.T) {
	const projectID = "5c21cbd4-d9a7-42df-9c3a-c0866f058746"
	empty := &listableMemberRepository{fakeMemberRepository: &fakeMemberRepository{}}
	handler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return projectID }, memberDependencies(empty))
	if got := serve(handler, http.MethodGet, "/projects/"+projectID+"/members", ""); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), `"members":[]`) {
		t.Fatalf("empty members = %d %s", got.Code, got.Body.String())
	}

	failing := &listableMemberRepository{
		fakeMemberRepository: &fakeMemberRepository{},
		listErr:              errors.New("connection refused: host=db"),
	}
	failingHandler := api.NewHTTPHandlerWithDependencies(&fakeProjectRepository{}, func() string { return projectID }, memberDependencies(failing))
	got := serve(failingHandler, http.MethodGet, "/projects/"+projectID+"/members", "")
	if got.Code != http.StatusInternalServerError || strings.Contains(got.Body.String(), "connection refused") {
		t.Fatalf("failing members = %d %s, want 500 without internal detail", got.Code, got.Body.String())
	}
}
