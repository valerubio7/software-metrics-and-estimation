// Package domain defines the report read model and its pure composition rules.
package domain

import (
	"time"

	metricsdomain "github.com/valerubio7/software-metrics-and-estimation/internal/metrics/domain"
)

// Project identifies the project covered by a report.
type Project struct {
	ID                string
	Name              string
	StartDate         time.Time
	PlannedFinishDate time.Time
}

// Sprint identifies the Sprint covered by a report. Closed reflects the stored state.
type Sprint struct {
	ID     string
	Goal   string
	Closed bool
}

// Task is one decomposed task of a story, with its Sprint and optional estimate.
type Task struct {
	ID             string
	Title          string
	SprintID       string
	EstimatedHours *float64
}

// Story is one story in the selected scope. Completed and CompletedAt come from the
// scope association, never from the global story status.
type Story struct {
	ID             string
	Title          string
	Priority       string
	Status         string
	StoryPoints    *int
	EstimatedHours *float64
	Completed      bool
	CompletedAt    *time.Time
	Tasks          []Task
}

// EffortRecord is one registered effort row. TaskSprintID is the Sprint owning TaskID
// and is nil for effort recorded directly against the story.
type EffortRecord struct {
	ID           string
	StoryID      string
	StoryTitle   string
	TaskID       *string
	TaskTitle    *string
	TaskSprintID *string
	MemberName   string
	WorkDate     time.Time
	Description  string
	Hours        float64
}

// Defect is one defect related to the selected scope.
type Defect struct {
	ID                   string
	StoryID              string
	StoryTitle           string
	Description          string
	Severity             string
	Status               string
	DetectionSprintID    string
	DetectionSprintGoal  string
	ResolutionSprintID   *string
	ResolutionSprintGoal *string
}

// Snapshot is one consistent read of every source needed by a report. ClosedSprints is
// populated for project scope only; Sprint scope derives velocity from its completed points.
type Snapshot struct {
	Project       Project
	Sprint        *Sprint
	Stories       []Story
	EffortRecords []EffortRecord
	Defects       []Defect
	ClosedSprints []metricsdomain.ClosedSprintInput
}

// Report is the data rendered into a PDF. Metrics are calculated from the same snapshot
// with the US-19 formulas, so they cannot disagree with the detail sections.
type Report struct {
	Project       Project
	Sprint        *Sprint
	Stories       []Story
	EffortRecords []EffortRecord
	Defects       []Defect
	Metrics       metricsdomain.Metrics
	GeneratedAt   time.Time
}

// ValidationError describes invalid report query identifiers by field.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "report query validation failed" }

// Build derives the report from one snapshot. The detail effort table keeps only rows
// attributable to the selected Sprint; the metrics input keeps every row of assigned
// stories so that US-19 can report direct story records as unattributed.
func Build(snapshot Snapshot, generatedAt time.Time) Report {
	metricsInput := metricsdomain.Input{
		ProjectID:     snapshot.Project.ID,
		ClosedSprints: snapshot.ClosedSprints,
	}
	if snapshot.Sprint != nil {
		sprintID := snapshot.Sprint.ID
		metricsInput.SprintID = &sprintID
	}

	storyIndexes := make(map[string]int, len(snapshot.Stories))
	for _, story := range snapshot.Stories {
		metricsInput.Stories = append(metricsInput.Stories, metricsdomain.StoryInput{
			ID:             story.ID,
			StoryPoints:    story.StoryPoints,
			Completed:      story.Completed,
			EstimatedHours: story.EstimatedHours,
		})
		storyIndexes[story.ID] = len(metricsInput.Stories) - 1
		for _, task := range story.Tasks {
			index := storyIndexes[story.ID]
			metricsInput.Stories[index].Tasks = append(metricsInput.Stories[index].Tasks, metricsdomain.TaskInput{
				ID:             task.ID,
				SprintID:       task.SprintID,
				EstimatedHours: task.EstimatedHours,
			})
		}
	}

	for _, record := range snapshot.EffortRecords {
		index, ok := storyIndexes[record.StoryID]
		if !ok {
			continue
		}
		metricsInput.Stories[index].EffortRecords = append(metricsInput.Stories[index].EffortRecords, metricsdomain.EffortRecordInput{
			TaskID:   cloneString(record.TaskID),
			SprintID: cloneString(record.TaskSprintID),
			Hours:    record.Hours,
		})
	}

	for _, defect := range snapshot.Defects {
		if snapshot.Sprint == nil {
			metricsInput.DefectsDetected++
			if defect.ResolutionSprintID != nil {
				metricsInput.DefectsResolved++
			}
			continue
		}
		if defect.DetectionSprintID == snapshot.Sprint.ID {
			metricsInput.DefectsDetected++
		}
		if defect.ResolutionSprintID != nil && *defect.ResolutionSprintID == snapshot.Sprint.ID {
			metricsInput.DefectsResolved++
		}
	}

	effortRecords := make([]EffortRecord, 0, len(snapshot.EffortRecords))
	for _, record := range snapshot.EffortRecords {
		if snapshot.Sprint != nil {
			if record.TaskID == nil || record.TaskSprintID == nil || *record.TaskSprintID != snapshot.Sprint.ID {
				continue
			}
		}
		effortRecords = append(effortRecords, record)
	}

	return Report{
		Project:       snapshot.Project,
		Sprint:        snapshot.Sprint,
		Stories:       snapshot.Stories,
		EffortRecords: effortRecords,
		Defects:       snapshot.Defects,
		Metrics:       metricsdomain.Calculate(metricsInput),
		GeneratedAt:   generatedAt,
	}
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
