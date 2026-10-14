package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/sprint/domain"
	sprintpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/sprint/infrastructure/postgres"
)

const (
	historyOtherProjectID   = "8756a2df-783d-4070-a236-b4284356ed50"
	historyMissingProjectID = "9b6847be-c44c-47e8-a91d-69aa2874a80f"
)

func TestListClosedByProjectReturnsNoSprintsWithoutError(t *testing.T) {
	t.Run("only open sprint", func(t *testing.T) {
		pool := sprintDatabase(t)
		insertProject(t, pool)
		createListSprint(t, pool, projectID, historyID(1), "Open sprint")

		sprints, err := sprintpostgres.NewPostgresSprintRepository(pool).ListClosedByProject(context.Background(), projectID)

		if err != nil || len(sprints) != 0 {
			t.Fatalf("ListClosedByProject(only open sprint) = (%v, %v), want no sprints and nil error", sprints, err)
		}
	})
	t.Run("project without sprints", func(t *testing.T) {
		pool := sprintDatabase(t)
		insertProject(t, pool)

		sprints, err := sprintpostgres.NewPostgresSprintRepository(pool).ListClosedByProject(context.Background(), projectID)

		if err != nil || len(sprints) != 0 {
			t.Fatalf("ListClosedByProject(project without sprints) = (%v, %v), want no sprints and nil error", sprints, err)
		}
	})
	t.Run("missing project", func(t *testing.T) {
		pool := sprintDatabase(t)

		sprints, err := sprintpostgres.NewPostgresSprintRepository(pool).ListClosedByProject(context.Background(), historyMissingProjectID)

		if !errors.Is(err, application.ErrProjectNotFound) || sprints != nil {
			t.Fatalf("ListClosedByProject(missing project) = (%v, %v), want (nil, ErrProjectNotFound)", sprints, err)
		}
	})
}

func TestListClosedByProjectOrdersSprintsByCreationDescendingNotByClosure(t *testing.T) {
	pool := sprintDatabase(t)
	insertProject(t, pool)
	a, b, c := historyID(1), historyID(2), historyID(3)
	createListSprint(t, pool, projectID, a, "A")
	createListSprint(t, pool, projectID, b, "B")
	createListSprint(t, pool, projectID, c, "C")
	for _, id := range []string{b, c, a} {
		closeListSprint(t, pool, projectID, id)
	}

	sprints, err := sprintpostgres.NewPostgresSprintRepository(pool).ListClosedByProject(context.Background(), projectID)
	if err != nil {
		t.Fatalf("ListClosedByProject() error = %v", err)
	}

	if got, want := strings.Join(sprintIDs(sprints), ","), strings.Join([]string{c, b, a}, ","); got != want {
		t.Errorf("sprint order = %s, want most recently created first %s", got, want)
	}
}

func TestListClosedByProjectOrdersStoriesByCreationAscendingNotByAssignment(t *testing.T) {
	pool := sprintDatabase(t)
	insertProject(t, pool)
	s1, s2, s3 := historyID(11), historyID(12), historyID(13)
	for index, id := range []string{s1, s2, s3} {
		seedListStory(t, pool, projectID, id, fmt.Sprintf("Story %d", index+1), "media", nil)
	}
	createListSprint(t, pool, projectID, sprintID, "Planned")
	for _, id := range []string{s3, s1, s2} {
		assignListStory(t, pool, projectID, sprintID, id, nil)
	}
	closeListSprint(t, pool, projectID, sprintID)

	sprints, err := sprintpostgres.NewPostgresSprintRepository(pool).ListClosedByProject(context.Background(), projectID)
	if err != nil || len(sprints) != 1 {
		t.Fatalf("ListClosedByProject() = (%v, %v), want one sprint", sprints, err)
	}

	if got, want := strings.Join(storyIDs(sprints[0].Stories), ","), strings.Join([]string{s1, s2, s3}, ","); got != want {
		t.Errorf("story order = %s, want creation order %s", got, want)
	}
}

