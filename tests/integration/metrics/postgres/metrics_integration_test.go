package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/valerubio7/software-metrics-and-estimation/internal/metrics/application"
	metricspostgres "github.com/valerubio7/software-metrics-and-estimation/internal/metrics/infrastructure/postgres"
	"github.com/valerubio7/software-metrics-and-estimation/tests/integration/testpostgres"
)

const (
	metricsProjectID         = "11111111-1111-4111-8111-111111111111"
	metricsSprintAID         = "22222222-2222-4222-8222-222222222222"
	metricsSprintBID         = "33333333-3333-4333-8333-333333333333"
	metricsSprintEmptyID     = "44444444-4444-4444-8444-444444444444"
	metricsStorySharedID     = "55555555-5555-4555-8555-555555555555"
	metricsStoryBacklogID    = "66666666-6666-4666-8666-666666666666"
	metricsStoryPendingID    = "77777777-7777-4777-8777-777777777777"
	metricsTaskAID           = "88888888-8888-4888-8888-888888888888"
	metricsTaskBID           = "99999999-9999-4999-8999-999999999999"
	metricsMemberID          = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	metricsForeignProjectID  = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	metricsForeignSprintID   = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	metricsForeignStoryID    = "dddddddd-dddd-4ddd-8ddd-dddddddddddd"
	metricsEmptyProjectID    = "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee"
	metricsDefectAID         = "f1111111-1111-4111-8111-111111111111"
	metricsDefectBID         = "f2222222-2222-4222-8222-222222222222"
	metricsDefectCID         = "f3333333-3333-4333-8333-333333333333"
	metricsForeignDefectID   = "f4444444-4444-4444-8444-444444444444"
	metricsDirectEffortAID   = "f5555555-5555-4555-8555-555555555555"
	metricsTaskEffortAID     = "f6666666-6666-4666-8666-666666666666"
	metricsTaskEffortBID     = "f7777777-7777-4777-8777-777777777777"
	metricsDirectEffortBID   = "f8888888-8888-4888-8888-888888888888"
	metricsEqualTaskEffortID = "f9999999-9999-4999-8999-999999999999"
)

