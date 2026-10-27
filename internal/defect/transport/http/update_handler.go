package transporthttp

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/application"
)

type updateDefectRequest struct {
	Status             string `json:"status"`
	ResolutionSprintID string `json:"resolution_sprint_id"`
}

// UpdateDefectHandler changes a defect's current status and resolution Sprint.
type UpdateDefectHandler struct {
	useCase *application.UpdateDefectUseCase
}

func NewUpdateDefectHandler(useCase *application.UpdateDefectUseCase) http.Handler {
	return &UpdateDefectHandler{useCase: useCase}
}

func (h *UpdateDefectHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPatch {
		writeJSON(response, http.StatusMethodNotAllowed, errorResponse{Error: "method_not_allowed", Message: "only PATCH is supported"})
		return
	}
	var input updateDefectRequest
	if err := decodeDefectObject(request.Body, &input); err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{
			Error: "invalid_request", Message: "request body must be a single valid JSON object with allowed fields",
		})
		return
	}
	projectID, err := uuid.Parse(request.PathValue("project_id"))
	if err != nil {
		writeFieldError(response, "project_id", invalidUUIDMessage)
		return
	}
	defectID, err := uuid.Parse(request.PathValue("defect_id"))
	if err != nil {
		writeFieldError(response, "defect_id", invalidUUIDMessage)
		return
	}
	defect, err := h.useCase.Execute(request.Context(), application.UpdateDefectCommand{
		ProjectID: projectID.String(), DefectID: defectID.String(), Status: input.Status,
		ResolutionSprintID: input.ResolutionSprintID,
	})
	if err != nil {
		writeUseCaseError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, newDefectResponse(defect))
}