func TestListClosedByProjectReportsCompletionPerSprint(t *testing.T) {
	pool := sprintDatabase(t)
	insertProject(t, pool)
	story := historyID(21)
	seedListStory(t, pool, projectID, story, "Carried over", "alta", nil)
	older, newer := historyID(1), historyID(2)
	createListSprint(t, pool, projectID, older, "Older")
	createListSprint(t, pool, projectID, newer, "Newer")
	completedAt := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	assignListStory(t, pool, projectID, older, story, nil)
	assignListStory(t, pool, projectID, newer, story, &completedAt)
	closeListSprint(t, pool, projectID, older)
	closeListSprint(t, pool, projectID, newer)
	if _, err := pool.Exec(context.Background(), "UPDATE stories SET status = 'completada' WHERE id = $1", story); err != nil {
		t.Fatalf("mark story completed: %v", err)
	}

	sprints, err := sprintpostgres.NewPostgresSprintRepository(pool).ListClosedByProject(context.Background(), projectID)
	if err != nil || len(sprints) != 2 {
		t.Fatalf("ListClosedByProject() = (%v, %v), want two sprints", sprints, err)
	}

	if sprints[0].ID != newer || len(sprints[0].Stories) != 1 || sprints[0].Stories[0].CompletedAt == nil || !sprints[0].Stories[0].CompletedAt.Equal(completedAt) {
		t.Errorf("newer sprint stories = %+v, want the story completed at %v", sprints[0].Stories, completedAt)
	}
	if sprints[1].ID != older || len(sprints[1].Stories) != 1 || sprints[1].Stories[0].CompletedAt != nil {
		t.Errorf("older sprint stories = %+v, want the story without completion despite its completed status", sprints[1].Stories)
	}
}

func TestListClosedByProjectKeepsClosedSprintWithoutStories(t *testing.T) {
	pool := sprintDatabase(t)
	insertProject(t, pool)
	empty, withStory := historyID(1), historyID(2)
	createListSprint(t, pool, projectID, empty, "Empty")
	createListSprint(t, pool, projectID, withStory, "Planned")
	seedListStory(t, pool, projectID, historyID(31), "Planned story", "baja", nil)
	assignListStory(t, pool, projectID, withStory, historyID(31), nil)
	closeListSprint(t, pool, projectID, empty)
	closeListSprint(t, pool, projectID, withStory)

	sprints, err := sprintpostgres.NewPostgresSprintRepository(pool).ListClosedByProject(context.Background(), projectID)
	if err != nil || len(sprints) != 2 {
		t.Fatalf("ListClosedByProject() = (%v, %v), want two sprints", sprints, err)
	}

	if sprints[0].ID != withStory || len(sprints[0].Stories) != 1 {
		t.Errorf("first sprint = %+v, want the sprint with one story", sprints[0])
	}
	if sprints[1].ID != empty || sprints[1].Stories == nil || len(sprints[1].Stories) != 0 {
		t.Errorf("second sprint stories = %#v, want an empty non-nil list", sprints[1].Stories)
	}
}

func TestListClosedByProjectIsolatesProjects(t *testing.T) {
	pool := sprintDatabase(t)
	insertProject(t, pool)
	insertListProject(t, pool, historyOtherProjectID, "Other")
	own, foreign := historyID(1), historyID(2)
	createListSprint(t, pool, projectID, own, "Own")
	createListSprint(t, pool, historyOtherProjectID, foreign, "Foreign")
	seedListStory(t, pool, historyOtherProjectID, historyID(41), "Foreign story", "alta", nil)
	assignListStory(t, pool, historyOtherProjectID, foreign, historyID(41), nil)
	closeListSprint(t, pool, projectID, own)
	closeListSprint(t, pool, historyOtherProjectID, foreign)

	sprints, err := sprintpostgres.NewPostgresSprintRepository(pool).ListClosedByProject(context.Background(), projectID)
	if err != nil {
		t.Fatalf("ListClosedByProject() error = %v", err)
	}

	if len(sprints) != 1 || sprints[0].ID != own || sprints[0].ProjectID != projectID || len(sprints[0].Stories) != 0 {
		t.Errorf("sprints = %+v, want only the queried project's sprint without foreign stories", sprints)
	}
}

