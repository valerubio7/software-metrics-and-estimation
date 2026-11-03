// Package postgres provides a consistent PostgreSQL reader for project and Sprint reports.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	metricsdomain "github.com/valerubio7/software-metrics-and-estimation/internal/metrics/domain"
	"github.com/valerubio7/software-metrics-and-estimation/internal/report/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/report/domain"
)

// PostgresReportReader loads one complete read-only snapshot for a project or Sprint.
type PostgresReportReader struct {
	pool *pgxpool.Pool
}

// NewPostgresReportReader creates a PostgreSQL-backed report reader.
func NewPostgresReportReader(pool *pgxpool.Pool) *PostgresReportReader {
	return &PostgresReportReader{pool: pool}
}

// ReadSnapshot loads the selected scope and all report sources in a single repeatable-read,
// read-only transaction. It returns no partial snapshot on failure.
func (r *PostgresReportReader) ReadSnapshot(ctx context.Context, projectID string, sprintID *string) (domain.Snapshot, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("begin report snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	snapshot := domain.Snapshot{}
	snapshot.Project, err = readProject(ctx, tx, projectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Snapshot{}, application.ErrProjectNotFound
	}
	if err != nil {
		return domain.Snapshot{}, err
	}

	if sprintID != nil {
		sprint, sprintErr := readSprint(ctx, tx, projectID, *sprintID)
		if errors.Is(sprintErr, pgx.ErrNoRows) {
			return domain.Snapshot{}, application.ErrSprintNotFound
		}
		if sprintErr != nil {
			return domain.Snapshot{}, sprintErr
		}
		snapshot.Sprint = &sprint
	}

	snapshot.Stories, err = readStories(ctx, tx, projectID, sprintID)
	if err != nil {
		return domain.Snapshot{}, err
	}
	if err := readTasks(ctx, tx, projectID, sprintID, snapshot.Stories); err != nil {
		return domain.Snapshot{}, err
	}
	snapshot.EffortRecords, err = readEffortRecords(ctx, tx, projectID, sprintID)
	if err != nil {
		return domain.Snapshot{}, err
	}
	snapshot.Defects, err = readDefects(ctx, tx, projectID, sprintID)
	if err != nil {
		return domain.Snapshot{}, err
	}
	if sprintID == nil {
		snapshot.ClosedSprints, err = readClosedSprints(ctx, tx, projectID)
		if err != nil {
			return domain.Snapshot{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Snapshot{}, fmt.Errorf("commit report snapshot: %w", err)
	}
	return snapshot, nil
}

func readProject(ctx context.Context, tx pgx.Tx, projectID string) (domain.Project, error) {
	var project domain.Project
	err := tx.QueryRow(ctx, `
		SELECT id::text, name, start_date, planned_finish_date
		FROM projects WHERE id = $1
	`, projectID).Scan(&project.ID, &project.Name, &project.StartDate, &project.PlannedFinishDate)
	if err != nil {
		return domain.Project{}, fmt.Errorf("read report project: %w", err)
	}
	return project, nil
}

func readSprint(ctx context.Context, tx pgx.Tx, projectID, sprintID string) (domain.Sprint, error) {
	var sprint domain.Sprint
	err := tx.QueryRow(ctx, `
		SELECT id::text, sprint_goal, is_closed
		FROM sprints WHERE id = $1 AND project_id = $2
	`, sprintID, projectID).Scan(&sprint.ID, &sprint.Goal, &sprint.Closed)
	if err != nil {
		return domain.Sprint{}, fmt.Errorf("read report Sprint: %w", err)
	}
	return sprint, nil
}

func readStories(ctx context.Context, tx pgx.Tx, projectID string, sprintID *string) ([]domain.Story, error) {
	var rows pgx.Rows
	var err error
	if sprintID == nil {
		rows, err = tx.Query(ctx, `
			SELECT s.id::text, s.title, s.priority, s.status, s.story_points, s.estimated_hours,
			       completed.completed_at
			FROM stories s
			LEFT JOIN LATERAL (
				SELECT MIN(ss.completed_at) AS completed_at
				FROM sprint_stories ss
				WHERE ss.project_id = s.project_id AND ss.story_id = s.id AND ss.completed_at IS NOT NULL
			) completed ON true
			WHERE s.project_id = $1
			ORDER BY s.seq
		`, projectID)
	} else {
		rows, err = tx.Query(ctx, `
			SELECT s.id::text, s.title, s.priority, s.status, s.story_points, s.estimated_hours,
			       ss.completed_at
			FROM sprint_stories ss
			JOIN stories s ON s.id = ss.story_id AND s.project_id = ss.project_id
			WHERE ss.project_id = $1 AND ss.sprint_id = $2
			ORDER BY s.seq
		`, projectID, *sprintID)
	}
	if err != nil {
		return nil, fmt.Errorf("read report stories: %w", err)
	}
	defer rows.Close()

	stories := make([]domain.Story, 0)
	for rows.Next() {
		var story domain.Story
		var completedAt *time.Time
		if err := rows.Scan(
			&story.ID, &story.Title, &story.Priority, &story.Status,
			&story.StoryPoints, &story.EstimatedHours, &completedAt,
		); err != nil {
			return nil, fmt.Errorf("scan report story: %w", err)
		}
		story.Completed = completedAt != nil
		story.CompletedAt = completedAt
		stories = append(stories, story)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate report stories: %w", err)
	}
	return stories, nil
}

func readTasks(ctx context.Context, tx pgx.Tx, projectID string, sprintID *string, stories []domain.Story) error {
	query := `
		SELECT id::text, story_id::text, sprint_id::text, title, estimated_hours
		FROM tasks WHERE project_id = $1 ORDER BY seq
	`
	args := []any{projectID}
	if sprintID != nil {
		query = `
			SELECT id::text, story_id::text, sprint_id::text, title, estimated_hours
			FROM tasks WHERE project_id = $1 AND sprint_id = $2 ORDER BY seq
		`
		args = append(args, *sprintID)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("read report tasks: %w", err)
	}
	defer rows.Close()

	indexes := make(map[string]int, len(stories))
	for index := range stories {
		indexes[stories[index].ID] = index
	}
	for rows.Next() {
		var task domain.Task
		var storyID string
		if err := rows.Scan(&task.ID, &storyID, &task.SprintID, &task.Title, &task.EstimatedHours); err != nil {
			return fmt.Errorf("scan report task: %w", err)
		}
		if index, ok := indexes[storyID]; ok {
			stories[index].Tasks = append(stories[index].Tasks, task)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate report tasks: %w", err)
	}
	return nil
}

func readEffortRecords(ctx context.Context, tx pgx.Tx, projectID string, sprintID *string) ([]domain.EffortRecord, error) {
	query := `
		SELECT er.id::text, er.story_id::text, s.title, er.task_id::text, t.title, t.sprint_id::text,
		       pm.full_name, er.work_date, er.description, er.hours::double precision
		FROM effort_records er
		JOIN stories s ON s.id = er.story_id AND s.project_id = er.project_id
		JOIN project_members pm ON pm.id = er.member_id AND pm.project_id = er.project_id
		LEFT JOIN tasks t ON t.id = er.task_id AND t.project_id = er.project_id AND t.story_id = er.story_id
		WHERE er.project_id = $1
		ORDER BY er.seq
	`
	args := []any{projectID}
	if sprintID != nil {
		query = `
			SELECT er.id::text, er.story_id::text, s.title, er.task_id::text, t.title, t.sprint_id::text,
			       pm.full_name, er.work_date, er.description, er.hours::double precision
			FROM effort_records er
			JOIN sprint_stories ss ON ss.project_id = er.project_id AND ss.story_id = er.story_id AND ss.sprint_id = $2
			JOIN stories s ON s.id = er.story_id AND s.project_id = er.project_id
			JOIN project_members pm ON pm.id = er.member_id AND pm.project_id = er.project_id
			LEFT JOIN tasks t ON t.id = er.task_id AND t.project_id = er.project_id AND t.story_id = er.story_id
			WHERE er.project_id = $1
			ORDER BY er.seq
		`
		args = append(args, *sprintID)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read report effort: %w", err)
	}
	defer rows.Close()

	records := make([]domain.EffortRecord, 0)
	for rows.Next() {
		var record domain.EffortRecord
		if err := rows.Scan(
			&record.ID, &record.StoryID, &record.StoryTitle, &record.TaskID, &record.TaskTitle,
			&record.TaskSprintID, &record.MemberName, &record.WorkDate, &record.Description, &record.Hours,
		); err != nil {
			return nil, fmt.Errorf("scan report effort: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate report effort: %w", err)
	}
	return records, nil
}

func readDefects(ctx context.Context, tx pgx.Tx, projectID string, sprintID *string) ([]domain.Defect, error) {
	query := `
		SELECT d.id::text, d.story_id::text, s.title, d.description, d.severity, d.status,
		       d.sprint_id::text, ds.sprint_goal, d.resolution_sprint_id::text, rs.sprint_goal
		FROM defects d
		JOIN stories s ON s.id = d.story_id AND s.project_id = d.project_id
		JOIN sprints ds ON ds.id = d.sprint_id AND ds.project_id = d.project_id
		LEFT JOIN sprints rs ON rs.id = d.resolution_sprint_id AND rs.project_id = d.project_id
		WHERE d.project_id = $1
		ORDER BY d.seq
	`
	args := []any{projectID}
	if sprintID != nil {
		query = `
			SELECT d.id::text, d.story_id::text, s.title, d.description, d.severity, d.status,
			       d.sprint_id::text, ds.sprint_goal, d.resolution_sprint_id::text, rs.sprint_goal
			FROM defects d
			JOIN stories s ON s.id = d.story_id AND s.project_id = d.project_id
			JOIN sprints ds ON ds.id = d.sprint_id AND ds.project_id = d.project_id
			LEFT JOIN sprints rs ON rs.id = d.resolution_sprint_id AND rs.project_id = d.project_id
			WHERE d.project_id = $1 AND (d.sprint_id = $2 OR d.resolution_sprint_id = $2)
			ORDER BY d.seq
		`
		args = append(args, *sprintID)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read report defects: %w", err)
	}
	defer rows.Close()

	defects := make([]domain.Defect, 0)
	for rows.Next() {
		var defect domain.Defect
		if err := rows.Scan(
			&defect.ID, &defect.StoryID, &defect.StoryTitle, &defect.Description,
			&defect.Severity, &defect.Status, &defect.DetectionSprintID, &defect.DetectionSprintGoal,
			&defect.ResolutionSprintID, &defect.ResolutionSprintGoal,
		); err != nil {
			return nil, fmt.Errorf("scan report defect: %w", err)
		}
		defects = append(defects, defect)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate report defects: %w", err)
	}
	return defects, nil
}

func readClosedSprints(ctx context.Context, tx pgx.Tx, projectID string) ([]metricsdomain.ClosedSprintInput, error) {
	rows, err := tx.Query(ctx, `
		SELECT sp.id::text, ss.completed_at IS NOT NULL, st.story_points
		FROM sprints sp
		LEFT JOIN sprint_stories ss ON ss.sprint_id = sp.id AND ss.project_id = sp.project_id
		LEFT JOIN stories st ON st.id = ss.story_id AND st.project_id = ss.project_id
		WHERE sp.project_id = $1 AND sp.is_closed = true
		ORDER BY sp.seq, st.seq
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("read report closed Sprints: %w", err)
	}
	defer rows.Close()

	closedSprints := make([]metricsdomain.ClosedSprintInput, 0)
	currentSprintID := ""
	for rows.Next() {
		var sprintID string
		var completed bool
		var points *int
		if err := rows.Scan(&sprintID, &completed, &points); err != nil {
			return nil, fmt.Errorf("scan report closed Sprint: %w", err)
		}
		if sprintID != currentSprintID {
			closedSprints = append(closedSprints, metricsdomain.ClosedSprintInput{CompletedStoryPoints: make([]*int, 0)})
			currentSprintID = sprintID
		}
		if completed {
			closedSprints[len(closedSprints)-1].CompletedStoryPoints = append(closedSprints[len(closedSprints)-1].CompletedStoryPoints, points)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate report closed Sprints: %w", err)
	}
	return closedSprints, nil
}

var _ application.SnapshotReader = (*PostgresReportReader)(nil)
