package domain_test

import (
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
)

func TestNewStoryPreservesValidInputAndInitializesPending(t *testing.T) {
	for _, priority := range []string{"alta", "media", "baja"} {
		t.Run(priority, func(t *testing.T) {
			criteria := []string{"  Primer criterio ", "Segundo criterio"}
			story, err := domain.NewStory("story-id", "project-id", " Título ", " Descripción ", priority, criteria)
			if err != nil {
				t.Fatalf("NewStory() error = %v", err)
			}
			if story.ID != "story-id" || story.ProjectID != "project-id" || story.Title != " Título " || story.Description != " Descripción " || story.Priority != priority {
				t.Errorf("story fields = %#v, want unchanged inputs", story)
			}
			if story.Status != "pendiente" || story.StoryPoints != nil {
				t.Errorf("initial status/points = %q/%v, want pendiente/nil", story.Status, story.StoryPoints)
			}
			if !reflect.DeepEqual(story.AcceptanceCriteria, criteria) {
				t.Errorf("criteria = %#v, want %#v", story.AcceptanceCriteria, criteria)
			}
		})
	}
}

func TestNewStoryRejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name, title, description, priority string
		criteria                           []string
		field                              string
	}{
		{name: "missing title", description: "Descripción", priority: "media", criteria: []string{"Hecho"}, field: "title"},
		{name: "ASCII blank title", title: " \t\n", description: "Descripción", priority: "media", criteria: []string{"Hecho"}, field: "title"},
		{name: "Unicode blank title", title: "\u2003", description: "Descripción", priority: "media", criteria: []string{"Hecho"}, field: "title"},
		{name: "missing description", title: "Título", priority: "media", criteria: []string{"Hecho"}, field: "description"},
		{name: "ASCII blank description", title: "Título", description: " \t", priority: "media", criteria: []string{"Hecho"}, field: "description"},
		{name: "Unicode blank description", title: "Título", description: "\u00a0", priority: "media", criteria: []string{"Hecho"}, field: "description"},
		{name: "missing priority", title: "Título", description: "Descripción", criteria: []string{"Hecho"}, field: "priority"},
		{name: "unknown priority", title: "Título", description: "Descripción", priority: "urgente", criteria: []string{"Hecho"}, field: "priority"},
		{name: "case sensitive priority", title: "Título", description: "Descripción", priority: "Alta", criteria: []string{"Hecho"}, field: "priority"},
		{name: "nil criteria", title: "Título", description: "Descripción", priority: "media", field: "acceptance_criteria"},
		{name: "empty criteria", title: "Título", description: "Descripción", priority: "media", criteria: []string{}, field: "acceptance_criteria"},
		{name: "empty entry", title: "Título", description: "Descripción", priority: "media", criteria: []string{"Hecho", ""}, field: "acceptance_criteria"},
		{name: "blank entry", title: "Título", description: "Descripción", priority: "media", criteria: []string{" \t", "Hecho"}, field: "acceptance_criteria"},
		{name: "Unicode blank criterion", title: "Título", description: "Descripción", priority: "media", criteria: []string{"Hecho", "\u2003\u00a0"}, field: "acceptance_criteria"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewStory("story-id", "project-id", tt.title, tt.description, tt.priority, tt.criteria)
			var validation *domain.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("NewStory() error = %v, want ValidationError", err)
			}
			if validation.Fields[tt.field] == "" {
				t.Errorf("validation fields = %#v, want %q", validation.Fields, tt.field)
			}
		})
	}
}

const (
	validTitle       = "Título"
	validDescription = "Descripción"
	estimatedHoursOK = "must be greater than 0, at most 99999.99 and have at most 2 decimals"
)

func floatPtr(v float64) *float64 { return &v }