func TestListClosedByProjectMapsStoryColumns(t *testing.T) {
	pool := sprintDatabase(t)
	insertProject(t, pool)
	five := 5
	withPoints, withoutPoints := historyID(51), historyID(52)
	seedListStory(t, pool, projectID, withPoints, "Estimated", "alta", &five)
	seedListStory(t, pool, projectID, withoutPoints, "Unestimated", "baja", nil)
	createListSprint(t, pool, projectID, sprintID, "Mapped goal")
	completedAt := time.Date(2026, 10, 12, 8, 30, 15, 0, time.UTC)
	assignListStory(t, pool, projectID, sprintID, withPoints, &completedAt)
	assignListStory(t, pool, projectID, sprintID, withoutPoints, nil)
	closeListSprint(t, pool, projectID, sprintID)

	sprints, err := sprintpostgres.NewPostgresSprintRepository(pool).ListClosedByProject(context.Background(), projectID)
	if err != nil || len(sprints) != 1 || len(sprints[0].Stories) != 2 {
		t.Fatalf("ListClosedByProject() = (%+v, %v), want one sprint with two stories", sprints, err)
	}

	sprint := sprints[0]
	if sprint.ID != sprintID || sprint.ProjectID != projectID || sprint.SprintGoal != "Mapped goal" {
		t.Errorf("sprint = (%q, %q, %q), want (%q, %q, %q)", sprint.ID, sprint.ProjectID, sprint.SprintGoal, sprintID, projectID, "Mapped goal")
	}
	first, second := sprint.Stories[0], sprint.Stories[1]
	if first.ID != withPoints || first.Title != "Estimated" || first.Priority != "alta" || first.StoryPoints == nil || *first.StoryPoints != 5 ||
		first.CompletedAt == nil || !first.CompletedAt.Equal(completedAt) {
		t.Errorf("first story = %+v, want estimated story with 5 points completed at %v", first, completedAt)
	}
	if second.ID != withoutPoints || second.Title != "Unestimated" || second.Priority != "baja" || second.StoryPoints != nil || second.CompletedAt != nil {
		t.Errorf("second story = %+v, want unestimated story without completion", second)
	}
}

