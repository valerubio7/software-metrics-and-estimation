package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
	storypostgres "github.com/valerubio7/software-metrics-and-estimation/internal/story/infrastructure/postgres"
	"github.com/valerubio7/software-metrics-and-estimation/tests/integration/testpostgres"
)

func TestSetStoryPointsOnMigratedSchemaV2IsScopedAndOnlyChangesPoints(t *testing.T) {
	pool := storyDatabaseV2(t)
	insertProject(t, pool, projectID)
	insertProject(t, pool, otherProjectID)
	seed := validStory(projectID)
	seed.Title = "Keep title"
	seed.Description = "Keep description"
	seed.Priority = "media"
	seed.AcceptanceCriteria = []string{"Keep criterion"}
	insertV2Story(t, pool, seed)
	other := validStory(otherProjectID)
	other.ID = "e89c05a4-03ea-4c19-8f59-286dd59e7aa1"
	insertV2Story(t, pool, other)
	got, err := storypostgres.NewPostgresStoryRepository(pool).SetStoryPoints(context.Background(), projectID, storyID, 13)
	if err != nil {
		t.Fatalf("SetStoryPoints() error=%v", err)
	}
	if got.ID != storyID || got.ProjectID != projectID || got.Title != "Keep title" || got.Description != "Keep description" || got.Priority != "media" || got.StoryPoints == nil || *got.StoryPoints != 13 {
		t.Errorf("returned story=%+v", got)
	}
	if got.EstimatedHours != nil {
		t.Errorf("schema v2 response estimated_hours=%v, want null", *got.EstimatedHours)
	}
	var title, description, priority, status string
	var storedPoints *int
	var criteria []string
	if err := pool.QueryRow(context.Background(), `SELECT title, description, priority, status, story_points, acceptance_criteria FROM stories WHERE id=$1 AND project_id=$2`, storyID, projectID).Scan(&title, &description, &priority, &status, &storedPoints, &criteria); err != nil {
		t.Fatalf("read updated v2 story: %v", err)
	}
	if title != seed.Title || description != seed.Description || priority != seed.Priority || status != seed.Status || !equalStrings(criteria, seed.AcceptanceCriteria) || storedPoints == nil || *storedPoints != 13 {
		t.Errorf("v2 stored values changed beyond points: title=%q description=%q priority=%q status=%q points=%v criteria=%v", title, description, priority, status, storedPoints, criteria)
	}
	var otherPoints *int
	if err := pool.QueryRow(context.Background(), `SELECT story_points FROM stories WHERE id=$1 AND project_id=$2`, other.ID, otherProjectID).Scan(&otherPoints); err != nil {
		t.Fatalf("read other-project story: %v", err)
	}
	if otherPoints != nil {
		t.Fatal("story in another project was modified")
	}
	missing, err := storypostgres.NewPostgresStoryRepository(pool).SetStoryPoints(context.Background(), otherProjectID, storyID, 8)
	if err == nil || missing.ID != "" {
		t.Errorf("cross-project update result=%+v err=%v, want scoped not-found", missing, err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT story_points FROM stories WHERE id=$1 AND project_id=$2`, storyID, projectID).Scan(&storedPoints); err != nil {
		t.Fatalf("verify cross-project request: %v", err)
	}
	if storedPoints == nil || *storedPoints != 13 {
		t.Error("cross-project request modified story")
	}
}

func TestSetStoryPointsOnMigratedSchemaV3PreservesEstimatedHoursAndStoryFields(t *testing.T) {
	pool := storyDatabaseV2(t)
	applyStoryMigration(t, pool, "000003_add_story_estimated_hours.up.sql")
	insertProject(t, pool, projectID)
	seed := validStory(projectID)
	seed.Title = "Keep title"
	seed.Description = "Keep description"
	seed.Priority = "media"
	seed.AcceptanceCriteria = []string{"Keep criterion"}
	estimatedHours := 7.5
	seed.EstimatedHours = &estimatedHours
	_, err := pool.Exec(context.Background(), `INSERT INTO stories (id, project_id, title, description, priority, status, story_points, acceptance_criteria, estimated_hours) VALUES ($1,$2,$3,$4,$5,$6,NULL,$7,$8)`, seed.ID, seed.ProjectID, seed.Title, seed.Description, seed.Priority, seed.Status, seed.AcceptanceCriteria, estimatedHours)
	if err != nil {
		t.Fatalf("insert schema v3 story: %v", err)
	}

	got, err := storypostgres.NewPostgresStoryRepository(pool).SetStoryPoints(context.Background(), projectID, storyID, 13)
	if err != nil {
		t.Fatalf("SetStoryPoints() error=%v", err)
	}
	if got.ID != seed.ID || got.ProjectID != seed.ProjectID || got.Title != seed.Title || got.Description != seed.Description || got.Priority != seed.Priority || got.Status != seed.Status || got.StoryPoints == nil || *got.StoryPoints != 13 || !equalStrings(got.AcceptanceCriteria, seed.AcceptanceCriteria) || got.EstimatedHours == nil || *got.EstimatedHours != estimatedHours {
		t.Errorf("returned schema v3 story=%+v, want unchanged story with points=13 and estimated_hours=%v", got, estimatedHours)
	}

	var title, description, priority, status string
	var storedPoints *int
	var criteria []string
	var storedHours *float64
	if err := pool.QueryRow(context.Background(), `SELECT title, description, priority, status, story_points, acceptance_criteria, estimated_hours FROM stories WHERE id=$1 AND project_id=$2`, storyID, projectID).Scan(&title, &description, &priority, &status, &storedPoints, &criteria, &storedHours); err != nil {
		t.Fatalf("read updated v3 story: %v", err)
	}
	if title != seed.Title || description != seed.Description || priority != seed.Priority || status != seed.Status || !equalStrings(criteria, seed.AcceptanceCriteria) || storedPoints == nil || *storedPoints != 13 || storedHours == nil || *storedHours != estimatedHours {
		t.Errorf("v3 stored values changed beyond points: title=%q description=%q priority=%q status=%q points=%v criteria=%v estimated_hours=%v", title, description, priority, status, storedPoints, criteria, storedHours)
	}
}

func insertV2Story(t *testing.T, pool *pgxpool.Pool, story domain.Story) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `INSERT INTO stories (id, project_id, title, description, priority, status, story_points, acceptance_criteria) VALUES ($1,$2,$3,$4,$5,$6,NULL,$7)`, story.ID, story.ProjectID, story.Title, story.Description, story.Priority, story.Status, story.AcceptanceCriteria)
	if err != nil {
		t.Fatalf("insert schema v2 story: %v", err)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func storyDatabaseV2(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("PostgreSQL integration requires Docker")
	}
	if dsn := os.Getenv("STORY_TEST_DATABASE_URL"); dsn != "" {
		return storyDatabaseV2FromDSN(t, dsn)
	}
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine", postgres.WithDatabase("stories_test"), postgres.WithUsername("postgres"), postgres.WithPassword("postgres"))
	if err != nil {
		t.Fatalf("start PostgreSQL container (no silent Docker skip): %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Errorf("terminate PostgreSQL container: %v", err)
		}
	})
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get PostgreSQL connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect PostgreSQL: %v", err)
	}
	t.Cleanup(pool.Close)
	deadline := time.Now().Add(10 * time.Second)
	for {
		ping, cancel := context.WithTimeout(ctx, time.Second)
		err = pool.Ping(ping)
		cancel()
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("wait for PostgreSQL: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}
	applyStoryMigration(t, pool, "000001_create_projects.up.sql")
	applyStoryMigration(t, pool, "000002_create_stories.up.sql")
	return pool
}
func storyDatabaseV2FromDSN(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	pool := testpostgres.OpenIsolated(t, dsn, "STORY_TEST_DATABASE_URL", "story_points_v2_test")
	applyStoryMigration(t, pool, "000001_create_projects.up.sql")
	applyStoryMigration(t, pool, "000002_create_stories.up.sql")
	return pool
}
