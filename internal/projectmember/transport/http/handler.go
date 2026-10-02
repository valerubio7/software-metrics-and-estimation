package transporthttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/domain"
)

type registerMembers interface {
	Execute(context.Context, application.RegisterMembersCommand) ([]domain.Member, error)
}
type RegisterMembersHandler struct{ useCase registerMembers }

func NewRegisterMembersHandler(useCase registerMembers) http.Handler {
	return &RegisterMembersHandler{useCase: useCase}
}

type requestBody struct {
	Members []application.MemberInput `json:"members"`
}
type errorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func (h *RegisterMembersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorBody{"method_not_allowed", "only POST is supported"})
		return
	}
	id, err := uuid.Parse(r.PathValue("project_id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{"invalid_request", "project_id must be a valid UUID"})
		return
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var body requestBody
	if decoder.Decode(&body) != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{"invalid_request", "request body must be valid JSON"})
		return
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		writeJSON(w, http.StatusBadRequest, errorBody{"invalid_request", "request body must contain a single JSON object"})
		return
	}
	members, err := h.useCase.Execute(r.Context(), application.RegisterMembersCommand{ProjectID: id.String(), Members: body.Members})
	if err != nil {
		switch {
		case errors.Is(err, application.ErrProjectNotFound):
			writeJSON(w, http.StatusNotFound, errorBody{"project_not_found", "project not found"})
		case errors.Is(err, application.ErrDuplicateMember):
			writeJSON(w, http.StatusConflict, errorBody{"duplicate_member", "member already exists for project"})
		default:
			var validation *domain.ValidationError
			if errors.As(err, &validation) {
				writeJSON(w, http.StatusBadRequest, errorBody{"validation_failed", validation.Error()})
			} else {
				writeJSON(w, http.StatusInternalServerError, errorBody{"internal_error", "an unexpected error occurred"})
			}
		}
		return
	}
	writeJSON(w, http.StatusCreated, struct {
		Members []domain.Member `json:"members"`
	}{Members: members})
}
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
