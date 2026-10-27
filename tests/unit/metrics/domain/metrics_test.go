package domain_test

import (
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/metrics/domain"
)

func intPtr(value int) *int           { return &value }
func floatPtr(value float64) *float64 { return &value }
func stringPtr(value string) *string  { return &value }

func requireFloat(t *testing.T, got *float64, want float64) {
	t.Helper()
	if got == nil {
		t.Fatalf("metric = nil, want %v", want)
	}
	if *got != want {
		t.Errorf("metric = %v, want %v", *got, want)
	}
}

func TestCalculateProjectStoryPointAndCompletionMetrics(t *testing.T) {
	got := domain.Calculate(domain.Input{
		ProjectID: "project",
		Stories: []domain.StoryInput{
			{ID: "completed", StoryPoints: intPtr(5), Completed: true},
			{ID: "pending", StoryPoints: intPtr(3)},
		},
	})

	requireFloat(t, got.StoryPointsPlanned, 8)
	requireFloat(t, got.StoryPointsCompleted, 5)
	requireFloat(t, got.CompletedStoriesPercentage, 50)
}

func TestCalculatePointCoverageForIncompleteEstimates(t *testing.T) {
	t.Run("missing points on a pending story only affects the planned sum", func(t *testing.T) {
		got := domain.Calculate(domain.Input{
			ProjectID: "project",
			Stories: []domain.StoryInput{
				{ID: "completed", StoryPoints: intPtr(3), Completed: true},
				{ID: "pending-without-points"},
			},
		})

		if got.StoryPointsPlanned != nil {
			t.Errorf("planned points = %v, want nil", *got.StoryPointsPlanned)
		}
		requireFloat(t, got.StoryPointsCompleted, 3)
		if got.Coverage.StoryPointsPlanned.MissingObservations != 1 || got.Coverage.StoryPointsCompleted.MissingObservations != 0 {
			t.Errorf("point coverage = %#v, want planned missing 1 and completed missing 0", got.Coverage)
		}
	})

	t.Run("missing points on a completed story makes both sums incomplete", func(t *testing.T) {
		got := domain.Calculate(domain.Input{
			ProjectID: "project",
			Stories:   []domain.StoryInput{{ID: "completed-without-points", Completed: true}},
		})

		if got.StoryPointsPlanned != nil || got.StoryPointsCompleted != nil {
			t.Errorf("planned/completed points = %v/%v, want nil/nil", got.StoryPointsPlanned, got.StoryPointsCompleted)
		}
		if got.Coverage.StoryPointsPlanned.MissingObservations != 1 || got.Coverage.StoryPointsCompleted.MissingObservations != 1 {
			t.Errorf("point coverage = %#v, want one missing observation for each aggregate", got.Coverage)
		}
	})
}

func TestCalculateEmptyScopeMetrics(t *testing.T) {
	got := domain.Calculate(domain.Input{ProjectID: "project"})

	requireFloat(t, got.StoryPointsPlanned, 0)
	requireFloat(t, got.StoryPointsCompleted, 0)
	if got.CompletedStoriesPercentage != nil {
		t.Errorf("completed percentage = %v, want nil without a denominator", *got.CompletedStoriesPercentage)
	}
	if got.EstimatedHours != nil || got.ActualHours != nil {
		t.Errorf("estimated/actual hours = %v/%v, want nil/nil for empty observations", got.EstimatedHours, got.ActualHours)
	}
	if got.DefectsDetected != 0 || got.DefectsResolved != 0 {
		t.Errorf("defect counts = %d/%d, want 0/0", got.DefectsDetected, got.DefectsResolved)
	}
}

func TestCalculateDefectCounts(t *testing.T) {
	got := domain.Calculate(domain.Input{ProjectID: "project", DefectsDetected: 4, DefectsResolved: 2})
	if got.DefectsDetected != 4 || got.DefectsResolved != 2 {
		t.Errorf("defect counts = %d/%d, want 4/2", got.DefectsDetected, got.DefectsResolved)
	}
}

func TestCalculateSprintVelocity(t *testing.T) {
	t.Run("completed points are the Sprint velocity", func(t *testing.T) {
		got := domain.Calculate(domain.Input{
			ProjectID: "project",
			SprintID:  stringPtr("sprint-a"),
			Stories: []domain.StoryInput{
				{ID: "completed", StoryPoints: intPtr(8), Completed: true},
				{ID: "pending", StoryPoints: intPtr(5)},
			},
		})
		requireFloat(t, got.Velocity, 8)
	})

	t.Run("no completed stories is zero velocity", func(t *testing.T) {
		got := domain.Calculate(domain.Input{ProjectID: "project", SprintID: stringPtr("empty-sprint")})
		requireFloat(t, got.Velocity, 0)
	})

	t.Run("missing completed Story Points make velocity unknown", func(t *testing.T) {
		got := domain.Calculate(domain.Input{
			ProjectID: "project",
			SprintID:  stringPtr("sprint-a"),
			Stories:   []domain.StoryInput{{ID: "completed-without-points", Completed: true}},
		})
		if got.Velocity != nil || got.Coverage.Velocity.MissingObservations != 1 {
			t.Errorf("velocity/coverage = %v/%d, want nil/1", got.Velocity, got.Coverage.Velocity.MissingObservations)
		}
	})
}

