// Package application contains the use case that registers a defect detected during development.
package application

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/domain"
)

const (
	requiredMessage    = "is required"
	invalidUUIDMessage = "must be a valid UUID"
)

// Sentinel errors returned by DefectRepository implementations and propagated unchanged.
var (
	ErrProjectNotFound = errors.New("project not found")
	ErrStoryNotFound   = errors.New("story not found")
	ErrSprintNotFound  = errors.New("sprint not found")
)

// RegisterDefectCommand carries the canonical route project ID and the body fields as they arrived.
type RegisterDefectCommand struct {
	ProjectID   string // from the route, already canonical
	StoryID     string // related story, required
	SprintID    string // detection sprint, required
	Description string
	Severity    string
	Status      string
}

// DefectRepository verifies, in one transaction, that the project exists and that the story
// and the detection sprint belong to it, then inserts the record. It returns
// ErrProjectNotFound, ErrStoryNotFound or ErrSprintNotFound for the corresponding failure;
// other errors are returned unchanged.
type DefectRepository interface {
	Create(ctx context.Context, defect domain.Defect) error
}

// IDGenerator creates identifiers for new defects.
type IDGenerator func() string

// RegisterDefectUseCase validates the whole request before delegating to persistence.
type RegisterDefectUseCase struct {
	repository DefectRepository
	generateID IDGenerator
}

// NewRegisterDefectUseCase builds a use case backed by the supplied port and ID generator.
func NewRegisterDefectUseCase(repository DefectRepository, generateID IDGenerator) *RegisterDefectUseCase {
	return &RegisterDefectUseCase{repository: repository, generateID: generateID}
}

// Execute validates the story and sprint identifiers plus the body fields, reports all
// failures together, generates the ID only when the input is valid, and returns the
// persisted record only after the repository confirms. Repository errors are returned
// unwrapped so callers can compare them with errors.Is.
func (u *RegisterDefectUseCase) Execute(ctx context.Context, command RegisterDefectCommand) (domain.Defect, error) {
	fields := make(map[string]string)

	storyID := normalizeUUID(fields, "story_id", command.StoryID)
	sprintID := normalizeUUID(fields, "sprint_id", command.SprintID)

	defect, err := domain.NewDefect(domain.NewDefectParams{
		ID:          "",
		ProjectID:   command.ProjectID,
		StoryID:     storyID,
		SprintID:    sprintID,
		Description: command.Description,
		Severity:    command.Severity,
		Status:      command.Status,
	})
	mergeDomainFields(fields, err)
	if len(fields) > 0 {
		return domain.Defect{}, &domain.ValidationError{Fields: fields}
	}

	defect.ID = u.generateID()
	if err := u.repository.Create(ctx, defect); err != nil {
		return domain.Defect{}, err
	}
	return defect, nil
}

// mergeDomainFields copies the field errors of a domain validation failure into fields.
func mergeDomainFields(fields map[string]string, err error) {
	var validation *domain.ValidationError
	if !errors.As(err, &validation) {
		return
	}
	for field, message := range validation.Fields {
		fields[field] = message
	}
}

// normalizeUUID records a field error for an empty or invalid value and returns the
// canonical lowercase form of a valid one.
func normalizeUUID(fields map[string]string, field, value string) string {
	if value == "" {
		fields[field] = requiredMessage
		return ""
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		fields[field] = invalidUUIDMessage
		return ""
	}
	return parsed.String()
}
