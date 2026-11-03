package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	metricsapplication "github.com/valerubio7/software-metrics-and-estimation/internal/metrics/application"
	metricspostgres "github.com/valerubio7/software-metrics-and-estimation/internal/metrics/infrastructure/postgres"
	reportapplication "github.com/valerubio7/software-metrics-and-estimation/internal/report/application"
	reportdomain "github.com/valerubio7/software-metrics-and-estimation/internal/report/domain"
	reportpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/report/infrastructure/postgres"
	"github.com/valerubio7/software-metrics-and-estimation/tests/integration/testpostgres"
)

const (
	reportProjectID         = "11111111-1111-4111-8111-111111111111"
	reportSprintAID         = "22222222-2222-4222-8222-222222222222"
	reportSprintBID         = "33333333-3333-4333-8333-333333333333"
	reportEmptySprintID     = "44444444-4444-4444-8444-444444444444"
	reportStorySharedID     = "55555555-5555-4555-8555-555555555555"
	reportStoryBacklogID    = "66666666-6666-4666-8666-666666666666"
	reportStoryPendingID    = "77777777-7777-4777-8777-777777777777"
	reportTaskAID           = "88888888-8888-4888-8888-888888888888"
	reportTaskBID           = "99999999-9999-4999-8999-999999999999"
	reportMemberID          = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	reportForeignProjectID  = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	reportForeignSprintID   = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	reportForeignStoryID    = "dddddddd-dddd-4ddd-8ddd-dddddddddddd"
	reportEmptyProjectID    = "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee"
	reportDefectAID         = "f1111111-1111-4111-8111-111111111111"
	reportDefectBID         = "f2222222-2222-4222-8222-222222222222"
	reportDefectCID         = "f3333333-3333-4333-8333-333333333333"
	reportForeignDefectID   = "f4444444-4444-4444-8444-444444444444"
	reportDirectEffortAID   = "f5555555-5555-4555-8555-555555555555"
	reportTaskEffortAID     = "f6666666-6666-4666-8666-666666666666"
	reportTaskEffortBID     = "f7777777-7777-4777-8777-777777777777"
	reportDirectEffortBID   = "f8888888-8888-4888-8888-888888888888"
	reportEqualTaskEffortID = "f9999999-9999-4999-8999-999999999999"
)

