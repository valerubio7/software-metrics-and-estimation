package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/project/domain"
)

// PostgresProjectRepository persists projects in PostgreSQL.
type PostgresProjectRepository struct {
	pool *pgxpool.Pool
}

// Update replaces only the basic project fields and reports an unknown ID.
func (r *PostgresProjectRepository) Update(ctx context.Context, project domain.Project) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE projects
		SET name = $2, start_date = $3, planned_finish_date = $4
		WHERE id = $1
	`, project.ID, project.Name, project.StartDate, project.PlannedFinishDate)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return application.ErrProjectNotFound
	}
	return nil
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
