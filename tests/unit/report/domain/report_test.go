package domain_test

import (
	"testing"
	"time"

	metricsdomain "github.com/valerubio7/software-metrics-and-estimation/internal/metrics/domain"
	"github.com/valerubio7/software-metrics-and-estimation/internal/report/domain"
)

func TestBuildProjectReportDerivesMetricsFromTheSameSnapshot(t *testing.T) {
	completedAt := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	generatedAt := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	storyA := domain.Story{
		ID: "story-a", Title: "Historia A", Priority: "alta", Status: "completada",
		StoryPoints: intPtr(5), EstimatedHours: floatPtr(10), Completed: true, CompletedAt: &completedAt,
		Tasks: []domain.Task{
			{ID: "task-a1", Title: "Tarea A1", SprintID: "sprint-a", EstimatedHours: floatPtr(2)},
			{ID: "task-a2", Title: "Tarea A2", SprintID: "sprint-b", EstimatedHours: floatPtr(3)},
		},
	}
	storyB := domain.Story{
		ID: "story-b", Title: "Historia B", Priority: "media", Status: "pendiente",
		StoryPoints: intPtr(3), EstimatedHours: floatPtr(4),
	}
	report := domain.Build(domain.Snapshot{
		Project: domain.Project{ID: "project", Name: "Proyecto"},
		Stories: []domain.Story{storyA, storyB},
		EffortRecords: []domain.EffortRecord{
			{ID: "effort-direct", StoryID: "story-a", StoryTitle: "Historia A", Hours: 2},
			{ID: "effort-a1", StoryID: "story-a", StoryTitle: "Historia A", TaskID: stringPtr("task-a1"), TaskTitle: stringPtr("Tarea A1"), TaskSprintID: stringPtr("sprint-a"), Hours: 3},
			{ID: "effort-a2", StoryID: "story-a", StoryTitle: "Historia A", TaskID: stringPtr("task-a2"), TaskTitle: stringPtr("Tarea A2"), TaskSprintID: stringPtr("sprint-b"), Hours: 4},
			{ID: "effort-b", StoryID: "story-b", StoryTitle: "Historia B", Hours: 1.5},
		},
		Defects: []domain.Defect{
			{ID: "defect-1", StoryID: "story-a", DetectionSprintID: "sprint-a", ResolutionSprintID: stringPtr("sprint-b")},
			{ID: "defect-2", StoryID: "story-a", DetectionSprintID: "sprint-a"},
		},
		ClosedSprints: []metricsdomain.ClosedSprintInput{
			{CompletedStoryPoints: []*int{intPtr(8)}},
			{CompletedStoryPoints: []*int{}},
		},
	}, generatedAt)

	if report.Project.ID != "project" || len(report.Stories) != 2 || len(report.EffortRecords) != 4 {
		t.Fatalf("report detail = project %q, stories %d, effort %d; want all project rows", report.Project.ID, len(report.Stories), len(report.EffortRecords))
	}
	if !report.GeneratedAt.Equal(generatedAt) {
		t.Errorf("generated at = %v, want %v", report.GeneratedAt, generatedAt)
	}
	assertFloat(t, "planned points", report.Metrics.StoryPointsPlanned, 8)
	assertFloat(t, "completed points", report.Metrics.StoryPointsCompleted, 5)
	assertFloat(t, "velocity", report.Metrics.Velocity, 4)
	assertFloat(t, "estimated hours", report.Metrics.EstimatedHours, 9)
	assertFloat(t, "actual hours", report.Metrics.ActualHours, 10.5)
	assertFloat(t, "deviation", report.Metrics.Deviation, 1.5)
	assertFloat(t, "deviation percentage", report.Metrics.DeviationPercentage, *report.Metrics.Deviation/9*100)
	assertFloat(t, "completed percentage", report.Metrics.CompletedStoriesPercentage, 50)
	if report.Metrics.DefectsDetected != 2 || report.Metrics.DefectsResolved != 1 {
		t.Errorf("defects = %d/%d, want 2 detected and 1 resolved", report.Metrics.DefectsDetected, report.Metrics.DefectsResolved)
	}
	if report.Metrics.Coverage.ActualHours.StoriesWithoutRecords != 0 || report.Metrics.Coverage.ActualHours.UnattributedStoryLevelRecords != 0 {
		t.Errorf("actual-hours coverage = %+v, want all project records attributable", report.Metrics.Coverage.ActualHours)
	}
}

