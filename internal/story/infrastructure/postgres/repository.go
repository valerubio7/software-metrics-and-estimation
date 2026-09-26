package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
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

// Update replaces the editable content of one story scoped by id and project_id and
// returns the stored row. Zero affected rows means the story does not exist under that
// project; no other error is reinterpreted.
func (r *PostgresStoryRepository) Update(ctx context.Context, story domain.Story) (domain.Story, error) {
	var stored domain.Story
	err := r.pool.QueryRow(ctx, `
		UPDATE stories
		SET title = $3, description = $4, priority = $5, status = $6,
		    acceptance_criteria = $7, estimated_hours = $8
		WHERE id = $1 AND project_id = $2
		RETURNING id, project_id, title, description, priority, status,
		          story_points, acceptance_criteria, estimated_hours
	`, story.ID, story.ProjectID, story.Title, story.Description, story.Priority, story.Status,
		story.AcceptanceCriteria, story.EstimatedHours).Scan(
		&stored.ID, &stored.ProjectID, &stored.Title, &stored.Description, &stored.Priority, &stored.Status,
		&stored.StoryPoints, &stored.AcceptanceCriteria, &stored.EstimatedHours)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Story{}, application.ErrStoryNotFound
	}
	if err != nil {
		return domain.Story{}, err
	}
	return stored, nil
}

var (
	_ application.StoryRepository = (*PostgresStoryRepository)(nil)
	_ application.StoryUpdater    = (*PostgresStoryRepository)(nil)
)