func TestNewStoryUpdate(t *testing.T) {
	tests := []struct {
		name, title, description, priority, status string
		criteria                                   []string
		wantFields                                 map[string]string
	}{
		{name: "valid pendiente", title: validTitle, description: validDescription, priority: "media", status: "pendiente", criteria: []string{"Hecho"}},
		{name: "valid en_progreso", title: validTitle, description: validDescription, priority: "alta", status: "en_progreso", criteria: []string{"Hecho"}},
		{name: "valid completada", title: validTitle, description: validDescription, priority: "baja", status: "completada", criteria: []string{"Hecho"}},
		{name: "empty title", description: validDescription, priority: "media", status: "pendiente", criteria: []string{"Hecho"}, wantFields: map[string]string{"title": "is required"}},
		{name: "ASCII blank title", title: " \t\n", description: validDescription, priority: "media", status: "pendiente", criteria: []string{"Hecho"}, wantFields: map[string]string{"title": "is required"}},
		{name: "Unicode blank title", title: " ", description: validDescription, priority: "media", status: "pendiente", criteria: []string{"Hecho"}, wantFields: map[string]string{"title": "is required"}},
		{name: "empty description", title: validTitle, priority: "media", status: "pendiente", criteria: []string{"Hecho"}, wantFields: map[string]string{"description": "is required"}},
		{name: "ASCII blank description", title: validTitle, description: "   ", priority: "media", status: "pendiente", criteria: []string{"Hecho"}, wantFields: map[string]string{"description": "is required"}},
		{name: "Unicode blank description", title: validTitle, description: " ", priority: "media", status: "pendiente", criteria: []string{"Hecho"}, wantFields: map[string]string{"description": "is required"}},
		{name: "unknown priority", title: validTitle, description: validDescription, priority: "urgente", status: "pendiente", criteria: []string{"Hecho"}, wantFields: map[string]string{"priority": "must be alta, media or baja"}},
		{name: "case sensitive priority", title: validTitle, description: validDescription, priority: "ALTA", status: "pendiente", criteria: []string{"Hecho"}, wantFields: map[string]string{"priority": "must be alta, media or baja"}},
		{name: "empty priority", title: validTitle, description: validDescription, status: "pendiente", criteria: []string{"Hecho"}, wantFields: map[string]string{"priority": "must be alta, media or baja"}},
		{name: "status outside the set", title: validTitle, description: validDescription, priority: "media", status: "cancelada", criteria: []string{"Hecho"}, wantFields: map[string]string{"status": "must be pendiente, en_progreso or completada"}},
		{name: "case sensitive status", title: validTitle, description: validDescription, priority: "media", status: "Completada", criteria: []string{"Hecho"}, wantFields: map[string]string{"status": "must be pendiente, en_progreso or completada"}},
		{name: "status with a space", title: validTitle, description: validDescription, priority: "media", status: "en progreso", criteria: []string{"Hecho"}, wantFields: map[string]string{"status": "must be pendiente, en_progreso or completada"}},
		{name: "empty status", title: validTitle, description: validDescription, priority: "media", criteria: []string{"Hecho"}, wantFields: map[string]string{"status": "must be pendiente, en_progreso or completada"}},
		{name: "nil criteria", title: validTitle, description: validDescription, priority: "media", status: "pendiente", wantFields: map[string]string{"acceptance_criteria": "must contain at least one non-blank criterion"}},
		{name: "empty criteria", title: validTitle, description: validDescription, priority: "media", status: "pendiente", criteria: []string{}, wantFields: map[string]string{"acceptance_criteria": "must contain at least one non-blank criterion"}},
		{name: "empty criterion entry", title: validTitle, description: validDescription, priority: "media", status: "pendiente", criteria: []string{"Hecho", ""}, wantFields: map[string]string{"acceptance_criteria": "must contain only non-blank criteria"}},
		{name: "blank criterion entry", title: validTitle, description: validDescription, priority: "media", status: "pendiente", criteria: []string{"   ", "Hecho"}, wantFields: map[string]string{"acceptance_criteria": "must contain only non-blank criteria"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			story, err := domain.NewStoryUpdate("story-id", "project-id", tt.title, tt.description, tt.priority, tt.status, tt.criteria, nil)
			if tt.wantFields == nil {
				if err != nil {
					t.Fatalf("NewStoryUpdate() error = %v, want nil", err)
				}
				if story.Status != tt.status || story.Priority != tt.priority {
					t.Errorf("story status/priority = %q/%q, want %q/%q", story.Status, story.Priority, tt.status, tt.priority)
				}
				return
			}
			var validation *domain.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("NewStoryUpdate() error = %v, want ValidationError", err)
			}
			if !reflect.DeepEqual(validation.Fields, tt.wantFields) {
				t.Errorf("validation fields = %#v, want %#v", validation.Fields, tt.wantFields)
			}
			if !reflect.DeepEqual(story, domain.Story{}) {
				t.Errorf("story = %#v, want zero value on validation failure", story)
			}
		})
	}
}

