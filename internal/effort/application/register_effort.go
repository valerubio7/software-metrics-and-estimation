// Package application contains the use case that registers actual effort on a story or task.
package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/valerubio7/software-metrics-and-estimation/internal/effort/domain"
)

const (
	workDateLayout = "2006-01-02"

	requiredMessage        = "is required"
	invalidUUIDMessage     = "must be a valid UUID"
	invalidWorkDateMessage = "must be a valid YYYY-MM-DD date"
)

// Sentinel errors returned by EffortRepository implementations and propagated unchanged.
var (
	ErrProjectNotFound = errors.New("project not found")
	ErrStoryNotFound   = errors.New("story not found")
	ErrMemberNotFound  = errors.New("member not found")
	ErrTaskNotFound    = errors.New("task not found")
)

// RegisterEffortCommand carries the canonical route IDs and the body fields as they arrived.
type RegisterEffortCommand struct {
	ProjectID   string
	StoryID     string
	MemberID    string
	TaskID      *string // nil means effort over the story itself
	WorkDate    string  // YYYY-MM-DD
	Description string
	Hours       *float64
}

// EffortRepository verifies, in one transaction, that the project exists, the story and the
// member belong to it and, when TaskID is not nil, the task belongs to that story and
// project; then it inserts the record. It returns ErrProjectNotFound, ErrStoryNotFound,
// ErrMemberNotFound or ErrTaskNotFound for the corresponding failure; other errors are
// returned unchanged.
type EffortRepository interface {
	Create(ctx context.Context, record domain.EffortRecord) error
}

// IDGenerator creates identifiers for new effort records.
type IDGenerator func() string

// Clock returns the current instant. The use case compares work dates against its UTC date.
type Clock func() time.Time

// RegisterEffortUseCase validates the whole request before delegating to persistence.
type RegisterEffortUseCase struct {
	repository EffortRepository
	generateID IDGenerator
	now        Clock
}

// NewRegisterEffortUseCase builds a use case backed by the supplied port, ID generator and clock.
func NewRegisterEffortUseCase(repository EffortRepository, generateID IDGenerator, now Clock) *RegisterEffortUseCase {
	return &RegisterEffortUseCase{repository: repository, generateID: generateID, now: now}
}

// Execute validates every field and reports all failures together, generates the ID only
// when the input is valid, persists the record and returns it only after the repository
// confirms. Repository errors are returned unwrapped so callers can compare them with errors.Is.
func (u *RegisterEffortUseCase) Execute(ctx context.Context, command RegisterEffortCommand) (domain.EffortRecord, error) {
	fields := make(map[string]string)

	memberID := normalizeUUID(fields, "member_id", command.MemberID, true)

	var taskID *string
	if command.TaskID != nil {
		normalized := normalizeUUID(fields, "task_id", *command.TaskID, false)
		taskID = &normalized
	}

	workDate := parseWorkDate(fields, command.WorkDate)

	record, err := domain.NewEffortRecord(domain.NewEffortRecordParams{
		ProjectID:   command.ProjectID,
		StoryID:     command.StoryID,
		MemberID:    memberID,
		TaskID:      taskID,
		WorkDate:    workDate,
		Description: command.Description,
		Hours:       command.Hours,
	}, truncateToUTCDate(u.now()))
	mergeDomainFields(fields, err)
	if len(fields) > 0 {
		return domain.EffortRecord{}, &domain.ValidationError{Fields: fields}
	}

	record.ID = u.generateID()
	if err := u.repository.Create(ctx, record); err != nil {
		return domain.EffortRecord{}, err
	}
	return record, nil
}

// mergeDomainFields copies the field errors of a domain validation failure into fields. The
// domain never revalidates a zero date or the IDs, so its keys do not collide with the
// application-level ones.
func mergeDomainFields(fields map[string]string, err error) {
	var validation *domain.ValidationError
	if !errors.As(err, &validation) {
		return
	}
	for field, message := range validation.Fields {
		fields[field] = message
	}
}

// normalizeUUID records a field error for an invalid value and returns the canonical
// lowercase form of a valid one. An empty value is reported as "required" only for mandatory
// fields; an empty optional value (a present but empty task_id) is simply an invalid UUID.
func normalizeUUID(fields map[string]string, field, value string, required bool) string {
	if required && value == "" {
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

// parseWorkDate parses a strict YYYY-MM-DD civil date at UTC midnight. A non-canonical
// spelling that time.Parse tolerates is rejected by comparing the round-trip. On failure it
// records the field error and returns the zero time so the domain does not revalidate it.
func parseWorkDate(fields map[string]string, value string) time.Time {
	if value == "" {
		fields["work_date"] = requiredMessage
		return time.Time{}
	}
	parsed, err := time.Parse(workDateLayout, value)
	if err != nil || parsed.Format(workDateLayout) != value {
		fields["work_date"] = invalidWorkDateMessage
		return time.Time{}
	}
	return parsed
}

// truncateToUTCDate returns the UTC civil date of instant at midnight.
func truncateToUTCDate(instant time.Time) time.Time {
	year, month, day := instant.UTC().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