func TestCalculateProjectVelocity(t *testing.T) {
	t.Run("averages every closed Sprint including zero and excludes open Sprints", func(t *testing.T) {
		got := domain.Calculate(domain.Input{
			ProjectID: "project",
			ClosedSprints: []domain.ClosedSprintInput{
				{CompletedStoryPoints: []*int{intPtr(8)}},
				{CompletedStoryPoints: []*int{}},
			},
		})
		requireFloat(t, got.Velocity, 4)
	})

	t.Run("no closed Sprints has no project velocity", func(t *testing.T) {
		got := domain.Calculate(domain.Input{ProjectID: "project"})
		if got.Velocity != nil || got.Coverage.Velocity.MissingObservations != 0 {
			t.Errorf("velocity/coverage = %v/%d, want nil/0", got.Velocity, got.Coverage.Velocity.MissingObservations)
		}
	})

	t.Run("one incomplete closed Sprint makes the project average unknown", func(t *testing.T) {
		got := domain.Calculate(domain.Input{
			ProjectID: "project",
			ClosedSprints: []domain.ClosedSprintInput{
				{CompletedStoryPoints: []*int{intPtr(8)}},
				{CompletedStoryPoints: []*int{nil}},
			},
		})
		if got.Velocity != nil || got.Coverage.Velocity.MissingObservations != 1 {
			t.Errorf("velocity/coverage = %v/%d, want nil/1", got.Velocity, got.Coverage.Velocity.MissingObservations)
		}
	})
}

func TestCalculateEstimatedHoursByScope(t *testing.T) {
	storyEstimate := floatPtr(10)
	got := domain.Calculate(domain.Input{
		ProjectID: "project",
		Stories: []domain.StoryInput{
			{
				ID:             "shared-story",
				EstimatedHours: storyEstimate,
				Tasks: []domain.TaskInput{
					{ID: "task-a", SprintID: "sprint-a", EstimatedHours: floatPtr(2)},
					{ID: "task-b", SprintID: "sprint-b", EstimatedHours: floatPtr(3)},
				},
			},
		},
	})
	requireFloat(t, got.EstimatedHours, 5)

	sprintA := domain.Calculate(domain.Input{
		ProjectID: "project",
		SprintID:  stringPtr("sprint-a"),
		Stories:   []domain.StoryInput{{ID: "shared-story", EstimatedHours: storyEstimate, Tasks: []domain.TaskInput{{ID: "task-a", SprintID: "sprint-a", EstimatedHours: floatPtr(2)}, {ID: "task-b", SprintID: "sprint-b", EstimatedHours: floatPtr(3)}}}},
	})
	requireFloat(t, sprintA.EstimatedHours, 2)

	sprintB := domain.Calculate(domain.Input{
		ProjectID: "project",
		SprintID:  stringPtr("sprint-b"),
		Stories:   []domain.StoryInput{{ID: "shared-story", EstimatedHours: storyEstimate, Tasks: []domain.TaskInput{{ID: "task-a", SprintID: "sprint-a", EstimatedHours: floatPtr(2)}}}},
	})
	requireFloat(t, sprintB.EstimatedHours, 10)
}

func TestCalculateEstimatedHoursMissingCoverage(t *testing.T) {
	got := domain.Calculate(domain.Input{
		ProjectID: "project",
		Stories: []domain.StoryInput{
			{ID: "tasked", EstimatedHours: floatPtr(10), Tasks: []domain.TaskInput{{ID: "estimated", SprintID: "sprint-a", EstimatedHours: floatPtr(2)}, {ID: "missing", SprintID: "sprint-a"}}},
			{ID: "unestimated", EstimatedHours: nil},
		},
	})
	if got.EstimatedHours != nil || got.Coverage.EstimatedHours.MissingObservations != 2 {
		t.Errorf("estimated hours/coverage = %v/%d, want nil/2", got.EstimatedHours, got.Coverage.EstimatedHours.MissingObservations)
	}
}

