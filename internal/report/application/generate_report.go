// Package application contains the read-only report generation use case.
package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/valerubio7/software-metrics-and-estimation/internal/report/domain"
)

// Sentinel errors returned by SnapshotReader implementations.
var (
	ErrProjectNotFound = errors.New("project not found")
	ErrSprintNotFound  = errors.New("sprint not found")
)

// SnapshotReader returns one consistent read of the selected project or Sprint scope.
type SnapshotReader interface {
	ReadSnapshot(ctx context.Context, projectID string, sprintID *string) (domain.Snapshot, error)
}

// ReportRenderer turns a report into the final document bytes.
type ReportRenderer interface {
	Render(report domain.Report) ([]byte, error)
}

// Clock returns the reference time of the generated report.
type Clock func() time.Time

// GenerateReportQuery identifies the project or project-scoped Sprint to report.
type GenerateReportQuery struct {
	ProjectID string
	SprintID  *string
}

// GenerateReportUseCase validates the scope, reads one snapshot and renders the report.
type GenerateReportUseCase struct {
	reader   SnapshotReader
	renderer ReportRenderer
	now      Clock
}

// NewGenerateReportUseCase constructs a read-only report generator.
func NewGenerateReportUseCase(reader SnapshotReader, renderer ReportRenderer, now Clock) *GenerateReportUseCase {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &GenerateReportUseCase{reader: reader, renderer: renderer, now: now}
}

// Execute validates and canonicalizes the requested UUIDs before reading, then renders
// the report. It returns no bytes when validation, reading or rendering fails.
func (u *GenerateReportUseCase) Execute(ctx context.Context, query GenerateReportQuery) ([]byte, error) {
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
		return nil, &domain.ValidationError{Fields: fields}
	}

	canonicalProjectID := projectID.String()
	snapshot, err := u.reader.ReadSnapshot(ctx, canonicalProjectID, sprintID)
	if err != nil {
		return nil, err
	}
	snapshot.Project.ID = canonicalProjectID
	if sprintID != nil {
		if snapshot.Sprint == nil {
			return nil, ErrSprintNotFound
		}
		snapshot.Sprint.ID = *sprintID
	}

	return u.renderer.Render(domain.Build(snapshot, u.now().UTC()))
}