func TestMetricsReaderCalculatesProjectAndSprintMetrics(t *testing.T) {
	pool := seededMetricsDatabase(t)
	useCase := application.NewReadMetricsUseCase(metricspostgres.NewPostgresMetricsReader(pool))
	ctx := context.Background()

	project, err := useCase.Execute(ctx, application.ReadMetricsQuery{ProjectID: metricsProjectID})
	if err != nil {
		t.Fatalf("project metrics: %v", err)
	}
	assertScope(t, project.ProjectID, metricsProjectID, project.SprintID, nil)
	assertFloat(t, "project planned points", project.StoryPointsPlanned, 21)
	assertFloat(t, "project completed points", project.StoryPointsCompleted, 5)
	assertFloat(t, "project velocity", project.Velocity, 2.5)
	assertFloat(t, "project estimated hours", project.EstimatedHours, 15)
	assertFloat(t, "project actual hours", project.ActualHours, 13.5)
	assertFloat(t, "project deviation", project.Deviation, -1.5)
	assertFloat(t, "project deviation percentage", project.DeviationPercentage, -10)
	assertFloat(t, "project completion percentage", project.CompletedStoriesPercentage, 100.0/3.0)
	if project.DefectsDetected != 3 || project.DefectsResolved != 1 {
		t.Errorf("project defects = %d/%d, want 3 detected and 1 resolved", project.DefectsDetected, project.DefectsResolved)
	}
	if project.Coverage.StoryPointsPlanned.MissingObservations != 0 || project.Coverage.StoryPointsCompleted.MissingObservations != 0 || project.Coverage.Velocity.MissingObservations != 0 || project.Coverage.EstimatedHours.MissingObservations != 0 || project.Coverage.ActualHours.StoriesWithoutRecords != 1 || project.Coverage.ActualHours.UnattributedStoryLevelRecords != 0 {
		t.Errorf("project coverage = %+v, want no missing estimates, one story without effort, and no unattributed project records", project.Coverage)
	}

	sprintA, err := useCase.Execute(ctx, application.ReadMetricsQuery{ProjectID: metricsProjectID, SprintID: stringPointer(metricsSprintAID)})
	if err != nil {
		t.Fatalf("Sprint A metrics: %v", err)
	}
	assertScope(t, sprintA.ProjectID, metricsProjectID, sprintA.SprintID, stringPointer(metricsSprintAID))
	assertFloat(t, "Sprint A planned points", sprintA.StoryPointsPlanned, 5)
	assertFloat(t, "Sprint A completed points", sprintA.StoryPointsCompleted, 5)
	assertFloat(t, "Sprint A velocity", sprintA.Velocity, 5)
	assertFloat(t, "Sprint A estimated hours", sprintA.EstimatedHours, 2)
	assertFloat(t, "Sprint A actual hours", sprintA.ActualHours, 6)
	assertFloat(t, "Sprint A deviation", sprintA.Deviation, 4)
	assertFloat(t, "Sprint A deviation percentage", sprintA.DeviationPercentage, 200)
	assertFloat(t, "Sprint A completion percentage", sprintA.CompletedStoriesPercentage, 100)
	if sprintA.DefectsDetected != 2 || sprintA.DefectsResolved != 0 {
		t.Errorf("Sprint A defects = %d/%d, want 2 detected and 0 resolved", sprintA.DefectsDetected, sprintA.DefectsResolved)
	}
	if sprintA.Coverage.ActualHours.StoriesWithoutRecords != 0 || sprintA.Coverage.ActualHours.UnattributedStoryLevelRecords != 1 {
		t.Errorf("Sprint A actual-hours coverage = %+v, want 0 without records and 1 unattributed story record", sprintA.Coverage.ActualHours)
	}

	sprintB, err := useCase.Execute(ctx, application.ReadMetricsQuery{ProjectID: metricsProjectID, SprintID: stringPointer(metricsSprintBID)})
	if err != nil {
		t.Fatalf("Sprint B metrics: %v", err)
	}
	assertFloat(t, "Sprint B planned points", sprintB.StoryPointsPlanned, 18)
	assertFloat(t, "Sprint B completed points", sprintB.StoryPointsCompleted, 0)
	assertFloat(t, "Sprint B velocity", sprintB.Velocity, 0)
	assertFloat(t, "Sprint B estimated hours", sprintB.EstimatedHours, 9)
	assertFloat(t, "Sprint B actual hours", sprintB.ActualHours, 4)
	assertFloat(t, "Sprint B deviation", sprintB.Deviation, -5)
	assertFloat(t, "Sprint B deviation percentage", sprintB.DeviationPercentage, -500.0/9.0)
	assertFloat(t, "Sprint B completion percentage", sprintB.CompletedStoriesPercentage, 0)
	if sprintB.DefectsDetected != 1 || sprintB.DefectsResolved != 1 {
		t.Errorf("Sprint B defects = %d/%d, want 1 detected and 1 resolved", sprintB.DefectsDetected, sprintB.DefectsResolved)
	}
	if sprintB.Coverage.ActualHours.StoriesWithoutRecords != 1 || sprintB.Coverage.ActualHours.UnattributedStoryLevelRecords != 1 {
		t.Errorf("Sprint B actual-hours coverage = %+v, want 1 without records and 1 unattributed story record", sprintB.Coverage.ActualHours)
	}

	emptySprint, err := useCase.Execute(ctx, application.ReadMetricsQuery{ProjectID: metricsProjectID, SprintID: stringPointer(metricsSprintEmptyID)})
	if err != nil {
		t.Fatalf("empty Sprint metrics: %v", err)
	}
	assertFloat(t, "empty Sprint planned points", emptySprint.StoryPointsPlanned, 0)
	assertFloat(t, "empty Sprint completed points", emptySprint.StoryPointsCompleted, 0)
	assertFloat(t, "empty Sprint velocity", emptySprint.Velocity, 0)
	assertNil(t, "empty Sprint estimated hours", emptySprint.EstimatedHours)
	assertNil(t, "empty Sprint actual hours", emptySprint.ActualHours)
	assertNil(t, "empty Sprint completion percentage", emptySprint.CompletedStoriesPercentage)

	emptyProject, err := useCase.Execute(ctx, application.ReadMetricsQuery{ProjectID: metricsEmptyProjectID})
	if err != nil {
		t.Fatalf("empty project metrics: %v", err)
	}
	assertFloat(t, "empty project planned points", emptyProject.StoryPointsPlanned, 0)
	assertFloat(t, "empty project completed points", emptyProject.StoryPointsCompleted, 0)
	assertNil(t, "empty project velocity", emptyProject.Velocity)
	assertNil(t, "empty project estimated hours", emptyProject.EstimatedHours)
	assertNil(t, "empty project actual hours", emptyProject.ActualHours)
	assertNil(t, "empty project completion percentage", emptyProject.CompletedStoriesPercentage)
	if emptyProject.DefectsDetected != 0 || emptyProject.DefectsResolved != 0 {
		t.Errorf("empty project defects = %d/%d, want 0/0", emptyProject.DefectsDetected, emptyProject.DefectsResolved)
	}

	if repeated, err := useCase.Execute(ctx, application.ReadMetricsQuery{ProjectID: metricsProjectID}); err != nil || !reflect.DeepEqual(repeated, project) {
		t.Errorf("repeated project query = (%+v, %v), want identical metrics and no observable mutation", repeated, err)
	}
	if _, err := useCase.Execute(ctx, application.ReadMetricsQuery{ProjectID: "00000000-0000-4000-8000-000000000001"}); !errors.Is(err, application.ErrProjectNotFound) {
		t.Errorf("missing project error = %v, want ErrProjectNotFound", err)
	}
	if _, err := useCase.Execute(ctx, application.ReadMetricsQuery{ProjectID: metricsProjectID, SprintID: stringPointer(metricsForeignSprintID)}); !errors.Is(err, application.ErrSprintNotFound) {
		t.Errorf("foreign Sprint error = %v, want ErrSprintNotFound", err)
	}
	if _, err := useCase.Execute(ctx, application.ReadMetricsQuery{ProjectID: metricsProjectID, SprintID: stringPointer("00000000-0000-4000-8000-000000000002")}); !errors.Is(err, application.ErrSprintNotFound) {
		t.Errorf("missing Sprint error = %v, want ErrSprintNotFound", err)
	}
	afterRejectedReads, err := useCase.Execute(ctx, application.ReadMetricsQuery{ProjectID: metricsProjectID})
	if err != nil || !reflect.DeepEqual(afterRejectedReads, project) {
		t.Errorf("metrics after rejected reads = (%+v, %v), want original public result with no mutation", afterRejectedReads, err)
	}
}

func seededMetricsDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := openMetricsPool(t)
	for version := 1; version <= 14; version++ {
		applyMetricsMigration(t, pool, fmt.Sprintf("%06d", version))
	}
	seedMetricsData(t, pool)
	return pool
}

func openMetricsPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("PostgreSQL integration requires Docker")
	}
	if dsn := os.Getenv("METRICS_TEST_DATABASE_URL"); dsn != "" {
		return testpostgres.OpenIsolated(t, dsn, "METRICS_TEST_DATABASE_URL", "metrics_test")
	}
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("metrics_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
	)
	if err != nil {
		t.Fatalf("start PostgreSQL container (no silent Docker skip): %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Errorf("terminate PostgreSQL container: %v", err)
		}
	})
	connectionString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get PostgreSQL connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, connectionString)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	t.Cleanup(pool.Close)
	deadline := time.Now().Add(10 * time.Second)
	for {
		pingCtx, cancel := context.WithTimeout(ctx, time.Second)
		err := pool.Ping(pingCtx)
		cancel()
		if err == nil {
			return pool
		}
		if time.Now().After(deadline) {
			t.Fatalf("wait for PostgreSQL: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func applyMetricsMigration(t *testing.T, pool *pgxpool.Pool, version string) {
	t.Helper()
	pattern := filepath.Join(metricsModuleRoot(t), "internal", "project", "infrastructure", "postgres", "migrations", version+"_*.up.sql")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) != 1 {
		t.Fatalf("migration %s matches = %v, error = %v", version, matches, err)
	}
	contents, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read migration %s: %v", matches[0], err)
	}
	if _, err := pool.Exec(context.Background(), string(contents)); err != nil {
		t.Fatalf("apply migration %s: %v", filepath.Base(matches[0]), err)
	}
}

