// Package domain defines the metric calculation model and pure calculations.
package domain

// Input contains one consistent, already scoped read of the source data. Project scope
// supplies each project story once and all of its tasks and effort records. Sprint scope
// supplies the assigned stories and their tasks and effort records, with SprintID on each
// task-linked row. ClosedSprints is populated for project scope only.
type Input struct {
	ProjectID       string
	SprintID        *string
	Stories         []StoryInput
	ClosedSprints   []ClosedSprintInput
	DefectsDetected int
	DefectsResolved int
}

// StoryInput contains the source values needed to calculate metrics for one unique story.
type StoryInput struct {
	ID             string
	StoryPoints    *int
	Completed      bool
	EstimatedHours *float64
	Tasks          []TaskInput
	EffortRecords  []EffortRecordInput
}

// TaskInput contains one task and its optional hour estimate.
type TaskInput struct {
	ID             string
	SprintID       string
	EstimatedHours *float64
}

// EffortRecordInput contains one recorded effort value. SprintID is the owning Sprint of
// TaskID and is nil for effort recorded directly against the story.
type EffortRecordInput struct {
	TaskID   *string
	SprintID *string
	Hours    float64
}

// ClosedSprintInput contains completed Story Points for one closed Sprint. A nil point
// entry means that Sprint's completed total is unknown; an empty slice means zero points.
type ClosedSprintInput struct {
	CompletedStoryPoints []*int
}

// Metrics is the flat response model shared with the read-only metrics transport.
type Metrics struct {
	ProjectID                  string         `json:"project_id"`
	SprintID                   *string        `json:"sprint_id"`
	StoryPointsPlanned         *float64       `json:"story_points_planned"`
	StoryPointsCompleted       *float64       `json:"story_points_completed"`
	Velocity                   *float64       `json:"velocity"`
	EstimatedHours             *float64       `json:"estimated_hours"`
	ActualHours                *float64       `json:"actual_hours"`
	Deviation                  *float64       `json:"deviation"`
	DeviationPercentage        *float64       `json:"deviation_percentage"`
	CompletedStoriesPercentage *float64       `json:"completed_stories_percentage"`
	DefectsDetected            int            `json:"defects_detected"`
	DefectsResolved            int            `json:"defects_resolved"`
	Coverage                   MetricCoverage `json:"coverage"`
}

// MetricCoverage records unavailable observations without changing metric values.
type MetricCoverage struct {
	StoryPointsPlanned   ValueCoverage       `json:"story_points_planned"`
	StoryPointsCompleted ValueCoverage       `json:"story_points_completed"`
	Velocity             ValueCoverage       `json:"velocity"`
	EstimatedHours       ValueCoverage       `json:"estimated_hours"`
	ActualHours          ActualHoursCoverage `json:"actual_hours"`
}

// ValueCoverage counts source observations needed to calculate a metric but not present.
type ValueCoverage struct {
	MissingObservations int `json:"missing_observations"`
}

// ActualHoursCoverage distinguishes stories with no attributable records from direct
// story-level records that cannot be assigned to a specific Sprint.
type ActualHoursCoverage struct {
	StoriesWithoutRecords         int `json:"stories_without_records"`
	UnattributedStoryLevelRecords int `json:"unattributed_story_level_records"`
}

// ValidationError describes invalid metric query identifiers by field.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "metrics query validation failed" }

// Calculate derives every metric from a consistent, scoped input snapshot.
func Calculate(input Input) Metrics {
	metrics := Metrics{
		ProjectID:       input.ProjectID,
		SprintID:        cloneString(input.SprintID),
		DefectsDetected: input.DefectsDetected,
		DefectsResolved: input.DefectsResolved,
	}

	plannedPoints, completedPoints := 0, 0
	completedStories := 0
	for _, story := range input.Stories {
		if story.StoryPoints == nil {
			metrics.Coverage.StoryPointsPlanned.MissingObservations++
		} else {
			plannedPoints += *story.StoryPoints
		}
		if !story.Completed {
			continue
		}
		completedStories++
		if story.StoryPoints == nil {
			metrics.Coverage.StoryPointsCompleted.MissingObservations++
		} else {
			completedPoints += *story.StoryPoints
		}
	}
	if metrics.Coverage.StoryPointsPlanned.MissingObservations == 0 {
		metrics.StoryPointsPlanned = floatPointer(float64(plannedPoints))
	}
	if metrics.Coverage.StoryPointsCompleted.MissingObservations == 0 {
		metrics.StoryPointsCompleted = floatPointer(float64(completedPoints))
	}
	if len(input.Stories) > 0 {
		metrics.CompletedStoriesPercentage = floatPointer(float64(completedStories) / float64(len(input.Stories)) * 100)
	}

	calculateVelocity(&metrics, input, completedPoints)
	calculateEstimatedHours(&metrics, input)
	calculateActualHours(&metrics, input)
	calculateDeviations(&metrics)
	return metrics
}

