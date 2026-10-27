// Package application contains the read-only metrics query use case.
package application

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/valerubio7/software-metrics-and-estimation/internal/metrics/domain"
)

// Sentinel errors returned by MetricsReader implementations.
var (
	ErrProjectNotFound = errors.New("project not found")
	ErrSprintNotFound  = errors.New("sprint not found")
)

// MetricsReader returns one consistent source snapshot for a project or Sprint. A nil
// sprintID requests project scope; reference errors are returned as the sentinels above.
type MetricsReader interface {
	ReadMetrics(ctx context.Context, projectID string, sprintID *string) (domain.Input, error)
}

// ReadMetricsQuery identifies the project or project-scoped Sprint to query.
type ReadMetricsQuery struct {
	ProjectID string
	SprintID  *string
}

// ReadMetricsUseCase validates the scope, reads source data, and calculates its metrics.
type ReadMetricsUseCase struct {
	reader MetricsReader
}

// NewReadMetricsUseCase constructs a read-only metrics query use case.
func NewReadMetricsUseCase(reader MetricsReader) *ReadMetricsUseCase {
	return &ReadMetricsUseCase{reader: reader}
}

// Execute validates and canonicalizes the requested UUIDs before reading. It returns no
// partial metrics when validation or the source read fails.
func (u *ReadMetricsUseCase) Execute(ctx context.Context, query ReadMetricsQuery) (domain.Metrics, error) {
	fields := make(map[string]string)
	projectID, err := uuid.Parse(query.ProjectID)
	if err != nil {
		fields["project_id"] = "must be a valid UUID"
	}

	var sprintID *string
	if query.SprintID != nil {
		parsed, parseErr := uuid.Parse(*query.SprintID)
		if parseErr != nil {
			fields["sprint_id"] = "must be a valid UUID"
		} else {
			canonical := parsed.String()
			sprintID = &canonical
		}
	}
	if len(fields) > 0 {
		return domain.Metrics{}, &domain.ValidationError{Fields: fields}
	}

	canonicalProjectID := projectID.String()
	input, err := u.reader.ReadMetrics(ctx, canonicalProjectID, sprintID)
	if err != nil {
		return domain.Metrics{}, err
	}
	input.ProjectID = canonicalProjectID
	input.SprintID = cloneString(sprintID)
	return domain.Calculate(input), nil
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
