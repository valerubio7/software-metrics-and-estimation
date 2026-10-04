package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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
