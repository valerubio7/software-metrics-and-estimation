package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
)

// PostgresStoryRepository persists stories with a database-enforced project association.
type PostgresStoryRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresStoryRepository creates a PostgreSQL-backed story repository.
func NewPostgresStoryRepository(pool *pgxpool.Pool) *PostgresStoryRepository {
	return &PostgresStoryRepository{pool: pool}
}

// Create inserts one story; only the named project FK identifies a missing project.
func (r *PostgresStoryRepository) Create(ctx context.Context, story domain.Story) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO stories (id, project_id, title, description, priority, status, story_points, acceptance_criteria)
		VALUES ($1, $2, $3, $4, $5, $6, NULL, $7)
	`, story.ID, story.ProjectID, story.Title, story.Description, story.Priority, story.Status, story.AcceptanceCriteria)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" && pgErr.ConstraintName == "stories_project_id_fkey" {
			return application.ErrProjectNotFound
		}
	}
	return err
}
