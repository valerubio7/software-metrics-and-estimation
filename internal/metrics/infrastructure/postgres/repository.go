// Package postgres provides a consistent PostgreSQL reader for project and Sprint metrics.
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/valerubio7/software-metrics-and-estimation/internal/metrics/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/metrics/domain"
)

// PostgresMetricsReader loads one complete read-only snapshot for a project or Sprint.
type PostgresMetricsReader struct {
	pool *pgxpool.Pool
}

// NewPostgresMetricsReader creates a PostgreSQL-backed metrics reader.
func NewPostgresMetricsReader(pool *pgxpool.Pool) *PostgresMetricsReader {
	return &PostgresMetricsReader{pool: pool}
}

// ReadMetrics loads the selected scope and all source observations in a single
// repeatable-read, read-only transaction. It returns no partial input on failure.
func (r *PostgresMetricsReader) ReadMetrics(ctx context.Context, projectID string, sprintID *string) (domain.Input, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return domain.Input{}, fmt.Errorf("begin metrics snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	var projectExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM projects WHERE id = $1)`, projectID).Scan(&projectExists); err != nil {
		return domain.Input{}, fmt.Errorf("check metrics project: %w", err)
	}
	if !projectExists {
		return domain.Input{}, application.ErrProjectNotFound
	}
	if sprintID != nil {
		var sprintExists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM sprints WHERE id = $1 AND project_id = $2)`, *sprintID, projectID).Scan(&sprintExists); err != nil {
			return domain.Input{}, fmt.Errorf("check metrics Sprint: %w", err)
		}
		if !sprintExists {
			return domain.Input{}, application.ErrSprintNotFound
		}
	}

	input := domain.Input{ProjectID: projectID, SprintID: cloneString(sprintID)}
	input.Stories, err = readStories(ctx, tx, projectID, sprintID)
	if err != nil {
		return domain.Input{}, err
	}
	storyIndexes := make(map[string]int, len(input.Stories))
	for index := range input.Stories {
		storyIndexes[input.Stories[index].ID] = index
	}
	if err := readTasks(ctx, tx, projectID, sprintID, input.Stories, storyIndexes); err != nil {
		return domain.Input{}, err
	}
	if err := readEffortRecords(ctx, tx, projectID, sprintID, input.Stories, storyIndexes); err != nil {
		return domain.Input{}, err
	}
	if err := readDefectCounts(ctx, tx, projectID, sprintID, &input); err != nil {
		return domain.Input{}, err
	}
	if sprintID == nil {
		input.ClosedSprints, err = readClosedSprints(ctx, tx, projectID)
		if err != nil {
			return domain.Input{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Input{}, fmt.Errorf("commit metrics snapshot: %w", err)
	}
	return input, nil
}

func readStories(ctx context.Context, tx pgx.Tx, projectID string, sprintID *string) ([]domain.StoryInput, error) {
	var rows pgx.Rows
	var err error
	if sprintID == nil {
		rows, err = tx.Query(ctx, `
			SELECT s.id::text, s.story_points, s.estimated_hours,
			       EXISTS (
			           SELECT 1 FROM sprint_stories ss
			           WHERE ss.project_id = s.project_id AND ss.story_id = s.id AND ss.completed_at IS NOT NULL
		       ) AS completed
		FROM stories s
		WHERE s.project_id = $1
		ORDER BY s.seq
		`, projectID)
	} else {
		rows, err = tx.Query(ctx, `
			SELECT s.id::text, s.story_points, s.estimated_hours, ss.completed_at IS NOT NULL AS completed
			FROM sprint_stories ss
		JOIN stories s ON s.id = ss.story_id AND s.project_id = ss.project_id
		WHERE ss.project_id = $1 AND ss.sprint_id = $2
		ORDER BY s.seq
		`, projectID, *sprintID)
	}
	if err != nil {
		return nil, fmt.Errorf("read metrics stories: %w", err)
	}
	defer rows.Close()

	stories := make([]domain.StoryInput, 0)
	for rows.Next() {
		var story domain.StoryInput
		if err := rows.Scan(&story.ID, &story.StoryPoints, &story.EstimatedHours, &story.Completed); err != nil {
			return nil, fmt.Errorf("scan metrics story: %w", err)
		}
		stories = append(stories, story)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metrics stories: %w", err)
	}
	return stories, nil
}

func readTasks(ctx context.Context, tx pgx.Tx, projectID string, sprintID *string, stories []domain.StoryInput, indexes map[string]int) error {
	query := `
		SELECT id::text, story_id::text, sprint_id::text, estimated_hours
		FROM tasks WHERE project_id = $1 ORDER BY seq
	`
	args := []any{projectID}
	if sprintID != nil {
		query = `
			SELECT id::text, story_id::text, sprint_id::text, estimated_hours
			FROM tasks WHERE project_id = $1 AND sprint_id = $2 ORDER BY seq
		`
		args = append(args, *sprintID)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("read metrics tasks: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var task domain.TaskInput
		var storyID string
		if err := rows.Scan(&task.ID, &storyID, &task.SprintID, &task.EstimatedHours); err != nil {
			return fmt.Errorf("scan metrics task: %w", err)
		}
		if index, ok := indexes[storyID]; ok {
			stories[index].Tasks = append(stories[index].Tasks, task)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate metrics tasks: %w", err)
	}
	return nil
}

func readEffortRecords(ctx context.Context, tx pgx.Tx, projectID string, sprintID *string, stories []domain.StoryInput, indexes map[string]int) error {
	query := `
		SELECT er.story_id::text, er.task_id::text, task.sprint_id::text, er.hours::double precision
		FROM effort_records er
		LEFT JOIN tasks task ON task.id = er.task_id AND task.project_id = er.project_id AND task.story_id = er.story_id
		WHERE er.project_id = $1
		ORDER BY er.seq
	`
	args := []any{projectID}
	if sprintID != nil {
		query = `
			SELECT er.story_id::text, er.task_id::text, task.sprint_id::text, er.hours::double precision
			FROM effort_records er
			JOIN sprint_stories ss ON ss.project_id = er.project_id AND ss.story_id = er.story_id AND ss.sprint_id = $2
			LEFT JOIN tasks task ON task.id = er.task_id AND task.project_id = er.project_id AND task.story_id = er.story_id
			WHERE er.project_id = $1 AND (er.task_id IS NULL OR task.sprint_id = $2)
			ORDER BY er.seq
		`
		args = append(args, *sprintID)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("read metrics effort: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var storyID string
		var record domain.EffortRecordInput
		if err := rows.Scan(&storyID, &record.TaskID, &record.SprintID, &record.Hours); err != nil {
			return fmt.Errorf("scan metrics effort: %w", err)
		}
		if index, ok := indexes[storyID]; ok {
			stories[index].EffortRecords = append(stories[index].EffortRecords, record)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate metrics effort: %w", err)
	}
	return nil
}

func readDefectCounts(ctx context.Context, tx pgx.Tx, projectID string, sprintID *string, input *domain.Input) error {
	query := `
		SELECT COUNT(*)::integer,
		       COUNT(*) FILTER (WHERE resolution_sprint_id IS NOT NULL)::integer
		FROM defects WHERE project_id = $1
	`
	args := []any{projectID}
	if sprintID != nil {
		query = `
			SELECT
			    (SELECT COUNT(*) FROM defects WHERE project_id = $1 AND sprint_id = $2)::integer,
			    (SELECT COUNT(*) FROM defects WHERE project_id = $1 AND resolution_sprint_id = $2)::integer
		`
		args = append(args, *sprintID)
	}
	if err := tx.QueryRow(ctx, query, args...).Scan(&input.DefectsDetected, &input.DefectsResolved); err != nil {
		return fmt.Errorf("read metrics defect counts: %w", err)
	}
	return nil
}

func readClosedSprints(ctx context.Context, tx pgx.Tx, projectID string) ([]domain.ClosedSprintInput, error) {
	rows, err := tx.Query(ctx, `
		SELECT sp.id::text, ss.completed_at IS NOT NULL, st.story_points
		FROM sprints sp
		LEFT JOIN sprint_stories ss ON ss.sprint_id = sp.id AND ss.project_id = sp.project_id
		LEFT JOIN stories st ON st.id = ss.story_id AND st.project_id = ss.project_id
		WHERE sp.project_id = $1 AND sp.is_closed = true
		ORDER BY sp.seq, st.seq
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("read closed metrics Sprints: %w", err)
	}
	defer rows.Close()

	closedSprints := make([]domain.ClosedSprintInput, 0)
	currentSprintID := ""
	for rows.Next() {
		var sprintID string
		var completed bool
		var points *int
		if err := rows.Scan(&sprintID, &completed, &points); err != nil {
			return nil, fmt.Errorf("scan closed metrics Sprint: %w", err)
		}
		if sprintID != currentSprintID {
			closedSprints = append(closedSprints, domain.ClosedSprintInput{CompletedStoryPoints: make([]*int, 0)})
			currentSprintID = sprintID
		}
		if completed {
			closedSprints[len(closedSprints)-1].CompletedStoryPoints = append(closedSprints[len(closedSprints)-1].CompletedStoryPoints, points)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate closed metrics Sprints: %w", err)
	}
	return closedSprints, nil
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

var _ application.MetricsReader = (*PostgresMetricsReader)(nil)
