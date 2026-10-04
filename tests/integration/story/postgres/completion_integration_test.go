package postgres_test

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/application"
	storypostgres "github.com/valerubio7/software-metrics-and-estimation/internal/story/infrastructure/postgres"
)

// Completion fixture: project A (projectID) has open sprints S1/S2 and stories H1
// (in S1 and S2), H2 (only S1) and H3 (unassigned); project B has its own sprint and story.
const (
	sprintOneID       = "11111111-1111-4111-8111-111111111111"
	sprintTwoID       = "22222222-2222-4222-8222-222222222222"
	foreignSprintID   = "33333333-3333-4333-8333-333333333333"
	storyOneID        = storyID
	storyTwoID        = "44444444-4444-4444-8444-444444444444"
	storyThreeID      = "55555555-5555-4555-8555-555555555555"
	foreignStoryID    = "66666666-6666-4666-8666-666666666666"
	missingResourceID = "77777777-7777-4777-8777-777777777777"
)

// completionClockMargin absorbs clock skew between the test process and the database host.
const completionClockMargin = 10 * time.Second

// completionFixture seeds the shared scenario; migration 000010 must already be applied
// when the tests call the repository, but not necessarily when the fixture is seeded.
func completionFixture(t *testing.T, pool *pgxpool.Pool) *storypostgres.PostgresStoryRepository {
	t.Helper()
	ctx := context.Background()
	insertProject(t, pool, projectID)
	insertProject(t, pool, otherProjectID)
	for _, sprint := range [][2]string{{sprintOneID, projectID}, {sprintTwoID, projectID}, {foreignSprintID, otherProjectID}} {
		if _, err := pool.Exec(ctx, "INSERT INTO sprints (id, project_id, sprint_goal) VALUES ($1, $2, 'Delivery')", sprint[0], sprint[1]); err != nil {
			t.Fatal(err)
		}
	}
	repo := storypostgres.NewPostgresStoryRepository(pool)
	for _, story := range [][2]string{{storyOneID, projectID}, {storyTwoID, projectID}, {storyThreeID, projectID}, {foreignStoryID, otherProjectID}} {
		created := validStory(story[1])
		created.ID = story[0]
		if err := repo.Create(ctx, created); err != nil {
			t.Fatal(err)
		}
	}
	for _, assignment := range [][3]string{{sprintOneID, storyOneID, projectID}, {sprintTwoID, storyOneID, projectID}, {sprintOneID, storyTwoID, projectID}, {foreignSprintID, foreignStoryID, otherProjectID}} {
		if _, err := pool.Exec(ctx, "INSERT INTO sprint_stories (sprint_id, story_id, project_id) VALUES ($1, $2, $3)", assignment[0], assignment[1], assignment[2]); err != nil {
			t.Fatal(err)
		}
	}
	return repo
}

// completionDatabase applies migrations 000009 and 000010 on top of the story schema and seeds the fixture.
func completionDatabase(t *testing.T) (*pgxpool.Pool, *storypostgres.PostgresStoryRepository) {
	t.Helper()
	pool := storyDatabase(t)
	applyStoryMigration(t, pool, "000009_create_tasks.up.sql")
	applyStoryMigration(t, pool, "000010_add_sprint_story_completion.up.sql")
	return pool, completionFixture(t, pool)
}

func completedAt(t *testing.T, pool *pgxpool.Pool, sprint, story string) *time.Time {
	t.Helper()
	var instant *time.Time
	if err := pool.QueryRow(context.Background(), "SELECT completed_at FROM sprint_stories WHERE sprint_id = $1 AND story_id = $2", sprint, story).Scan(&instant); err != nil {
		t.Fatalf("read completed_at: %v", err)
	}
	return instant
}

func completedCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(completed_at) FROM sprint_stories").Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestCompletionSchemaRequiresMigrationTen(t *testing.T) {
	pool := storyDatabase(t)
	applyStoryMigration(t, pool, "000009_create_tasks.up.sql")
	completionFixture(t, pool)
	applyStoryMigration(t, pool, "000010_add_sprint_story_completion.up.sql")

	var nullable, dataType string
	err := pool.QueryRow(context.Background(), `
		SELECT is_nullable, data_type FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'sprint_stories' AND column_name = 'completed_at'
	`).Scan(&nullable, &dataType)
	if err != nil {
		t.Fatalf("inspect sprint_stories.completed_at: %v", err)
	}
	if nullable != "YES" || dataType != "timestamp with time zone" {
		t.Fatalf("completed_at = nullable %s, type %s; want YES, timestamp with time zone", nullable, dataType)
	}
	// An association created before the migration reads as not completed.
	if got := completedAt(t, pool, sprintOneID, storyOneID); got != nil {
		t.Fatalf("pre-existing association completed_at = %v, want NULL", got)
	}
}