func TestNewStoryUpdateEstimatedHours(t *testing.T) {
	tests := []struct {
		name  string
		hours *float64
		valid bool
	}{
		{name: "absent is valid", hours: nil, valid: true},
		{name: "zero", hours: floatPtr(0)},
		{name: "negative integer", hours: floatPtr(-1)},
		{name: "negative decimal", hours: floatPtr(-0.01)},
		{name: "three decimals", hours: floatPtr(1.001)},
		{name: "below one cent", hours: floatPtr(0.005)},
		{name: "above the maximum", hours: floatPtr(100000)},
		{name: "above the maximum by a fraction", hours: floatPtr(99999.991)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			story, err := domain.NewStoryUpdate("story-id", "project-id", validTitle, validDescription, "media", "pendiente", []string{"Hecho"}, tt.hours)
			if tt.valid {
				if err != nil {
					t.Fatalf("NewStoryUpdate() error = %v, want nil", err)
				}
				if story.EstimatedHours != nil {
					t.Errorf("EstimatedHours = %v, want nil", *story.EstimatedHours)
				}
				return
			}
			var validation *domain.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("NewStoryUpdate() error = %v, want ValidationError", err)
			}
			want := map[string]string{"estimated_hours": estimatedHoursOK}
			if !reflect.DeepEqual(validation.Fields, want) {
				t.Errorf("validation fields = %#v, want %#v", validation.Fields, want)
			}
		})
	}
}

func TestNewStoryUpdateAcceptsEstimatedHoursWithinRange(t *testing.T) {
	for _, hours := range []float64{0.01, 0.07, 2.5, 3, 8.5, 99999.99} {
		t.Run(strconv.FormatFloat(hours, 'f', -1, 64), func(t *testing.T) {
			value := hours
			story, err := domain.NewStoryUpdate("story-id", "project-id", validTitle, validDescription, "media", "pendiente", []string{"Hecho"}, &value)
			if err != nil {
				t.Fatalf("NewStoryUpdate() error = %v, want nil", err)
			}
			if story.EstimatedHours == nil || *story.EstimatedHours != hours {
				t.Errorf("EstimatedHours = %v, want %v", story.EstimatedHours, hours)
			}
		})
	}
}

func TestNewStoryUpdateCopiesEstimatedHours(t *testing.T) {
	hours := 4.0
	story, err := domain.NewStoryUpdate("story-id", "project-id", validTitle, validDescription, "media", "pendiente", []string{"Hecho"}, &hours)
	if err != nil {
		t.Fatalf("NewStoryUpdate() error = %v", err)
	}
	hours = 99
	if story.EstimatedHours == nil || *story.EstimatedHours != 4 {
		t.Errorf("EstimatedHours = %v, want 4: story must not alias the caller's pointer", story.EstimatedHours)
	}
}

func TestNewStoryUpdateRejectsUnicodeBlankCriterion(t *testing.T) {
	for _, blank := range []string{" ", " ", "  "} {
		_, err := domain.NewStoryUpdate("story-id", "project-id", validTitle, validDescription, "media", "pendiente", []string{"Hecho", blank}, nil)
		var validation *domain.ValidationError
		if !errors.As(err, &validation) {
			t.Fatalf("NewStoryUpdate(%q) error = %v, want ValidationError", blank, err)
		}
		if got := validation.Fields["acceptance_criteria"]; got != "must contain only non-blank criteria" {
			t.Errorf("acceptance_criteria for %q = %q, want the non-blank message", blank, got)
		}
	}
}