func TestBuildSprintReportKeepsOnlyAttributableEffortInTheDetail(t *testing.T) {
	completedAt := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	report := domain.Build(domain.Snapshot{
		Project: domain.Project{ID: "project", Name: "Proyecto"},
		Sprint:  &domain.Sprint{ID: "sprint-a", Goal: "Entrega A", Closed: true},
		Stories: []domain.Story{{
			ID: "story-a", Title: "Historia A", Priority: "alta", Status: "completada",
			StoryPoints: intPtr(5), EstimatedHours: floatPtr(10), Completed: true, CompletedAt: &completedAt,
			Tasks: []domain.Task{{ID: "task-a1", Title: "Tarea A1", SprintID: "sprint-a", EstimatedHours: floatPtr(2)}},
		}},
		EffortRecords: []domain.EffortRecord{
			{ID: "effort-direct", StoryID: "story-a", StoryTitle: "Historia A", Hours: 2},
			{ID: "effort-a1", StoryID: "story-a", StoryTitle: "Historia A", TaskID: stringPtr("task-a1"), TaskTitle: stringPtr("Tarea A1"), TaskSprintID: stringPtr("sprint-a"), Hours: 3},
			{ID: "effort-b", StoryID: "story-a", StoryTitle: "Historia A", TaskID: stringPtr("task-b"), TaskTitle: stringPtr("Tarea B"), TaskSprintID: stringPtr("sprint-b"), Hours: 4},
		},
		Defects: []domain.Defect{
			{ID: "defect-detected", StoryID: "story-a", DetectionSprintID: "sprint-a", ResolutionSprintID: stringPtr("sprint-b")},
			{ID: "defect-only-detected", StoryID: "story-a", DetectionSprintID: "sprint-a"},
			{ID: "defect-resolved", StoryID: "story-a", DetectionSprintID: "sprint-b", ResolutionSprintID: stringPtr("sprint-a")},
		},
	}, time.Now())

	if report.Sprint == nil || report.Sprint.ID != "sprint-a" {
		t.Fatalf("report Sprint = %+v, want sprint-a", report.Sprint)
	}
	if len(report.EffortRecords) != 1 || report.EffortRecords[0].ID != "effort-a1" {
		t.Fatalf("detail effort = %+v, want only the task record attributed to sprint-a", report.EffortRecords)
	}
	assertFloat(t, "planned points", report.Metrics.StoryPointsPlanned, 5)
	assertFloat(t, "completed points", report.Metrics.StoryPointsCompleted, 5)
	assertFloat(t, "velocity", report.Metrics.Velocity, 5)
	assertFloat(t, "estimated hours", report.Metrics.EstimatedHours, 2)
	assertFloat(t, "actual hours", report.Metrics.ActualHours, 3)
	assertFloat(t, "deviation", report.Metrics.Deviation, 1)
	assertFloat(t, "deviation percentage", report.Metrics.DeviationPercentage, 50)
	if report.Metrics.DefectsDetected != 2 || report.Metrics.DefectsResolved != 1 {
		t.Errorf("defects = %d/%d, want 2 detected and 1 resolved", report.Metrics.DefectsDetected, report.Metrics.DefectsResolved)
	}
	if got := report.Metrics.Coverage.ActualHours.UnattributedStoryLevelRecords; got != 1 {
		t.Errorf("unattributed story records = %d, want 1", got)
	}
	if got := report.Metrics.Coverage.ActualHours.StoriesWithoutRecords; got != 0 {
		t.Errorf("stories without attributable records = %d, want 0", got)
	}
}

func TestBuildEmptyScopesKeepUS19NullSemantics(t *testing.T) {
	t.Run("empty project", func(t *testing.T) {
		report := domain.Build(domain.Snapshot{Project: domain.Project{ID: "project"}}, time.Time{})

		assertFloat(t, "planned points", report.Metrics.StoryPointsPlanned, 0)
		assertFloat(t, "completed points", report.Metrics.StoryPointsCompleted, 0)
		assertNil(t, "velocity", report.Metrics.Velocity)
		assertNil(t, "estimated hours", report.Metrics.EstimatedHours)
		assertNil(t, "actual hours", report.Metrics.ActualHours)
		assertNil(t, "completed percentage", report.Metrics.CompletedStoriesPercentage)
		if report.Metrics.DefectsDetected != 0 || report.Metrics.DefectsResolved != 0 {
			t.Errorf("defects = %d/%d, want 0/0", report.Metrics.DefectsDetected, report.Metrics.DefectsResolved)
		}
	})

	t.Run("empty Sprint", func(t *testing.T) {
		report := domain.Build(domain.Snapshot{
			Project: domain.Project{ID: "project"},
			Sprint:  &domain.Sprint{ID: "sprint"},
		}, time.Time{})

		assertFloat(t, "planned points", report.Metrics.StoryPointsPlanned, 0)
		assertFloat(t, "completed points", report.Metrics.StoryPointsCompleted, 0)
		assertFloat(t, "velocity", report.Metrics.Velocity, 0)
		assertNil(t, "estimated hours", report.Metrics.EstimatedHours)
		assertNil(t, "actual hours", report.Metrics.ActualHours)
		assertNil(t, "completed percentage", report.Metrics.CompletedStoriesPercentage)
	})
}

func TestBuildKeepsMissingEstimatesUnknown(t *testing.T) {
	report := domain.Build(domain.Snapshot{
		Project: domain.Project{ID: "project"},
		Stories: []domain.Story{{ID: "story", Title: "Sin estimar"}},
	}, time.Time{})

	assertNil(t, "planned points", report.Metrics.StoryPointsPlanned)
	assertFloat(t, "completed points", report.Metrics.StoryPointsCompleted, 0)
	if got := report.Metrics.Coverage.StoryPointsPlanned.MissingObservations; got != 1 {
		t.Errorf("planned point coverage = %d, want 1", got)
	}
	if got := report.Metrics.Coverage.StoryPointsCompleted.MissingObservations; got != 0 {
		t.Errorf("completed point coverage = %d, want 0", got)
	}
}

func assertFloat(t *testing.T, name string, got *float64, want float64) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %v", name, want)
	}
	if *got != want {
		t.Errorf("%s = %v, want %v", name, *got, want)
	}
}

func assertNil(t *testing.T, name string, got *float64) {
	t.Helper()
	if got != nil {
		t.Errorf("%s = %v, want null", name, *got)
	}
}

func intPtr(value int) *int           { return &value }
func floatPtr(value float64) *float64 { return &value }
func stringPtr(value string) *string  { return &value }
