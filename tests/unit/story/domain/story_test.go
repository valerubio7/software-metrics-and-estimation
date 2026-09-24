package domain_test

import (
	"errors"
	"reflect"
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
