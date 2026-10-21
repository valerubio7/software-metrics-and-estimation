package transporthttp

import (
	"context"
	"net/http"

	"github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/application"
)

type listMembers interface {
	Execute(context.Context, application.ListMembersQuery) (application.ProjectMembers, error)
}

// ListMembersHandler serves the members of the project identified by the route.
type ListMembersHandler struct {
	useCase listMembers
}

// NewListMembersHandler builds the isolated HTTP adapter; routing is composed elsewhere.
func NewListMembersHandler(useCase listMembers) http.Handler {
	return &ListMembersHandler{useCase: useCase}
}

func (h *ListMembersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}
