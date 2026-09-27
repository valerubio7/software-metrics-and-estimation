package postgres

import (
	"context"
	"errors"

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
