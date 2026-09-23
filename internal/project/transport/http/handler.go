package transporthttp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/valerubio7/software-metrics-and-estimation/internal/project/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/domain"
)

// CreateProjectHandler handles create-project HTTP requests.
type CreateProjectHandler struct {
	useCase *application.CreateProjectUseCase
}

// NewCreateProjectHandler creates a handler with the supplied use case.
func NewCreateProjectHandler(useCase *application.CreateProjectUseCase) http.Handler {
	return &CreateProjectHandler{useCase: useCase}
}

type createProjectRequest struct {
	Name               string `json:"name"`
	StartDate          string `json:"start_date"`
	PlannedFinishDate string `json:"planned_finish_date"`
}

type projectResponse struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	StartDate         string `json:"start_date"`
	PlannedFinishDate string `json:"planned_finish_date"`
}

type errorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (h *CreateProjectHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeJSON(response, http.StatusMethodNotAllowed, errorResponse{
			Error:   "method_not_allowed",
			Message: "only POST is supported",
		})
		return
	}

	input, err := decodeCreateProjectRequest(request.Body)
	if err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	project, err := h.useCase.Execute(request.Context(), application.CreateProjectCommand{
		Name:               input.Name,
		StartDate:          input.StartDate,
		PlannedFinishDate: input.PlannedFinishDate,
	})
	if err != nil {
		var validationError *domain.ValidationError
		if errors.As(err, &validationError) {
			writeJSON(response, http.StatusUnprocessableEntity, errorResponse{
				Error:   "validation_failed",
				Message: "one or more fields are invalid",
				Fields:  validationError.Fields,
			})
			return
		}
		writeJSON(response, http.StatusInternalServerError, errorResponse{
			Error:   "internal_error",
			Message: "an unexpected error occurred",
		})
		return
	}

	writeJSON(response, http.StatusCreated, projectResponse{
		ID:                project.ID,
		Name:              project.Name,
		StartDate:         project.StartDate.Format("2006-01-02"),
		PlannedFinishDate: project.PlannedFinishDate.Format("2006-01-02"),
	})
}

func decodeCreateProjectRequest(body io.Reader) (createProjectRequest, error) {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	var request createProjectRequest
	if err := decoder.Decode(&request); err != nil {
		return createProjectRequest{}, errors.New("request body must be valid JSON")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return createProjectRequest{}, errors.New("request body must contain a single JSON object")
	}
	return request, nil
}

func writeJSON(response http.ResponseWriter, status int, body any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(body)
}
