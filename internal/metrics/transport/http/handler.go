// Package transporthttp exposes project and Sprint metrics over HTTP.
package transporthttp

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/valerubio7/software-metrics-and-estimation/internal/metrics/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/metrics/domain"
)

const validationMessage = "one or more fields are invalid"

type errorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// MetricsHandler serves project or Sprint metrics using the request's route scope.
type MetricsHandler struct {
	useCase *application.ReadMetricsUseCase
}

// NewMetricsHandler creates the HTTP adapter; route registration belongs to the API composer.
func NewMetricsHandler(useCase *application.ReadMetricsUseCase) http.Handler {
	return &MetricsHandler{useCase: useCase}
}

func (h *MetricsHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writeJSON(response, request, http.StatusMethodNotAllowed, errorResponse{
			Error: "method_not_allowed", Message: "only GET and HEAD are supported",
		})
		return
	}

	query := application.ReadMetricsQuery{ProjectID: request.PathValue("project_id")}
	if sprintID := request.PathValue("sprint_id"); sprintID != "" {
		query.SprintID = &sprintID
	}
	metrics, err := h.useCase.Execute(request.Context(), query)
	if err != nil {
		writeUseCaseError(response, request, err)
		return
	}
	writeJSON(response, request, http.StatusOK, metrics)
}

func writeUseCaseError(response http.ResponseWriter, request *http.Request, err error) {
	var validation *domain.ValidationError
	switch {
	case errors.As(err, &validation):
		writeJSON(response, request, http.StatusUnprocessableEntity, errorResponse{
			Error: "validation_failed", Message: validationMessage, Fields: validation.Fields,
		})
	case errors.Is(err, application.ErrProjectNotFound):
		writeJSON(response, request, http.StatusNotFound, errorResponse{Error: "project_not_found", Message: "project not found"})
	case errors.Is(err, application.ErrSprintNotFound):
		writeJSON(response, request, http.StatusNotFound, errorResponse{Error: "sprint_not_found", Message: "sprint not found"})
	default:
		writeJSON(response, request, http.StatusInternalServerError, errorResponse{
			Error: "internal_error", Message: "an unexpected error occurred",
		})
	}
}

func writeJSON(response http.ResponseWriter, request *http.Request, status int, body any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	if request.Method == http.MethodHead {
		return
	}
	_ = json.NewEncoder(response).Encode(body)
}
