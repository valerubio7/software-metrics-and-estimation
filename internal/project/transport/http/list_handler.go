package transporthttp

import (
	"net/http"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/project/application"
)

// ListProjectsHandler serves the project collection so clients can select a project
// without knowing its identifier in advance.
type ListProjectsHandler struct {
	useCase *application.ListProjectsUseCase
	now     func() time.Time
}

// NewListProjectsHandler builds the isolated HTTP adapter; routing is composed elsewhere.
func NewListProjectsHandler(useCase *application.ListProjectsUseCase, now func() time.Time) http.Handler {
	return &ListProjectsHandler{useCase: useCase, now: now}
}

// listProjectsResponse is the container of the collection, so paging metadata can be added later.
type listProjectsResponse struct {
	Projects []projectSummaryResponse `json:"projects"`
}

type projectSummaryResponse struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	StartDate         string `json:"start_date"`
	PlannedFinishDate string `json:"planned_finish_date"`
	Status            string `json:"status"`
}

func (h *ListProjectsHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	// A GET route also serves HEAD, so both methods must reach the handler.
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writeJSON(response, http.StatusMethodNotAllowed, errorResponse{
			Error: "method_not_allowed", Message: "only GET is supported",
		})
		return
	}

	summaries, err := h.useCase.Execute(request.Context(), h.now().UTC())
	if err != nil {
		writeJSON(response, http.StatusInternalServerError, errorResponse{
			Error: "internal_error", Message: "an unexpected error occurred",
		})
		return
	}

	// A non-nil slice serializes as [] even when there are no projects.
	projects := make([]projectSummaryResponse, 0, len(summaries))
	for _, summary := range summaries {
		projects = append(projects, projectSummaryResponse{
			ID:                summary.Project.ID,
			Name:              summary.Project.Name,
			StartDate:         summary.Project.StartDate.Format("2006-01-02"),
			PlannedFinishDate: summary.Project.PlannedFinishDate.Format("2006-01-02"),
			Status:            string(summary.Status),
		})
	}
	writeJSON(response, http.StatusOK, listProjectsResponse{Projects: projects})
}