func TestSprintStoryCompletionMigrationCanBeReversedAndReapplied(t *testing.T) {
	pool, _ := completionDatabase(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, "UPDATE sprint_stories SET completed_at = now() WHERE sprint_id = $1 AND story_id = $2", sprintOneID, storyOneID); err != nil {
		t.Fatal(err)
	}
	// A pending task on a completed story must survive the round trip.
	if _, err := pool.Exec(ctx, "INSERT INTO tasks (id, project_id, sprint_id, story_id, title) VALUES ($1, $2, $3, $4, 'Pending task')", missingResourceID, projectID, sprintOneID, storyOneID); err != nil {
		t.Fatal(err)
	}

	applyStoryMigration(t, pool, "000010_add_sprint_story_completion.down.sql")
	var column, associations, tasks int
	err := pool.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'sprint_stories' AND column_name = 'completed_at'),
		       (SELECT count(*) FROM sprint_stories), (SELECT count(*) FROM tasks)
	`).Scan(&column, &associations, &tasks)
	if err != nil {
		t.Fatal(err)
	}
	if column != 0 || associations != 4 || tasks != 1 {
		t.Fatalf("after down: column/associations/tasks = %d/%d/%d, want 0/4/1", column, associations, tasks)
	}

	applyStoryMigration(t, pool, "000010_add_sprint_story_completion.up.sql")
	if got := completedAt(t, pool, sprintOneID, storyOneID); got != nil {
		t.Fatalf("completed_at after reapply = %v, want NULL", got)
	}
}

func TestCompleteSprintStoryPersistsCompletionForThatSprintOnly(t *testing.T) {
	pool, repo := completionDatabase(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, "UPDATE stories SET story_points = 5 WHERE id = $1", storyOneID); err != nil {
		t.Fatal(err)
	}
	storiesBefore, err := repo.ListByProject(ctx, projectID)
	if err != nil {
		t.Fatal(err)
	}

	before := time.Now()
	got, err := repo.CompleteSprintStory(ctx, projectID, sprintOneID, storyOneID)
	after := time.Now()
	if err != nil {
		t.Fatalf("CompleteSprintStory() error = %v", err)
	}

	stored := completedAt(t, pool, sprintOneID, storyOneID)
	if stored == nil || !got.CompletedAt.Equal(*stored) {
		t.Fatalf("returned completed_at = %v, stored = %v; want equal", got.CompletedAt, stored)
	}
	if got.CompletedAt.Before(before.Add(-completionClockMargin)) || got.CompletedAt.After(after.Add(completionClockMargin)) {
		t.Fatalf("completed_at = %v, want within [%v, %v]", got.CompletedAt, before, after)
	}
	if got.ProjectID != projectID || got.SprintID != sprintOneID || got.StoryID != storyOneID {
		t.Fatalf("completion IDs = %+v", got)
	}
	if other := completedAt(t, pool, sprintTwoID, storyOneID); other != nil {
		t.Fatalf("same story in the other sprint completed_at = %v, want NULL", other)
	}
	assertAssignmentCount(t, pool, 4)

	// Completion never touches the story itself, even after the story is edited.
	storiesAfter, err := repo.ListByProject(ctx, projectID)
	if err != nil || !reflect.DeepEqual(storiesBefore, storiesAfter) {
		t.Fatalf("stories after completion = %#v (%v), want unchanged %#v", storiesAfter, err, storiesBefore)
	}
	edited := validStory(projectID)
	edited.Description = "Edited after completion"
	if _, err := repo.Update(ctx, edited); err != nil {
		t.Fatal(err)
	}
	if again := completedAt(t, pool, sprintOneID, storyOneID); again == nil || !again.Equal(*stored) {
		t.Fatalf("completed_at after story update = %v, want %v", again, stored)
	}

	// The same story completes independently in its second sprint.
	second, err := repo.CompleteSprintStory(ctx, projectID, sprintTwoID, storyOneID)
	if err != nil {
		t.Fatalf("second sprint CompleteSprintStory() error = %v", err)
	}
	if secondStored := completedAt(t, pool, sprintTwoID, storyOneID); secondStored == nil || !second.CompletedAt.Equal(*secondStored) || second.CompletedAt.Before(got.CompletedAt) {
		t.Fatalf("second completed_at = %v, stored %v, first %v", second.CompletedAt, secondStored, got.CompletedAt)
	}

	// Metrics scenario: one completed story and its points per sprint, no double counting.
	var completed, points int
	err = pool.QueryRow(ctx, `
		SELECT count(ss.completed_at), COALESCE(sum(s.story_points), 0)
		FROM sprint_stories ss JOIN stories s ON s.id = ss.story_id
		WHERE ss.sprint_id = $1 AND ss.completed_at IS NOT NULL
	`, sprintOneID).Scan(&completed, &points)
	if err != nil || completed != 1 || points != 5 {
		t.Fatalf("sprint metrics = %d stories / %d points (%v), want 1 / 5", completed, points, err)
	}
}

func TestCompleteSprintStoryRejectsMissingOrForeignResourcesWithoutWrites(t *testing.T) {
	pool, repo := completionDatabase(t)
	tests := []struct {
		name                   string
		project, sprint, story string
		want                   error
	}{
		{"missing project", missingResourceID, sprintOneID, storyOneID, application.ErrProjectNotFound},
		{"missing sprint", projectID, missingResourceID, storyOneID, application.ErrSprintNotFound},
		{"sprint of another project", projectID, foreignSprintID, storyOneID, application.ErrSprintNotFound},
		{"missing story", projectID, sprintOneID, missingResourceID, application.ErrStoryNotFound},
		{"story of another project", projectID, sprintOneID, foreignStoryID, application.ErrStoryNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := repo.CompleteSprintStory(context.Background(), tc.project, tc.sprint, tc.story); !errors.Is(err, tc.want) {
				t.Fatalf("CompleteSprintStory() error = %v, want %v", err, tc.want)
			}
			if count := completedCount(t, pool); count != 0 {
				t.Fatalf("completed associations = %d, want 0", count)
			}
		})
	}
}

func TestCompleteSprintStoryRejectsUnassignedStory(t *testing.T) {
	pool, repo := completionDatabase(t)
	for _, tc := range []struct{ name, sprint, story string }{
		{"story without any sprint", sprintOneID, storyThreeID},
		{"story assigned only to another sprint", sprintTwoID, storyTwoID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := repo.CompleteSprintStory(context.Background(), projectID, tc.sprint, tc.story); !errors.Is(err, application.ErrStoryNotInSprint) {
				t.Fatalf("CompleteSprintStory() error = %v, want ErrStoryNotInSprint", err)
			}
			assertAssignmentCount(t, pool, 4)
			if count := completedCount(t, pool); count != 0 {
				t.Fatalf("completed associations = %d, want 0", count)
			}
		})
	}
}

func TestCompleteSprintStoryRejectsClosedSprint(t *testing.T) {
	pool, repo := completionDatabase(t)
	ctx := context.Background()
	original, err := repo.CompleteSprintStory(ctx, projectID, sprintOneID, storyOneID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE sprints SET is_closed = true WHERE id = $1", sprintOneID); err != nil {
		t.Fatal(err)
	}
	// A closed sprint wins over story_not_in_sprint (H3) and story_already_completed (H1).
	for _, story := range []string{storyTwoID, storyThreeID, storyOneID} {
		if _, err := repo.CompleteSprintStory(ctx, projectID, sprintOneID, story); !errors.Is(err, application.ErrSprintClosed) {
			t.Fatalf("story %s in closed sprint error = %v, want ErrSprintClosed", story, err)
		}
	}
	if got := completedAt(t, pool, sprintOneID, storyTwoID); got != nil {
		t.Fatalf("uncompleted story completed_at = %v, want NULL", got)
	}
	if got := completedAt(t, pool, sprintOneID, storyOneID); got == nil || !got.Equal(original.CompletedAt) {
		t.Fatalf("completed story completed_at = %v, want original %v", got, original.CompletedAt)
	}
}

func TestCompleteSprintStoryNeverCountsTwice(t *testing.T) {
	pool, repo := completionDatabase(t)
	first, err := repo.CompleteSprintStory(context.Background(), projectID, sprintOneID, storyOneID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CompleteSprintStory(context.Background(), projectID, sprintOneID, storyOneID); !errors.Is(err, application.ErrStoryAlreadyCompleted) {
		t.Fatalf("second CompleteSprintStory() error = %v, want ErrStoryAlreadyCompleted", err)
	}
	if got := completedAt(t, pool, sprintOneID, storyOneID); got == nil || !got.Equal(first.CompletedAt) {
		t.Fatalf("completed_at after second call = %v, want %v", got, first.CompletedAt)
	}
}

func TestCompleteSprintStoryConcurrentRequestsRecordOnce(t *testing.T) {
	pool, repo := completionDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	type outcome struct {
		completion application.SprintStoryCompletion
		err        error
	}
	const workers = 8
	start := make(chan struct{})
	results := make(chan outcome, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			completion, err := repo.CompleteSprintStory(ctx, projectID, sprintOneID, storyOneID)
			results <- outcome{completion, err}
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	successes, conflicts := 0, 0
	var winner application.SprintStoryCompletion
	for result := range results {
		switch {
		case result.err == nil:
			successes++
			winner = result.completion
		case errors.Is(result.err, application.ErrStoryAlreadyCompleted):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent error: %v", result.err)
		}
	}
	if successes != 1 || conflicts != workers-1 {
		t.Fatalf("successes/conflicts = %d/%d, want 1/%d", successes, conflicts, workers-1)
	}
	if got := completedAt(t, pool, sprintOneID, storyOneID); got == nil || !got.Equal(winner.CompletedAt) {
		t.Fatalf("persisted completed_at = %v, want winner's %v", got, winner.CompletedAt)
	}
}
