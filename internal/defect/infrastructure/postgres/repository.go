// Package postgres provides the PostgreSQL-backed DefectRepository.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/domain"
)

const (
	foreignKeyViolation = "23503"

	storyProjectFKey            = "defects_story_project_fkey"
	sprintProjectFKey           = "defects_sprint_project_fkey"
	resolutionSprintProjectFKey = "defects_resolution_sprint_project_fkey"

	projectExistsQuery = "SELECT EXISTS (SELECT 1 FROM projects WHERE id = $1)"
	// The reference checks lock the row with FOR KEY SHARE, the lock the foreign keys take
	// on insert, so a concurrent delete cannot invalidate a check before the INSERT.
	storyInProjectQuery  = "SELECT 1 FROM stories WHERE id = $1 AND project_id = $2 FOR KEY SHARE"
	sprintInProjectQuery = "SELECT 1 FROM sprints WHERE id = $1 AND project_id = $2 FOR KEY SHARE"

	insertDefectQuery = `
		INSERT INTO defects (id, project_id, story_id, sprint_id, description, severity, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	lockDefectForUpdateQuery = `
		SELECT id, project_id, story_id, sprint_id, description, severity, status
		FROM defects WHERE id = $1 AND project_id = $2 FOR UPDATE
	`
	resolutionSprintChronologyQuery = `
		SELECT resolution.seq, detection.seq
		FROM sprints AS resolution
		JOIN sprints AS detection ON detection.id = $3 AND detection.project_id = $2
		WHERE resolution.id = $1 AND resolution.project_id = $2
		FOR KEY SHARE OF resolution, detection
	`
	updateDefectQuery = `
		UPDATE defects SET status = $3, resolution_sprint_id = $4
		WHERE id = $1 AND project_id = $2
	`
	listDefectsByProjectQuery = `
		SELECT id, project_id, story_id, sprint_id, description, severity, status, resolution_sprint_id
		FROM defects WHERE project_id = $1 ORDER BY seq
	`
)

// PostgresDefectRepository verifies the references of a defect and persists it within one
// transaction.
type PostgresDefectRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresDefectRepository creates a PostgreSQL-backed defect repository.
func NewPostgresDefectRepository(pool *pgxpool.Pool) *PostgresDefectRepository {
	return &PostgresDefectRepository{pool: pool}
}

// Create verifies, in order, that the project exists, that the story belongs to it and that
// the detection sprint belongs to it; then it inserts the record. Everything happens in one
// transaction: any failure leaves no record persisted. The order fixes which error is
// reported when several references are missing.
func (r *PostgresDefectRepository) Create(ctx context.Context, defect domain.Defect) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var projectExists bool
	if err := tx.QueryRow(ctx, projectExistsQuery, defect.ProjectID).Scan(&projectExists); err != nil {
		return err
	}
	if !projectExists {
		return application.ErrProjectNotFound
	}

	if err := requireRow(ctx, tx, application.ErrStoryNotFound, storyInProjectQuery, defect.StoryID, defect.ProjectID); err != nil {
		return err
	}
	if err := requireRow(ctx, tx, application.ErrSprintNotFound, sprintInProjectQuery, defect.SprintID, defect.ProjectID); err != nil {
		return err
	}

	_, err = tx.Exec(ctx, insertDefectQuery,
		defect.ID, defect.ProjectID, defect.StoryID, defect.SprintID, defect.Description, defect.Severity, defect.Status)
	if err != nil {
		return MapConstraintError(err)
	}

	return tx.Commit(ctx)
}

// Update locks the defect and related Sprint rows, validates project ownership and creation
// order, then writes lifecycle state in one transaction. Any validation failure leaves the
// stored defect unchanged.
func (r *PostgresDefectRepository) Update(ctx context.Context, command application.UpdateDefectCommand) (domain.Defect, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Defect{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var defect domain.Defect
	if err := tx.QueryRow(ctx, lockDefectForUpdateQuery, command.DefectID, command.ProjectID).Scan(
		&defect.ID, &defect.ProjectID, &defect.StoryID, &defect.SprintID,
		&defect.Description, &defect.Severity, &defect.Status,
	); errors.Is(err, pgx.ErrNoRows) {
		return domain.Defect{}, application.ErrDefectNotFound
	} else if err != nil {
		return domain.Defect{}, err
	}

	var resolutionSprint any
	if command.ResolutionSprintID != "" {
		var resolutionSequence, detectionSequence int64
		if err := tx.QueryRow(ctx, resolutionSprintChronologyQuery, command.ResolutionSprintID, command.ProjectID, defect.SprintID).Scan(
			&resolutionSequence, &detectionSequence,
		); errors.Is(err, pgx.ErrNoRows) {
			return domain.Defect{}, application.ErrResolutionSprintNotFound
		} else if err != nil {
			return domain.Defect{}, err
		}
		if resolutionSequence < detectionSequence {
			return domain.Defect{}, application.ErrResolutionSprintBeforeDetection
		}
		resolutionSprint = command.ResolutionSprintID
	}

	if _, err := tx.Exec(ctx, updateDefectQuery, command.DefectID, command.ProjectID, command.Status, resolutionSprint); err != nil {
		return domain.Defect{}, MapConstraintError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Defect{}, err
	}
	defect.Status = command.Status
	defect.ResolutionSprintID = command.ResolutionSprintID
	return defect, nil
}

// ListByProject returns defects for an existing project in registration order. The empty
// resolution Sprint is represented by a NULL database value and remains empty in the domain.
func (r *PostgresDefectRepository) ListByProject(ctx context.Context, projectID string) ([]domain.Defect, error) {
	var projectExists bool
	if err := r.pool.QueryRow(ctx, projectExistsQuery, projectID).Scan(&projectExists); err != nil {
		return nil, err
	}
	if !projectExists {
		return nil, application.ErrProjectNotFound
	}

	rows, err := r.pool.Query(ctx, listDefectsByProjectQuery, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	defects := make([]domain.Defect, 0)
	for rows.Next() {
		var defect domain.Defect
		var resolutionSprintID pgtype.Text
		if err := rows.Scan(
			&defect.ID, &defect.ProjectID, &defect.StoryID, &defect.SprintID,
			&defect.Description, &defect.Severity, &defect.Status, &resolutionSprintID,
		); err != nil {
			return nil, err
		}
		if resolutionSprintID.Valid {
			defect.ResolutionSprintID = resolutionSprintID.String
		}
		defects = append(defects, defect)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return defects, nil
}

// requireRow runs a single-row locking query and returns missing when it matches no row.
func requireRow(ctx context.Context, tx pgx.Tx, missing error, query string, args ...any) error {
	var row int
	err := tx.QueryRow(ctx, query, args...).Scan(&row)
	if errors.Is(err, pgx.ErrNoRows) {
		return missing
	}
	return err
}

// MapConstraintError translates a foreign key violation (23503) of defects into the domain
// error of the violated reference. The transactional checks make this path unreachable
// except when a parent row disappears between the check and the INSERT. Any other error,
// including CHECK violations, is returned unchanged.
func MapConstraintError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != foreignKeyViolation {
		return err
	}
	switch pgErr.ConstraintName {
	case storyProjectFKey:
		return application.ErrStoryNotFound
	case sprintProjectFKey:
		return application.ErrSprintNotFound
	case resolutionSprintProjectFKey:
		return application.ErrResolutionSprintNotFound
	default:
		return err
	}
}

var _ application.DefectRepository = (*PostgresDefectRepository)(nil)
var _ application.DefectLister = (*PostgresDefectRepository)(nil)