func TestReportReaderBuildsProjectAndSprintReportsFromOneSnapshot(t *testing.T) {
	pool := seededReportDatabase(t)
	reader := reportpostgres.NewPostgresReportReader(pool)
	metricsUseCase := metricsapplication.NewReadMetricsUseCase(metricspostgres.NewPostgresMetricsReader(pool))
	ctx := context.Background()
	generatedAt := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

	projectSnapshot, err := reader.ReadSnapshot(ctx, reportProjectID, nil)
	if err != nil {
		t.Fatalf("project snapshot: %v", err)
	}
	if projectSnapshot.Project.ID != reportProjectID || projectSnapshot.Project.Name != "Report Project" || projectSnapshot.Sprint != nil {
		t.Errorf("project snapshot scope = %+v, want project without Sprint", projectSnapshot.Project)
	}
	if projectSnapshot.Project.StartDate.Format("2006-01-02") != "2026-01-01" || projectSnapshot.Project.PlannedFinishDate.Format("2006-01-02") != "2026-12-31" {
		t.Errorf("project dates = %v/%v, want 2026-01-01/2026-12-31", projectSnapshot.Project.StartDate, projectSnapshot.Project.PlannedFinishDate)
	}
	if got := countTaskRows(projectSnapshot.Stories); got != 2 {
		t.Errorf("project tasks = %d, want 2", got)
	}
	if len(projectSnapshot.Stories) != 3 || len(projectSnapshot.EffortRecords) != 5 || len(projectSnapshot.Defects) != 3 || len(projectSnapshot.ClosedSprints) != 2 {
		t.Errorf("project snapshot sizes = stories %d, effort %d, defects %d, closed Sprints %d; want 3/5/3/2",
			len(projectSnapshot.Stories), len(projectSnapshot.EffortRecords), len(projectSnapshot.Defects), len(projectSnapshot.ClosedSprints))
	}
	projectReport := reportdomain.Build(projectSnapshot, generatedAt)
	if len(projectReport.EffortRecords) != 5 {
		t.Errorf("project report detail effort = %d, want all 5 project records", len(projectReport.EffortRecords))
	}
	projectMetrics, err := metricsUseCase.Execute(ctx, metricsapplication.ReadMetricsQuery{ProjectID: reportProjectID})
	if err != nil {
		t.Fatalf("project metrics: %v", err)
	}
	if !reflect.DeepEqual(projectReport.Metrics, projectMetrics) {
		t.Errorf("project report metrics = %+v, want metrics API %+v", projectReport.Metrics, projectMetrics)
	}

	sprintASnapshot, err := reader.ReadSnapshot(ctx, reportProjectID, stringPointer(reportSprintAID))
	if err != nil {
		t.Fatalf("Sprint A snapshot: %v", err)
	}
	if sprintASnapshot.Sprint == nil || sprintASnapshot.Sprint.ID != reportSprintAID || sprintASnapshot.Sprint.Goal != "closed delivery" || !sprintASnapshot.Sprint.Closed {
		t.Fatalf("Sprint A snapshot = %+v, want closed delivery", sprintASnapshot.Sprint)
	}
	if len(sprintASnapshot.Stories) != 1 || sprintASnapshot.Stories[0].ID != reportStorySharedID {
		t.Errorf("Sprint A stories = %+v, want only the shared story", sprintASnapshot.Stories)
	}
	if got := countTaskRows(sprintASnapshot.Stories); got != 1 {
		t.Errorf("Sprint A tasks = %d, want only Sprint A tasks", got)
	}
	if len(sprintASnapshot.EffortRecords) != 4 || len(sprintASnapshot.Defects) != 2 {
		t.Errorf("Sprint A snapshot sizes = effort %d, defects %d; want 4/2", len(sprintASnapshot.EffortRecords), len(sprintASnapshot.Defects))
	}
	if len(sprintASnapshot.ClosedSprints) != 0 {
		t.Errorf("Sprint A closed Sprints = %d, want none for Sprint scope", len(sprintASnapshot.ClosedSprints))
	}
	sprintAReport := reportdomain.Build(sprintASnapshot, generatedAt)
	if len(sprintAReport.EffortRecords) != 2 {
		t.Fatalf("Sprint A detail effort = %d, want only the 2 task records attributed to Sprint A", len(sprintAReport.EffortRecords))
	}
	for _, record := range sprintAReport.EffortRecords {
		if record.TaskSprintID == nil || *record.TaskSprintID != reportSprintAID {
			t.Errorf("Sprint A detail effort includes foreign record %+v", record)
		}
	}
	if got := sumHours(sprintAReport.EffortRecords); got != 6 {
		t.Errorf("Sprint A detail effort hours = %v, want 6", got)
	}
	if ids := defectIDs(sprintAReport.Defects); !reflect.DeepEqual(ids, []string{reportDefectAID, reportDefectBID}) {
		t.Errorf("Sprint A defects = %v, want detected A and detected B", ids)
	}
	sprintAMetrics, err := metricsUseCase.Execute(ctx, metricsapplication.ReadMetricsQuery{ProjectID: reportProjectID, SprintID: stringPointer(reportSprintAID)})
	if err != nil {
		t.Fatalf("Sprint A metrics: %v", err)
	}
	if !reflect.DeepEqual(sprintAReport.Metrics, sprintAMetrics) {
		t.Errorf("Sprint A report metrics = %+v, want metrics API %+v", sprintAReport.Metrics, sprintAMetrics)
	}
	assertReportFloat(t, "Sprint A actual hours", sprintAReport.Metrics.ActualHours, 6)
	assertReportFloat(t, "Sprint A velocity", sprintAReport.Metrics.Velocity, 5)
	if sprintAReport.Metrics.DefectsDetected != 2 || sprintAReport.Metrics.DefectsResolved != 0 {
		t.Errorf("Sprint A defects = %d/%d, want 2/0", sprintAReport.Metrics.DefectsDetected, sprintAReport.Metrics.DefectsResolved)
	}
	if sprintAReport.Metrics.Coverage.ActualHours.UnattributedStoryLevelRecords != 1 {
		t.Errorf("Sprint A unattributed records = %d, want 1 direct story record in coverage", sprintAReport.Metrics.Coverage.ActualHours.UnattributedStoryLevelRecords)
	}

	sprintBSnapshot, err := reader.ReadSnapshot(ctx, reportProjectID, stringPointer(reportSprintBID))
	if err != nil {
		t.Fatalf("Sprint B snapshot: %v", err)
	}
	if sprintBSnapshot.Sprint == nil || sprintBSnapshot.Sprint.Closed {
		t.Fatalf("Sprint B snapshot = %+v, want open Sprint", sprintBSnapshot.Sprint)
	}
	if len(sprintBSnapshot.Stories) != 2 || len(sprintBSnapshot.EffortRecords) != 4 || len(sprintBSnapshot.Defects) != 2 {
		t.Errorf("Sprint B snapshot sizes = stories %d, effort %d, defects %d; want 2/4/2",
			len(sprintBSnapshot.Stories), len(sprintBSnapshot.EffortRecords), len(sprintBSnapshot.Defects))
	}
	sprintBReport := reportdomain.Build(sprintBSnapshot, generatedAt)
	if len(sprintBReport.EffortRecords) != 1 || sprintBReport.EffortRecords[0].ID != reportTaskEffortBID {
		t.Errorf("Sprint B detail effort = %+v, want only the Sprint B task record", sprintBReport.EffortRecords)
	}
	if ids := defectIDs(sprintBReport.Defects); !reflect.DeepEqual(ids, []string{reportDefectAID, reportDefectCID}) {
		t.Errorf("Sprint B defects = %v, want detected C and resolved A", ids)
	}
	sprintBMetrics, err := metricsUseCase.Execute(ctx, metricsapplication.ReadMetricsQuery{ProjectID: reportProjectID, SprintID: stringPointer(reportSprintBID)})
	if err != nil {
		t.Fatalf("Sprint B metrics: %v", err)
	}
	if !reflect.DeepEqual(sprintBReport.Metrics, sprintBMetrics) {
		t.Errorf("Sprint B report metrics = %+v, want metrics API %+v", sprintBReport.Metrics, sprintBMetrics)
	}
	if sprintBReport.Metrics.DefectsDetected != 1 || sprintBReport.Metrics.DefectsResolved != 1 {
		t.Errorf("Sprint B defects = %d/%d, want 1/1", sprintBReport.Metrics.DefectsDetected, sprintBReport.Metrics.DefectsResolved)
	}

	emptySprintSnapshot, err := reader.ReadSnapshot(ctx, reportProjectID, stringPointer(reportEmptySprintID))
	if err != nil {
		t.Fatalf("empty Sprint snapshot: %v", err)
	}
	emptySprintReport := reportdomain.Build(emptySprintSnapshot, generatedAt)
	emptySprintMetrics, err := metricsUseCase.Execute(ctx, metricsapplication.ReadMetricsQuery{ProjectID: reportProjectID, SprintID: stringPointer(reportEmptySprintID)})
	if err != nil {
		t.Fatalf("empty Sprint metrics: %v", err)
	}
	if !reflect.DeepEqual(emptySprintReport.Metrics, emptySprintMetrics) {
		t.Errorf("empty Sprint report metrics = %+v, want metrics API %+v", emptySprintReport.Metrics, emptySprintMetrics)
	}

	before := tableCounts(t, pool)
	repeated, err := reader.ReadSnapshot(ctx, reportProjectID, nil)
	if err != nil {
		t.Fatalf("repeated project snapshot: %v", err)
	}
	if !reflect.DeepEqual(repeated, projectSnapshot) {
		t.Error("repeated project snapshot differs from the first read")
	}
	if after := tableCounts(t, pool); !reflect.DeepEqual(after, before) {
		t.Errorf("table counts after reads = %v, want unchanged %v", after, before)
	}
}

