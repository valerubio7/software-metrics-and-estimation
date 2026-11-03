// Package transporthttp exposes project and Sprint PDF reports over HTTP.
package transporthttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/valerubio7/software-metrics-and-estimation/internal/report/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/report/domain"
)

const validationMessage = "one or more fields are invalid"

type errorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// ReportHandler serves the PDF report selected by the request route scope.
type ReportHandler struct {
	useCase *application.GenerateReportUseCase
}

// NewReportHandler creates the HTTP adapter; route registration belongs to the API composer.
func NewReportHandler(useCase *application.GenerateReportUseCase) http.Handler {
	return &ReportHandler{useCase: useCase}
}

func (h *ReportHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writeJSON(response, request, http.StatusMethodNotAllowed, errorResponse{
			Error: "method_not_allowed", Message: "only GET and HEAD are supported",
		})
		return
	}

	query := application.GenerateReportQuery{ProjectID: request.PathValue("project_id")}
	if sprintID := request.PathValue("sprint_id"); sprintID != "" {
		query.SprintID = &sprintID
	}
	document, err := h.useCase.Execute(request.Context(), query)
	if err != nil {
		writeUseCaseError(response, request, err)
		return
	}

	response.Header().Set("Content-Type", "application/pdf")
	response.Header().Set("Content-Disposition", contentDisposition(query))
	response.Header().Set("Content-Length", strconv.Itoa(len(document)))
	response.WriteHeader(http.StatusOK)
	if request.Method == http.MethodHead {
		return
	}
	_, _ = response.Write(document)
}

func contentDisposition(query application.GenerateReportQuery) string {
	if query.SprintID != nil {
		return fmt.Sprintf(`attachment; filename="reporte-sprint-%s.pdf"`, *query.SprintID)
	}
	return fmt.Sprintf(`attachment; filename="reporte-proyecto-%s.pdf"`, query.ProjectID)
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
