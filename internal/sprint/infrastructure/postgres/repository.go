package postgres

import (
	"context"
	"errors"
	"time"

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

var _ application.SprintHistoryReader = (*PostgresSprintRepository)(nil)
var _ application.OpenSprintReader = (*PostgresSprintRepository)(nil)

// ListOpenByProject returns open sprints in creation order and distinguishes an empty project from a missing one.
func (r *PostgresSprintRepository) ListOpenByProject(ctx context.Context, projectID string) ([]domain.Sprint, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT sp.id::text, sp.project_id::text, sp.sprint_goal
		FROM projects p
		LEFT JOIN sprints sp ON sp.project_id = p.id AND NOT sp.is_closed
		WHERE p.id = $1
		ORDER BY sp.seq ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	found := false
	sprints := make([]domain.Sprint, 0)
	for rows.Next() {
		var id, projectID, goal *string
		if err := rows.Scan(&id, &projectID, &goal); err != nil {
			return nil, err
		}
		found = true
		if id != nil {
			sprints = append(sprints, domain.Sprint{ID: *id, ProjectID: *projectID, SprintGoal: *goal})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if !found {
		return nil, application.ErrProjectNotFound
	}
	return sprints, nil
}

// historyRow is one row of the history query. Every column is nullable because the outer joins yield a
// project row without sprints, and a sprint row without stories.
type historyRow struct {
	sprintID    *string
	projectID   *string
	sprintGoal  *string
	storyID     *string
	title       *string
	priority    *string
	storyPoints *int
	completedAt *time.Time
}

// ListClosedByProject reads the closed sprints of a project and their planned stories in one statement,
// so that project existence and the history come from the same snapshot. The is_closed filter lives in
// the ON clause: in WHERE it would drop the project row of a project without closed sprints and make it
// indistinguishable from a missing project. Sprints are ordered from the most recently created and stories
// by creation, both through their database-assigned sequence.
func (r *PostgresSprintRepository) ListClosedByProject(ctx context.Context, projectID string) ([]domain.ClosedSprint, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT sp.id::text, sp.project_id::text, sp.sprint_goal,
		       st.id::text, st.title, st.priority, st.story_points, ss.completed_at
		FROM projects p
		LEFT JOIN sprints sp ON sp.project_id = p.id AND sp.is_closed
		LEFT JOIN sprint_stories ss ON ss.sprint_id = sp.id AND ss.project_id = sp.project_id
		LEFT JOIN stories st ON st.id = ss.story_id AND st.project_id = ss.project_id
		WHERE p.id = $1
		ORDER BY sp.seq DESC, st.seq ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	found := false
	sprints := make([]domain.ClosedSprint, 0)
	for rows.Next() {
		var row historyRow
		if err := rows.Scan(&row.sprintID, &row.projectID, &row.sprintGoal,
			&row.storyID, &row.title, &row.priority, &row.storyPoints, &row.completedAt); err != nil {
			return nil, err
		}
		found = true
		if row.sprintID == nil {
			continue
		}
		if len(sprints) == 0 || sprints[len(sprints)-1].ID != *row.sprintID {
			sprints = append(sprints, domain.ClosedSprint{
				ID:         *row.sprintID,
				ProjectID:  *row.projectID,
				SprintGoal: *row.sprintGoal,
				Stories:    make([]domain.PlannedStory, 0),
			})
		}
		if row.storyID == nil {
			continue
		}
		current := &sprints[len(sprints)-1]
		current.Stories = append(current.Stories, domain.PlannedStory{
			ID:          *row.storyID,
			Title:       *row.title,
			Priority:    *row.priority,
			StoryPoints: row.storyPoints,
			CompletedAt: row.completedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if !found {
		return nil, application.ErrProjectNotFound
	}
	return sprints, nil
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