func TestCalculateActualHoursByScope(t *testing.T) {
	got := domain.Calculate(domain.Input{
		ProjectID: "project",
		Stories: []domain.StoryInput{
			{
				ID:    "shared-story",
				Tasks: []domain.TaskInput{{ID: "task-a", SprintID: "sprint-a"}, {ID: "task-b", SprintID: "sprint-b"}},
				EffortRecords: []domain.EffortRecordInput{
					{Hours: 2},
					{TaskID: stringPtr("task-a"), SprintID: stringPtr("sprint-a"), Hours: 3},
					{TaskID: stringPtr("task-b"), SprintID: stringPtr("sprint-b"), Hours: 4},
					{TaskID: stringPtr("task-a-equal-hours"), SprintID: stringPtr("sprint-a"), Hours: 3},
				},
			},
			{ID: "no-effort"},
		},
	})
	requireFloat(t, got.ActualHours, 12)
	if got.Coverage.ActualHours.StoriesWithoutRecords != 1 {
		t.Errorf("project stories_without_records = %d, want 1", got.Coverage.ActualHours.StoriesWithoutRecords)
	}
	if got.Coverage.ActualHours.UnattributedStoryLevelRecords != 0 {
		t.Errorf("project unattributed_story_level_records = %d, want 0", got.Coverage.ActualHours.UnattributedStoryLevelRecords)
	}

	sprintA := domain.Calculate(domain.Input{
		ProjectID: "project",
		SprintID:  stringPtr("sprint-a"),
		Stories: []domain.StoryInput{{
			ID:    "shared-story",
			Tasks: []domain.TaskInput{{ID: "task-a", SprintID: "sprint-a"}, {ID: "task-b", SprintID: "sprint-b"}},
			EffortRecords: []domain.EffortRecordInput{
				{Hours: 2},
				{TaskID: stringPtr("task-a"), SprintID: stringPtr("sprint-a"), Hours: 3},
				{TaskID: stringPtr("task-b"), SprintID: stringPtr("sprint-b"), Hours: 4},
			},
		}},
	})
	requireFloat(t, sprintA.ActualHours, 3)
	if sprintA.Coverage.ActualHours.UnattributedStoryLevelRecords != 1 || sprintA.Coverage.ActualHours.StoriesWithoutRecords != 0 {
		t.Errorf("Sprint A actual-hour coverage = %#v, want one unattributed record and no story missing", sprintA.Coverage.ActualHours)
	}

	sprintB := domain.Calculate(domain.Input{
		ProjectID: "project",
		SprintID:  stringPtr("sprint-b"),
		Stories: []domain.StoryInput{{
			ID:            "shared-story",
			EffortRecords: []domain.EffortRecordInput{{Hours: 2}, {TaskID: stringPtr("task-a"), SprintID: stringPtr("sprint-a"), Hours: 3}},
		}},
	})
	if sprintB.ActualHours != nil || sprintB.Coverage.ActualHours.UnattributedStoryLevelRecords != 1 || sprintB.Coverage.ActualHours.StoriesWithoutRecords != 1 {
		t.Errorf("Sprint B hours/coverage = %v/%#v, want nil, one unattributed and one without attributable records", sprintB.ActualHours, sprintB.Coverage.ActualHours)
	}
}

func TestCalculateEffortDeviations(t *testing.T) {
	t.Run("positive and negative deviations have matching signed percentages", func(t *testing.T) {
		for _, tc := range []struct {
			name               string
			actual, estimated  float64
			deviation, percent float64
		}{{"over", 10, 8, 2, 25}, {"under", 6, 8, -2, -25}} {
			t.Run(tc.name, func(t *testing.T) {
				got := domain.Calculate(domain.Input{ProjectID: "project", Stories: []domain.StoryInput{{ID: "story", EstimatedHours: floatPtr(tc.estimated), EffortRecords: []domain.EffortRecordInput{{Hours: tc.actual}}}}})
				requireFloat(t, got.Deviation, tc.deviation)
				requireFloat(t, got.DeviationPercentage, tc.percent)
			})
		}
	})

	t.Run("percentage is undefined for zero estimated hours", func(t *testing.T) {
		got := domain.Calculate(domain.Input{ProjectID: "project", Stories: []domain.StoryInput{{ID: "story", EstimatedHours: floatPtr(0), EffortRecords: []domain.EffortRecordInput{{Hours: 2}}}}})
		requireFloat(t, got.Deviation, 2)
		if got.DeviationPercentage != nil {
			t.Errorf("deviation percentage = %v, want nil for zero estimate", *got.DeviationPercentage)
		}
	})

	t.Run("missing estimate makes both deviations unavailable", func(t *testing.T) {
		got := domain.Calculate(domain.Input{ProjectID: "project", Stories: []domain.StoryInput{{ID: "story", EffortRecords: []domain.EffortRecordInput{{Hours: 2}}}}})
		if got.Deviation != nil || got.DeviationPercentage != nil {
			t.Errorf("deviations = %v/%v, want nil/nil", got.Deviation, got.DeviationPercentage)
		}
	})
}