func calculateVelocity(metrics *Metrics, input Input, sprintCompletedPoints int) {
	if input.SprintID != nil {
		if metrics.Coverage.StoryPointsCompleted.MissingObservations == 0 {
			metrics.Velocity = floatPointer(float64(sprintCompletedPoints))
		}
		metrics.Coverage.Velocity.MissingObservations = metrics.Coverage.StoryPointsCompleted.MissingObservations
		return
	}
	if len(input.ClosedSprints) == 0 {
		return
	}

	total := 0
	incompleteSprints := 0
	for _, sprint := range input.ClosedSprints {
		sprintPoints := 0
		incomplete := false
		for _, points := range sprint.CompletedStoryPoints {
			if points == nil {
				incomplete = true
				continue
			}
			sprintPoints += *points
		}
		if incomplete {
			incompleteSprints++
			continue
		}
		total += sprintPoints
	}
	metrics.Coverage.Velocity.MissingObservations = incompleteSprints
	if incompleteSprints == 0 {
		metrics.Velocity = floatPointer(float64(total) / float64(len(input.ClosedSprints)))
	}
}

func calculateEstimatedHours(metrics *Metrics, input Input) {
	total := 0.0
	observations := 0
	for _, story := range input.Stories {
		tasks := selectedTasks(story.Tasks, input.SprintID)
		if len(tasks) == 0 {
			observations++
			if story.EstimatedHours == nil {
				metrics.Coverage.EstimatedHours.MissingObservations++
			} else {
				total += *story.EstimatedHours
			}
			continue
		}
		for _, task := range tasks {
			observations++
			if task.EstimatedHours == nil {
				metrics.Coverage.EstimatedHours.MissingObservations++
			} else {
				total += *task.EstimatedHours
			}
		}
	}
	if observations > 0 && metrics.Coverage.EstimatedHours.MissingObservations == 0 {
		metrics.EstimatedHours = floatPointer(total)
	}
}

func calculateActualHours(metrics *Metrics, input Input) {
	total := 0.0
	observedRecords := 0
	for _, story := range input.Stories {
		storyHasAttributableRecord := false
		for _, record := range story.EffortRecords {
			if input.SprintID == nil {
				observedRecords++
				total += record.Hours
				storyHasAttributableRecord = true
				continue
			}
			if record.TaskID == nil {
				metrics.Coverage.ActualHours.UnattributedStoryLevelRecords++
				continue
			}
			if record.SprintID == nil || *record.SprintID != *input.SprintID {
				continue
			}
			observedRecords++
			total += record.Hours
			storyHasAttributableRecord = true
		}
		if !storyHasAttributableRecord {
			metrics.Coverage.ActualHours.StoriesWithoutRecords++
		}
	}
	if observedRecords > 0 {
		metrics.ActualHours = floatPointer(total)
	}
}

func calculateDeviations(metrics *Metrics) {
	if metrics.ActualHours == nil || metrics.EstimatedHours == nil {
		return
	}
	deviation := *metrics.ActualHours - *metrics.EstimatedHours
	metrics.Deviation = floatPointer(deviation)
	if *metrics.EstimatedHours > 0 {
		metrics.DeviationPercentage = floatPointer(deviation / *metrics.EstimatedHours * 100)
	}
}

func selectedTasks(tasks []TaskInput, sprintID *string) []TaskInput {
	if sprintID == nil {
		return tasks
	}
	selected := make([]TaskInput, 0, len(tasks))
	for _, task := range tasks {
		if task.SprintID == *sprintID {
			selected = append(selected, task)
		}
	}
	return selected
}

func floatPointer(value float64) *float64 { return &value }

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