func TestReportReaderRejectsMissingAndForeignScopes(t *testing.T) {
	pool := seededReportDatabase(t)
	reader := reportpostgres.NewPostgresReportReader(pool)
	ctx := context.Background()

	if _, err := reader.ReadSnapshot(ctx, "00000000-0000-4000-8000-000000000001", nil); !errors.Is(err, reportapplication.ErrProjectNotFound) {
		t.Errorf("missing project error = %v, want ErrProjectNotFound", err)
	}
	if _, err := reader.ReadSnapshot(ctx, reportEmptyProjectID, nil); err != nil {
		t.Errorf("empty project error = %v, want an empty report snapshot", err)
	}
	if _, err := reader.ReadSnapshot(ctx, reportProjectID, stringPointer(reportForeignSprintID)); !errors.Is(err, reportapplication.ErrSprintNotFound) {
		t.Errorf("foreign Sprint error = %v, want ErrSprintNotFound", err)
	}
	if _, err := reader.ReadSnapshot(ctx, reportProjectID, stringPointer("00000000-0000-4000-8000-000000000002")); !errors.Is(err, reportapplication.ErrSprintNotFound) {
		t.Errorf("missing Sprint error = %v, want ErrSprintNotFound", err)
	}
}

func seededReportDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := openReportPool(t)
	for version := 1; version <= 14; version++ {
		applyReportMigration(t, pool, fmt.Sprintf("%06d", version))
	}
	seedReportData(t, pool)
	return pool
}

func openReportPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("PostgreSQL integration requires Docker")
	}
	if dsn := os.Getenv("REPORT_TEST_DATABASE_URL"); dsn != "" {
		return testpostgres.OpenIsolated(t, dsn, "REPORT_TEST_DATABASE_URL", "report_test")
	}
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("report_test"),
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

func applyReportMigration(t *testing.T, pool *pgxpool.Pool, version string) {
	t.Helper()
	pattern := filepath.Join(reportModuleRoot(t), "internal", "project", "infrastructure", "postgres", "migrations", version+"_*.up.sql")
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

func reportModuleRoot(t *testing.T) string {
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

func seedReportData(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	for _, project := range []struct{ id, name string }{
		{reportProjectID, "Report Project"},
		{reportForeignProjectID, "Foreign Project"},
		{reportEmptyProjectID, "Empty Project"},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO projects (id, name, start_date, planned_finish_date) VALUES ($1, $2, '2026-01-01', '2026-12-31')`, project.id, project.name); err != nil {
			t.Fatalf("insert project %s: %v", project.id, err)
		}
	}
	for _, sprint := range []struct{ id, projectID, goal string }{
		{reportSprintAID, reportProjectID, "closed delivery"},
		{reportSprintBID, reportProjectID, "open delivery"},
		{reportEmptySprintID, reportProjectID, "empty closed delivery"},
		{reportForeignSprintID, reportForeignProjectID, "foreign delivery"},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO sprints (id, project_id, sprint_goal) VALUES ($1, $2, $3)`, sprint.id, sprint.projectID, sprint.goal); err != nil {
			t.Fatalf("insert Sprint %s: %v", sprint.id, err)
		}
	}
	for _, sprintID := range []string{reportSprintAID, reportEmptySprintID, reportForeignSprintID} {
		if _, err := pool.Exec(ctx, `UPDATE sprints SET is_closed = true WHERE id = $1`, sprintID); err != nil {
			t.Fatalf("close Sprint %s: %v", sprintID, err)
		}
	}
	for _, story := range []struct {
		id, projectID, title, status string
		points                       *int
		hours                        *float64
	}{
		{reportStorySharedID, reportProjectID, "Shared story", "pendiente", intPointer(5), floatPointer(10)},
		{reportStoryBacklogID, reportProjectID, "Backlog story", "pendiente", intPointer(3), floatPointer(4)},
		{reportStoryPendingID, reportProjectID, "Pending story", "completada", intPointer(13), floatPointer(6)},
		{reportForeignStoryID, reportForeignProjectID, "Foreign story", "pendiente", intPointer(100), floatPointer(100)},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO stories (id, project_id, title, description, priority, status, story_points, acceptance_criteria, estimated_hours) VALUES ($1, $2, $3, 'Details', 'media', $4, $5, ARRAY['Done'], $6)`, story.id, story.projectID, story.title, story.status, story.points, story.hours); err != nil {
			t.Fatalf("insert story %s: %v", story.id, err)
		}
	}
	for _, association := range [][3]string{
		{reportSprintAID, reportStorySharedID, reportProjectID},
		{reportSprintBID, reportStorySharedID, reportProjectID},
		{reportSprintBID, reportStoryPendingID, reportProjectID},
		{reportForeignSprintID, reportForeignStoryID, reportForeignProjectID},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO sprint_stories (sprint_id, story_id, project_id) VALUES ($1, $2, $3)`, association[0], association[1], association[2]); err != nil {
			t.Fatalf("assign story to Sprint: %v", err)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE sprint_stories SET completed_at = '2026-05-01T12:00:00Z' WHERE sprint_id = $1 AND story_id = $2`, reportSprintAID, reportStorySharedID); err != nil {
		t.Fatalf("complete shared story in Sprint A: %v", err)
	}
	for _, task := range []struct {
		id, projectID, sprintID, storyID, title string
		hours                                   float64
	}{
		{reportTaskAID, reportProjectID, reportSprintAID, reportStorySharedID, "Sprint A task", 2},
		{reportTaskBID, reportProjectID, reportSprintBID, reportStorySharedID, "Sprint B task", 3},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO tasks (id, project_id, sprint_id, story_id, title, estimated_hours) VALUES ($1, $2, $3, $4, $5, $6)`, task.id, task.projectID, task.sprintID, task.storyID, task.title, task.hours); err != nil {
			t.Fatalf("insert task %s: %v", task.id, err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO project_members (id, project_id, full_name) VALUES ($1, $2, 'Report Member')`, reportMemberID, reportProjectID); err != nil {
		t.Fatalf("insert project member: %v", err)
	}
	for _, effort := range []struct {
		id, storyID string
		taskID      *string
		hours       float64
	}{
		{reportDirectEffortAID, reportStorySharedID, nil, 2},
		{reportTaskEffortAID, reportStorySharedID, stringPointer(reportTaskAID), 3},
		{reportTaskEffortBID, reportStorySharedID, stringPointer(reportTaskBID), 4},
		{reportEqualTaskEffortID, reportStorySharedID, stringPointer(reportTaskAID), 3},
		{reportDirectEffortBID, reportStoryBacklogID, nil, 1.5},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO effort_records (id, project_id, story_id, task_id, member_id, work_date, description, hours) VALUES ($1, $2, $3, $4, $5, '2026-05-02', 'Report test', $6)`, effort.id, reportProjectID, effort.storyID, effort.taskID, reportMemberID, effort.hours); err != nil {
			t.Fatalf("insert effort record %s: %v", effort.id, err)
		}
	}
	for _, defect := range []struct {
		id, projectID, storyID, sprintID, status string
		resolutionSprintID                       *string
	}{
		{reportDefectAID, reportProjectID, reportStorySharedID, reportSprintAID, "resuelto", stringPointer(reportSprintBID)},
		{reportDefectBID, reportProjectID, reportStoryBacklogID, reportSprintAID, "resuelto", nil},
		{reportDefectCID, reportProjectID, reportStoryPendingID, reportSprintBID, "resuelto", nil},
		{reportForeignDefectID, reportForeignProjectID, reportForeignStoryID, reportForeignSprintID, "detectado", nil},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO defects (id, project_id, story_id, sprint_id, description, severity, status, resolution_sprint_id) VALUES ($1, $2, $3, $4, 'Report defect', 'media', $5, $6)`, defect.id, defect.projectID, defect.storyID, defect.sprintID, defect.status, defect.resolutionSprintID); err != nil {
			t.Fatalf("insert defect %s: %v", defect.id, err)
		}
	}
}

func tableCounts(t *testing.T, pool *pgxpool.Pool) map[string]int {
	t.Helper()
	counts := make(map[string]int)
	for _, table := range []string{"projects", "sprints", "stories", "sprint_stories", "tasks", "project_members", "effort_records", "defects"} {
		var count int
		if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		counts[table] = count
	}
	return counts
}

func countTaskRows(stories []reportdomain.Story) int {
	total := 0
	for _, story := range stories {
		total += len(story.Tasks)
	}
	return total
}

func sumHours(records []reportdomain.EffortRecord) float64 {
	total := 0.0
	for _, record := range records {
		total += record.Hours
	}
	return total
}

func defectIDs(defects []reportdomain.Defect) []string {
	ids := make([]string, 0, len(defects))
	for _, defect := range defects {
		ids = append(ids, defect.ID)
	}
	return ids
}

func assertReportFloat(t *testing.T, name string, got *float64, want float64) {
	t.Helper()
	if got == nil || *got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func intPointer(value int) *int           { return &value }
func floatPointer(value float64) *float64 { return &value }
func stringPointer(value string) *string  { return &value }