func TestNewStoryUpdateAccumulatesAllInvalidFields(t *testing.T) {
	zero := 0.0
	_, err := domain.NewStoryUpdate("story-id", "project-id", "", "   ", "urgente", "cancelada", []string{}, &zero)
	var validation *domain.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("NewStoryUpdate() error = %v, want ValidationError", err)
	}
	want := map[string]string{
		"title":               "is required",
		"description":         "is required",
		"priority":            "must be alta, media or baja",
		"status":              "must be pendiente, en_progreso or completada",
		"acceptance_criteria": "must contain at least one non-blank criterion",
		"estimated_hours":     estimatedHoursOK,
	}
	if !reflect.DeepEqual(validation.Fields, want) {
		t.Errorf("validation fields = %#v, want %#v", validation.Fields, want)
	}
}

func TestNewStoryUpdatePreservesTextOrderAndIdentity(t *testing.T) {
	criteria := []string{"  Segundo ", "Primero — ñ", "Tercero"}
	hours := 8.5
	story, err := domain.NewStoryUpdate("story-id", "project-id", "  Título  con   espacios — ñ ", " Descripción\ncon Unicode ✓ ", "alta", "en_progreso", criteria, &hours)
	if err != nil {
		t.Fatalf("NewStoryUpdate() error = %v", err)
	}
	if story.ID != "story-id" || story.ProjectID != "project-id" {
		t.Errorf("identity = %q/%q, want story-id/project-id", story.ID, story.ProjectID)
	}
	if story.Title != "  Título  con   espacios — ñ " || story.Description != " Descripción\ncon Unicode ✓ " {
		t.Errorf("text = %q/%q, want unchanged inputs", story.Title, story.Description)
	}
	if !reflect.DeepEqual(story.AcceptanceCriteria, criteria) {
		t.Errorf("criteria = %#v, want %#v in the same order", story.AcceptanceCriteria, criteria)
	}
	if story.StoryPoints != nil {
		t.Errorf("StoryPoints = %v, want nil: story points are not editable here", *story.StoryPoints)
	}
	if story.EstimatedHours == nil || *story.EstimatedHours != 8.5 {
		t.Errorf("EstimatedHours = %v, want 8.5", story.EstimatedHours)
	}
}

func TestNewStoryUpdateCopiesCriteria(t *testing.T) {
	criteria := []string{"Uno", "Dos"}
	story, err := domain.NewStoryUpdate("story-id", "project-id", validTitle, validDescription, "media", "pendiente", criteria, nil)
	if err != nil {
		t.Fatalf("NewStoryUpdate() error = %v", err)
	}
	criteria[0] = "mutado"
	if story.AcceptanceCriteria[0] != "Uno" {
		t.Errorf("criteria[0] = %q, want Uno: story must not alias the caller's slice", story.AcceptanceCriteria[0])
	}
}

func TestNewStoryInitializesWithoutEstimatedHours(t *testing.T) {
	story, err := domain.NewStory("story-id", "project-id", validTitle, validDescription, "media", []string{"Hecho"})
	if err != nil {
		t.Fatalf("NewStory() error = %v", err)
	}
	if story.EstimatedHours != nil {
		t.Errorf("EstimatedHours = %v, want nil", *story.EstimatedHours)
	}
	if story.Status != domain.StatusPending {
		t.Errorf("Status = %q, want %q", story.Status, domain.StatusPending)
	}
}

func TestAllowedStatusesReturnsTheClosedSetAsACopy(t *testing.T) {
	want := []string{"pendiente", "en_progreso", "completada"}
	got := domain.AllowedStatuses()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AllowedStatuses() = %#v, want %#v", got, want)
	}
	got[0] = "mutado"
	_ = append(got, "extra")
	if again := domain.AllowedStatuses(); !reflect.DeepEqual(again, want) {
		t.Errorf("AllowedStatuses() after mutating a previous result = %#v, want %#v", again, want)
	}
}
