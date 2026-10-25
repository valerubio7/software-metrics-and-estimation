package application

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/domain"
)

var (
	ErrDefectNotFound                  = errors.New("defect not found")
	ErrResolutionSprintNotFound        = errors.New("resolution sprint not found")
	ErrResolutionSprintBeforeDetection = errors.New("resolution sprint is earlier than detection sprint")
)

// UpdateDefectCommand contains the route-scoped identity and replacement lifecycle state.
type UpdateDefectCommand struct {
	ProjectID          string
	DefectID           string
	Status             string
	ResolutionSprintID string
}

// DefectUpdater atomically validates and updates a defect's resolution state.
type DefectUpdater interface {
	Update(ctx context.Context, command UpdateDefectCommand) (domain.Defect, error)
}

// UpdateDefectUseCase validates the requested lifecycle state before persistence.
type UpdateDefectUseCase struct {
	updater DefectUpdater
}

func NewUpdateDefectUseCase(updater DefectUpdater) *UpdateDefectUseCase {
	return &UpdateDefectUseCase{updater: updater}
}

func (u *UpdateDefectUseCase) Execute(ctx context.Context, command UpdateDefectCommand) (domain.Defect, error) {
	fields := make(map[string]string)
	if !domainContainsStatus(command.Status) {
		fields["status"] = domain.StatusMessage
	}

	if command.Status == "resuelto" {
		switch {
		case command.ResolutionSprintID == "":
			fields["resolution_sprint_id"] = "is required when status is resuelto"
		default:
			parsed, err := uuid.Parse(command.ResolutionSprintID)
			if err != nil {
				fields["resolution_sprint_id"] = "must be a valid UUID"
			} else {
				command.ResolutionSprintID = parsed.String()
			}
		}
	} else if command.ResolutionSprintID != "" {
		fields["resolution_sprint_id"] = "must be omitted unless status is resuelto"
	}
	if len(fields) != 0 {
		return domain.Defect{}, &domain.ValidationError{Fields: fields}
	}

	parsed, err := uuid.Parse(command.DefectID)
	if err != nil {
		return domain.Defect{}, &domain.ValidationError{Fields: map[string]string{"defect_id": "must be a valid UUID"}}
	}
	command.DefectID = parsed.String()
	return u.updater.Update(ctx, command)
}

func domainContainsStatus(status string) bool {
	for _, allowed := range domain.Statuses {
		if allowed == status {
			return true
		}
	}
	return false
}
