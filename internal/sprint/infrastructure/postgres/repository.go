package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/domain"
)

// PostgresSprintRepository persists sprints with a database-enforced project association.
type PostgresSprintRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresSprintRepository creates a PostgreSQL-backed sprint repository.
func NewPostgresSprintRepository(pool *pgxpool.Pool) *PostgresSprintRepository {
	return &PostgresSprintRepository{pool: pool}
}

// CloseSprint atomically closes one project-scoped sprint under the same row lock
// used by assignment and completion, preserving all associated records.
func (r *PostgresSprintRepository) CloseSprint(ctx context.Context, projectID, sprintID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var projectExists bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM projects WHERE id = $1)", projectID).Scan(&projectExists); err != nil {
		return err
	}
	if !projectExists {
		return application.ErrProjectNotFound
	}

	var closed bool
	err = tx.QueryRow(ctx, "SELECT is_closed FROM sprints WHERE id = $1 AND project_id = $2 FOR UPDATE", sprintID, projectID).Scan(&closed)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrSprintNotFound
	}
	if err != nil {
		return err
	}
	if closed {
		return application.ErrSprintClosed
	}
	if _, err := tx.Exec(ctx, "UPDATE sprints SET is_closed = true WHERE id = $1 AND project_id = $2", sprintID, projectID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ application.SprintCloser = (*PostgresSprintRepository)(nil)

// Create inserts one sprint; only the named project FK identifies a missing project.
func (r *PostgresSprintRepository) Create(ctx context.Context, sprint domain.Sprint) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO sprints (id, project_id, sprint_goal)
		VALUES ($1, $2, $3)
	`, sprint.ID, sprint.ProjectID, sprint.SprintGoal)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" && pgErr.ConstraintName == "sprints_project_id_fkey" {
			return application.ErrProjectNotFound
		}
	}
	return err
}
