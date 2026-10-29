package transporthttp

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/domain"
)

// ListDefectsHandler serves the defects belonging to the project identified by the route.
type ListDefectsHandler struct {
	useCase *application.ListDefectsUseCase
}

// NewListDefectsHandler builds the isolated HTTP adapter; routing is composed elsewhere.
func NewListDefectsHandler(useCase *application.ListDefectsUseCase) http.Handler {
	return &ListDefectsHandler{useCase: useCase}
}

type defectListResponse struct {
	ProjectID string           `json:"project_id"`
	Defects   []defectListItem `json:"defects"`
}

type defectListItem struct {
	ID                 string  `json:"id"`
	Description        string  `json:"description"`
	Severity           string  `json:"severity"`
	Status             string  `json:"status"`
	StoryID            string  `json:"story_id"`
	SprintID           string  `json:"sprint_id"`
	ResolutionSprintID *string `json:"resolution_sprint_id"`
}

func (h *ListDefectsHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writeJSON(response, http.StatusMethodNotAllowed, errorResponse{
			Error: "method_not_allowed", Message: "only GET is supported",
		})
		return
	}

	projectID := request.PathValue("project_id")
	defects, err := h.useCase.Execute(request.Context(), application.ListDefectsQuery{ProjectID: projectID})
	if err != nil {
		var validation *domain.ValidationError
		switch {
		case errors.As(err, &validation):
			writeJSON(response, http.StatusUnprocessableEntity, errorResponse{
				Error: "validation_failed", Message: validationMessage, Fields: validation.Fields,
			})
		case errors.Is(err, application.ErrProjectNotFound):
			writeJSON(response, http.StatusNotFound, errorResponse{Error: "project_not_found", Message: "project not found"})
		default:
			writeJSON(response, http.StatusInternalServerError, errorResponse{
				Error: "internal_error", Message: "an unexpected error occurred",
			})
		}
		return
	}

	items := make([]defectListItem, 0, len(defects))
	for _, defect := range defects {
		items = append(items, newDefectListItem(defect))
	}
	canonicalProjectID, _ := uuid.Parse(projectID) // The use case already validated this UUID.
	writeJSON(response, http.StatusOK, defectListResponse{ProjectID: canonicalProjectID.String(), Defects: items})
}

func newDefectListItem(defect domain.Defect) defectListItem {
	item := defectListItem{
		ID: defect.ID, Description: defect.Description, Severity: defect.Severity, Status: defect.Status,
		StoryID: defect.StoryID, SprintID: defect.SprintID,
	}
	if defect.ResolutionSprintID != "" {
		item.ResolutionSprintID = &defect.ResolutionSprintID
	}
	return item
}