func metricsModuleRoot(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			t.Fatal("locate module root: go.mod not found")
		}
		directory = parent
	}
}

func seedMetricsData(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	for _, projectID := range []string{metricsProjectID, metricsForeignProjectID, metricsEmptyProjectID} {
		if _, err := pool.Exec(ctx, `INSERT INTO projects (id, name, start_date, planned_finish_date) VALUES ($1, 'Metrics', '2026-01-01', '2026-12-31')`, projectID); err != nil {
			t.Fatalf("insert project %s: %v", projectID, err)
		}
	}
	for _, sprint := range []struct{ id, projectID, goal string }{
		{metricsSprintAID, metricsProjectID, "closed delivery"},
		{metricsSprintBID, metricsProjectID, "open delivery"},
		{metricsSprintEmptyID, metricsProjectID, "empty closed delivery"},
		{metricsForeignSprintID, metricsForeignProjectID, "foreign delivery"},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO sprints (id, project_id, sprint_goal) VALUES ($1, $2, $3)`, sprint.id, sprint.projectID, sprint.goal); err != nil {
			t.Fatalf("insert Sprint %s: %v", sprint.id, err)
		}
	}
	for _, sprintID := range []string{metricsSprintAID, metricsSprintEmptyID, metricsForeignSprintID} {
		if _, err := pool.Exec(ctx, `UPDATE sprints SET is_closed = true WHERE id = $1`, sprintID); err != nil {
			t.Fatalf("close Sprint %s: %v", sprintID, err)
		}
	}
	stories := []struct {
		id, projectID, status string
		points                *int
		hours                 *float64
	}{
		{metricsStorySharedID, metricsProjectID, "pendiente", intPointer(5), floatPointer(10)},
		{metricsStoryBacklogID, metricsProjectID, "pendiente", intPointer(3), floatPointer(4)},
		{metricsStoryPendingID, metricsProjectID, "completada", intPointer(13), floatPointer(6)},
		{metricsForeignStoryID, metricsForeignProjectID, "pendiente", intPointer(100), floatPointer(100)},
	}
	for _, story := range stories {
		if _, err := pool.Exec(ctx, `INSERT INTO stories (id, project_id, title, description, priority, status, story_points, acceptance_criteria, estimated_hours) VALUES ($1, $2, 'Story', 'Details', 'media', $3, $4, ARRAY['Done'], $5)`, story.id, story.projectID, story.status, story.points, story.hours); err != nil {
			t.Fatalf("insert story %s: %v", story.id, err)
		}
	}
	for _, association := range [][3]string{
		{metricsSprintAID, metricsStorySharedID, metricsProjectID},
		{metricsSprintBID, metricsStorySharedID, metricsProjectID},
		{metricsSprintBID, metricsStoryPendingID, metricsProjectID},
		{metricsForeignSprintID, metricsForeignStoryID, metricsForeignProjectID},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO sprint_stories (sprint_id, story_id, project_id) VALUES ($1, $2, $3)`, association[0], association[1], association[2]); err != nil {
			t.Fatalf("assign story to Sprint: %v", err)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE sprint_stories SET completed_at = '2026-05-01T12:00:00Z' WHERE sprint_id = $1 AND story_id = $2`, metricsSprintAID, metricsStorySharedID); err != nil {
		t.Fatalf("complete shared story in Sprint A: %v", err)
	}
	for _, task := range []struct {
		id, projectID, sprintID, storyID string
		hours                            float64
	}{
		{metricsTaskAID, metricsProjectID, metricsSprintAID, metricsStorySharedID, 2},
		{metricsTaskBID, metricsProjectID, metricsSprintBID, metricsStorySharedID, 3},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO tasks (id, project_id, sprint_id, story_id, title, estimated_hours) VALUES ($1, $2, $3, $4, 'Task', $5)`, task.id, task.projectID, task.sprintID, task.storyID, task.hours); err != nil {
			t.Fatalf("insert task %s: %v", task.id, err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO project_members (id, project_id, full_name) VALUES ($1, $2, 'Metrics Member')`, metricsMemberID, metricsProjectID); err != nil {
		t.Fatalf("insert project member: %v", err)
	}
	for _, effort := range []struct {
		id, storyID string
		taskID      *string
		hours       float64
	}{
		{metricsDirectEffortAID, metricsStorySharedID, nil, 2},
		{metricsTaskEffortAID, metricsStorySharedID, stringPointer(metricsTaskAID), 3},
		{metricsTaskEffortBID, metricsStorySharedID, stringPointer(metricsTaskBID), 4},
		{metricsEqualTaskEffortID, metricsStorySharedID, stringPointer(metricsTaskAID), 3},
		{metricsDirectEffortBID, metricsStoryBacklogID, nil, 1.5},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO effort_records (id, project_id, story_id, task_id, member_id, work_date, description, hours) VALUES ($1, $2, $3, $4, $5, '2026-05-02', 'Metrics test', $6)`, effort.id, metricsProjectID, effort.storyID, effort.taskID, metricsMemberID, effort.hours); err != nil {
			t.Fatalf("insert effort record %s: %v", effort.id, err)
		}
	}
	for _, defect := range []struct {
		id, projectID, storyID, sprintID, status string
		resolutionSprintID                       *string
	}{
		{metricsDefectAID, metricsProjectID, metricsStorySharedID, metricsSprintAID, "resuelto", stringPointer(metricsSprintBID)},
		{metricsDefectBID, metricsProjectID, metricsStoryBacklogID, metricsSprintAID, "resuelto", nil},
		{metricsDefectCID, metricsProjectID, metricsStoryPendingID, metricsSprintBID, "resuelto", nil},
		{metricsForeignDefectID, metricsForeignProjectID, metricsForeignStoryID, metricsForeignSprintID, "detectado", nil},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO defects (id, project_id, story_id, sprint_id, description, severity, status, resolution_sprint_id) VALUES ($1, $2, $3, $4, 'Metrics defect', 'media', $5, $6)`, defect.id, defect.projectID, defect.storyID, defect.sprintID, defect.status, defect.resolutionSprintID); err != nil {
			t.Fatalf("insert defect %s: %v", defect.id, err)
		}
	}
}

func assertScope(t *testing.T, projectID string, wantProjectID string, sprintID, wantSprintID *string) {
	t.Helper()
	if projectID != wantProjectID || !reflect.DeepEqual(sprintID, wantSprintID) {
		t.Errorf("metrics scope = %q/%v, want %q/%v", projectID, sprintID, wantProjectID, wantSprintID)
	}
}

func assertFloat(t *testing.T, name string, got *float64, want float64) {
	t.Helper()
	if got == nil || math.Abs(*got-want) > 1e-9 {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func assertNil(t *testing.T, name string, got *float64) {
	t.Helper()
	if got != nil {
		t.Errorf("%s = %v, want null", name, *got)
	}
}

func intPointer(value int) *int           { return &value }
func floatPointer(value float64) *float64 { return &value }
func stringPointer(value string) *string  { return &value }
