// Package postgres provides the PostgreSQL-backed TaskRepository.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valerubio7/software-metrics-and-estimation/internal/task/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/task/domain"
)

// PostgresTaskRepository verifies project/sprint/story membership and persists a task
// batch within one transaction.
type PostgresTaskRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresTaskRepository creates a PostgreSQL-backed task repository.
func NewPostgresTaskRepository(pool *pgxpool.Pool) *PostgresTaskRepository {
	return &PostgresTaskRepository{pool: pool}
}

// CreateForSprintStory verifies, in order, that the project exists, the sprint and the
// story belong to it and the story is assigned to the sprint, then inserts every task of
// the batch. Everything happens in one transaction: any failure leaves no task persisted.
func (r *PostgresTaskRepository) CreateForSprintStory(ctx context.Context, projectID, sprintID, storyID string, tasks []domain.Task) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var projectExists bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM projects WHERE id = $1)", projectID).Scan(&projectExists); err != nil {
		return err
	}
	if !projectExists {
		return application.ErrProjectNotFound
	}

	var sprintExists bool
	err = tx.QueryRow(ctx, "SELECT 1 FROM sprints WHERE id = $1 AND project_id = $2 FOR KEY SHARE", sprintID, projectID).Scan(&sprintExists)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrSprintNotFound
	}
	if err != nil {
		return err
	}

	var storyExists bool
	err = tx.QueryRow(ctx, "SELECT 1 FROM stories WHERE id = $1 AND project_id = $2 FOR KEY SHARE", storyID, projectID).Scan(&storyExists)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrStoryNotFound
	}
	if err != nil {
		return err
	}

	var assigned bool
	err = tx.QueryRow(ctx, "SELECT 1 FROM sprint_stories WHERE sprint_id = $1 AND story_id = $2 FOR KEY SHARE", sprintID, storyID).Scan(&assigned)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrStoryNotInSprint
	}
	if err != nil {
		return err
	}

	for _, task := range tasks {
		_, err := tx.Exec(ctx, `
			INSERT INTO tasks (id, project_id, sprint_id, story_id, title, estimated_hours)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, task.ID, projectID, sprintID, storyID, task.Title, task.EstimatedHours)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" && pgErr.ConstraintName == "tasks_sprint_story_fkey" {
				return application.ErrStoryNotInSprint
			}
			return err
		}
	}

	return tx.Commit(ctx)
}

var _ application.TaskRepository = (*PostgresTaskRepository)(nil)
