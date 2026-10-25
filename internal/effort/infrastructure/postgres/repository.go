// Package postgres provides the PostgreSQL-backed EffortRepository.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/valerubio7/software-metrics-and-estimation/internal/effort/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/effort/domain"
)

const (
	foreignKeyViolation = "23503"

	storyProjectConstraint     = "effort_records_story_project_fkey"
	memberProjectConstraint    = "effort_records_member_project_fkey"
	taskStoryProjectConstraint = "effort_records_task_story_project_fkey"

	projectExistsQuery = "SELECT EXISTS (SELECT 1 FROM projects WHERE id = $1)"
	// The reference checks lock the row with FOR KEY SHARE, the lock the foreign keys take on
	// insert, so a concurrent delete cannot invalidate a check before the INSERT.
	storyInProjectQuery  = "SELECT 1 FROM stories WHERE id = $1 AND project_id = $2 FOR KEY SHARE"
	memberInProjectQuery = "SELECT 1 FROM project_members WHERE id = $1 AND project_id = $2 FOR KEY SHARE"
	taskInStoryQuery     = "SELECT 1 FROM tasks WHERE id = $1 AND story_id = $2 AND project_id = $3 FOR KEY SHARE"

	insertEffortRecordQuery = `
		INSERT INTO effort_records (id, project_id, story_id, task_id, member_id, work_date, description, hours)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
)

// PostgresEffortRepository verifies the references of an effort record and persists it
// within one transaction.
type PostgresEffortRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresEffortRepository creates a PostgreSQL-backed effort repository.
func NewPostgresEffortRepository(pool *pgxpool.Pool) *PostgresEffortRepository {
	return &PostgresEffortRepository{pool: pool}
}

// Create verifies, in order, that the project exists, the story and the member belong to it
// and, when the record targets a task, that the task belongs to that story and project; then
// it inserts the record. Everything happens in one transaction: any failure leaves no record
// persisted. The order fixes which error is reported when several references are missing.
func (r *PostgresEffortRepository) Create(ctx context.Context, record domain.EffortRecord) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var projectExists bool
	if err := tx.QueryRow(ctx, projectExistsQuery, record.ProjectID).Scan(&projectExists); err != nil {
		return err
	}
	if !projectExists {
		return application.ErrProjectNotFound
	}

	if err := requireRow(ctx, tx, application.ErrStoryNotFound, storyInProjectQuery, record.StoryID, record.ProjectID); err != nil {
		return err
	}
	if err := requireRow(ctx, tx, application.ErrMemberNotFound, memberInProjectQuery, record.MemberID, record.ProjectID); err != nil {
		return err
	}
	if record.TaskID != nil {
		if err := requireRow(ctx, tx, application.ErrTaskNotFound, taskInStoryQuery, *record.TaskID, record.StoryID, record.ProjectID); err != nil {
			return err
		}
	}

	_, err = tx.Exec(ctx, insertEffortRecordQuery,
		record.ID, record.ProjectID, record.StoryID, record.TaskID, record.MemberID, record.WorkDate, record.Description, record.Hours)
	if err != nil {
		return MapConstraintError(err)
	}

	return tx.Commit(ctx)
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

// MapConstraintError translates a foreign key violation (23503) of effort_records into the
// domain error of the violated reference. The transactional checks make this path unreachable
// except when a parent row disappears between the check and the INSERT. Any other error,
// including CHECK violations, is returned unchanged.
func MapConstraintError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != foreignKeyViolation {
		return err
	}
	switch pgErr.ConstraintName {
	case storyProjectConstraint:
		return application.ErrStoryNotFound
	case memberProjectConstraint:
		return application.ErrMemberNotFound
	case taskStoryProjectConstraint:
		return application.ErrTaskNotFound
	default:
		return err
	}
}

var _ application.EffortRepository = (*PostgresEffortRepository)(nil)
