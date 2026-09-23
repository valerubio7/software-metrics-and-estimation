package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valerubio7/software-metrics-estimation/internal/project/domain"
)

// PostgresProjectRepository persists projects in PostgreSQL.
type PostgresProjectRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresProjectRepository creates a PostgreSQL-backed project repository.
func NewPostgresProjectRepository(pool *pgxpool.Pool) *PostgresProjectRepository {
	return &PostgresProjectRepository{pool: pool}
}

// Create stores a valid project using a parameterized INSERT statement.
func (r *PostgresProjectRepository) Create(ctx context.Context, project domain.Project) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO projects (id, name, start_date, planned_finish_date)
		VALUES ($1, $2, $3, $4)
	`, project.ID, project.Name, project.StartDate, project.PlannedFinishDate)
	return err
}
