package transporthttp

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/domain"
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

type validationErrorBody struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields"`
}

type listMembersResponse struct {
	ProjectID string          `json:"project_id"`
	Members   []domain.Member `json:"members"`
}

func (h *ListMembersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// A GET route also serves HEAD, so both methods must reach the handler.
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, errorBody{"method_not_allowed", "only GET is supported"})
		return
	}

	projectID, err := uuid.Parse(r.PathValue("project_id"))
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, validationErrorBody{
			Error:   "validation_failed",
			Message: "one or more fields are invalid",
			Fields:  map[string]string{"project_id": "must be a valid UUID"},
		})
		return
	}

	result, err := h.useCase.Execute(r.Context(), application.ListMembersQuery{
		ProjectID: projectID.String(),
	})
	if err != nil {
		var validation *domain.ValidationError
		switch {
		case errors.As(err, &validation):
			writeJSON(w, http.StatusUnprocessableEntity, validationErrorBody{
				Error:   "validation_failed",
				Message: "one or more fields are invalid",
				Fields:  map[string]string{validation.Field: validation.Message},
			})
		case errors.Is(err, application.ErrProjectNotFound):
			writeJSON(w, http.StatusNotFound, errorBody{"project_not_found", "project not found"})
		default:
			writeJSON(w, http.StatusInternalServerError, errorBody{"internal_error", "an unexpected error occurred"})
		}
		return
	}

	members := result.Members
	if members == nil {
		members = []domain.Member{}
	}
	writeJSON(w, http.StatusOK, listMembersResponse{ProjectID: result.ProjectID, Members: members})
}