func TestListClosedByProjectDoesNotWrite(t *testing.T) {
	pool := sprintDatabase(t)
	insertProject(t, pool)
	insertListProject(t, pool, historyOtherProjectID, "Empty")
	five := 5
	completedAt := time.Date(2026, 10, 12, 8, 30, 15, 0, time.UTC)
	seedListStory(t, pool, projectID, historyID(61), "Story", "media", &five)
	createListSprint(t, pool, projectID, historyID(1), "Closed")
	createListSprint(t, pool, projectID, historyID(2), "Open")
	assignListStory(t, pool, projectID, historyID(1), historyID(61), &completedAt)
	closeListSprint(t, pool, projectID, historyID(1))
	_, err := pool.Exec(context.Background(),
		"INSERT INTO tasks (id, project_id, sprint_id, story_id, title, estimated_hours) VALUES ($1, $2, $3, $4, 'Task', 1.5)",
		historyID(71), projectID, historyID(1), historyID(61))
	if err != nil {
		t.Fatalf("seed task: %v", err)
	}
	before := readOnlySnapshot(t, pool)
	repository := sprintpostgres.NewPostgresSprintRepository(pool)

	for _, id := range []string{projectID, historyOtherProjectID, historyMissingProjectID, projectID, projectID} {
		_, _ = repository.ListClosedByProject(context.Background(), id)
	}

	if after := readOnlySnapshot(t, pool); after != before {
		t.Errorf("listing changed stored data (including row versions):\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestListClosedByProjectDoesNotReinterpretDatabaseErrors(t *testing.T) {
	pool := sprintDatabase(t)
	insertProject(t, pool)
	createListSprint(t, pool, projectID, historyID(1), "Closed")
	closeListSprint(t, pool, projectID, historyID(1))
	if _, err := pool.Exec(context.Background(), "ALTER TABLE sprints RENAME COLUMN seq TO seq_renamed"); err != nil {
		t.Fatalf("rename sprints.seq: %v", err)
	}

	sprints, err := sprintpostgres.NewPostgresSprintRepository(pool).ListClosedByProject(context.Background(), projectID)

	if err == nil || errors.Is(err, application.ErrProjectNotFound) || sprints != nil {
		t.Fatalf("ListClosedByProject(broken schema) = (%v, %v), want (nil, database error other than ErrProjectNotFound)", sprints, err)
	}
}

func TestListClosedByProjectRepeatedQueriesAreIdentical(t *testing.T) {
	pool := sprintDatabase(t)
	insertProject(t, pool)
	five := 5
	completedAt := time.Date(2026, 10, 12, 8, 30, 15, 0, time.UTC)
	seedListStory(t, pool, projectID, historyID(81), "Shared story", "alta", nil)
	for index := 1; index <= 3; index++ {
		seedListStory(t, pool, projectID, historyID(81+index), fmt.Sprintf("Story %d", index), "media", &five)
		createListSprint(t, pool, projectID, historyID(index), fmt.Sprintf("Sprint %d", index))
		assignListStory(t, pool, projectID, historyID(index), historyID(81), nil)
		assignListStory(t, pool, projectID, historyID(index), historyID(81+index), &completedAt)
		closeListSprint(t, pool, projectID, historyID(index))
	}
	repository := sprintpostgres.NewPostgresSprintRepository(pool)

	first, err := repository.ListClosedByProject(context.Background(), projectID)
	if err != nil || len(first) != 3 {
		t.Fatalf("first ListClosedByProject() = (%v, %v), want three sprints", first, err)
	}

	for attempt := 0; attempt < 3; attempt++ {
		again, err := repository.ListClosedByProject(context.Background(), projectID)
		if err != nil {
			t.Fatalf("repeated ListClosedByProject() error = %v", err)
		}
		if got, want := renderHistory(again), renderHistory(first); got != want {
			t.Fatalf("repeated result differs:\nfirst: %s\nagain: %s", want, got)
		}
	}
}

// renderHistory renders pointer fields by value so that results compare by content.
func renderHistory(sprints []domain.ClosedSprint) string {
	lines := make([]string, 0, len(sprints))
	for _, sprint := range sprints {
		line := fmt.Sprintf("%s|%s|%s", sprint.ID, sprint.ProjectID, sprint.SprintGoal)
		for _, story := range sprint.Stories {
			points, completed := "nil", "nil"
			if story.StoryPoints != nil {
				points = fmt.Sprint(*story.StoryPoints)
			}
			if story.CompletedAt != nil {
				completed = story.CompletedAt.UTC().Format(time.RFC3339Nano)
			}
			line += fmt.Sprintf("|%s,%s,%s,%s,%s", story.ID, story.Title, story.Priority, points, completed)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// readOnlySnapshot renders every row of the tables read by the listing, with its row version (xmin), so
// that any write, even one that stores identical values, changes the snapshot.
func readOnlySnapshot(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	parts := make([]string, 0, 4)
	for _, table := range []string{"sprints", "sprint_stories", "stories", "tasks"} {
		var snapshot string
		query := fmt.Sprintf(`SELECT coalesce(string_agg(t.xmin::text || ' ' || to_jsonb(t)::text, E'\n' ORDER BY to_jsonb(t)::text), '') FROM %s t`, table)
		if err := pool.QueryRow(context.Background(), query).Scan(&snapshot); err != nil {
			t.Fatalf("snapshot %s: %v", table, err)
		}
		parts = append(parts, table+":\n"+snapshot)
	}
	return strings.Join(parts, "\n")
}

func historyID(n int) string {
	return fmt.Sprintf("5a3e0000-0000-4000-8000-%012d", n)
}

func sprintIDs(sprints []domain.ClosedSprint) []string {
	ids := make([]string, 0, len(sprints))
	for _, sprint := range sprints {
		ids = append(ids, sprint.ID)
	}
	return ids
}

func storyIDs(stories []domain.PlannedStory) []string {
	ids := make([]string, 0, len(stories))
	for _, story := range stories {
		ids = append(ids, story.ID)
	}
	return ids
}

func insertListProject(t *testing.T, pool *pgxpool.Pool, id, name string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		"INSERT INTO projects (id, name, start_date, planned_finish_date) VALUES ($1, $2, '2026-03-01', '2026-06-30')", id, name)
	if err != nil {
		t.Fatalf("insert project %s: %v", id, err)
	}
}

func createListSprint(t *testing.T, pool *pgxpool.Pool, project, id, goal string) {
	t.Helper()
	sprint := domain.Sprint{ID: id, ProjectID: project, SprintGoal: goal}
	if err := sprintpostgres.NewPostgresSprintRepository(pool).Create(context.Background(), sprint); err != nil {
		t.Fatalf("Create(%s) error = %v", id, err)
	}
}

func closeListSprint(t *testing.T, pool *pgxpool.Pool, project, id string) {
	t.Helper()
	if err := sprintpostgres.NewPostgresSprintRepository(pool).CloseSprint(context.Background(), project, id); err != nil {
		t.Fatalf("CloseSprint(%s) error = %v", id, err)
	}
}

func seedListStory(t *testing.T, pool *pgxpool.Pool, project, id, title, priority string, points *int) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO stories (id, project_id, title, description, priority, status, story_points, acceptance_criteria)
		VALUES ($1, $2, $3, 'Description', $4, 'pendiente', $5, ARRAY['criterion'])
	`, id, project, title, priority, points)
	if err != nil {
		t.Fatalf("seed story %s: %v", id, err)
	}
}

func assignListStory(t *testing.T, pool *pgxpool.Pool, project, sprint, story string, completedAt *time.Time) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		"INSERT INTO sprint_stories (sprint_id, story_id, project_id, completed_at) VALUES ($1, $2, $3, $4)",
		sprint, story, project, completedAt)
	if err != nil {
		t.Fatalf("assign story %s to sprint %s: %v", story, sprint, err)
	}
}
