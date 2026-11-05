package tui

import (
	"net/http"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDefectCreationCanBeCancelledWithoutMutation(t *testing.T) {
	model := NewModel("http://example.invalid")
	model.Screen, model.activeProjectID = "story-detail", "p-1"
	model.backlogStories = []Story{{ID: "story-1", Title: "Story"}}
	model.storyDetail = storyDetailState{storyID: "story-1"}

	updated, cmd := model.Update(runeKey("d"))
	model = updated.(Model)
	if cmd != nil || model.Screen != "defect-create-form" || model.defectCreateForm.storyID != "story-1" {
		t.Fatalf("opening the form = screen %q story %q cmd nil %v", model.Screen, model.defectCreateForm.storyID, cmd == nil)
	}

	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.Screen != "story-detail" {
		t.Fatalf("Escape did not cancel the draft: screen = %q, cmd nil = %v", model.Screen, cmd == nil)
	}
}

func TestDefectDetectionSprintSelectorExcludesClosedSprints(t *testing.T) {
	model := NewModel("http://example.invalid")
	model.Screen, model.activeProjectID = "defect-sprint-select", "p-1"
	model.defectSprintSelector.startLoading()
	result := defectOptionsResult{
		step: "sprints", projectID: "p-1",
		response: APIResponse{StatusCode: http.StatusOK, Body: []byte(`{"sprints":[{"id":"open","sprint_goal":"Open","is_closed":false},{"id":"closed","sprint_goal":"Closed","is_closed":true}]}`)},
	}
	model = updateDefectTestModel(model, result)
	if len(model.defectSprintSelector.items) != 1 || model.defectSprintSelector.items[0].ID != "open" {
		t.Fatalf("detection Sprint options = %#v, want only the open Sprint", model.defectSprintSelector.items)
	}
}

func TestDefectCreationFormOffersAPIRegistrationCatalogs(t *testing.T) {
	model := NewModel("http://example.invalid")
	model.Screen, model.activeProjectID = "story-detail", "p-1"
	model.backlogStories = []Story{{ID: "story-1", Title: "Story"}}
	model.storyDetail = storyDetailState{storyID: "story-1"}
	model, _ = storyStep(model, runeKey("d"))
	for _, r := range "Defect description" {
		model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if model.defectCreateForm.description != "Defect description" || !strings.Contains(model.View(), "Defect description") {
		t.Fatal("creation form did not retain the entered description")
	}
	for _, want := range []string{"alta", "detectado"} {
		if !strings.Contains(model.View(), want) {
			t.Errorf("creation form default does not contain %q", want)
		}
	}
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyTab})
	for _, want := range []string{"media", "baja", "alta"} {
		model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyRight})
		if !strings.Contains(model.View(), "Severidad: "+want) {
			t.Errorf("severity option = %q, want %q", model.defectCreateForm.severity(), want)
		}
	}
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyTab})
	for _, want := range []string{"en_correccion", "detectado"} {
		model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyRight})
		if !strings.Contains(model.View(), "Estado inicial: "+want) {
			t.Errorf("initial status option = %q, want %q", model.defectCreateForm.status(), want)
		}
	}
}

func TestUnsubmittedDefectDraftNeverClaimsReadyForReview(t *testing.T) {
	for _, tc := range []struct {
		name        string
		description string
	}{
		{name: "empty description"},
		{name: "entered description", description: "Cache crash"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := NewModel("http://example.invalid")
			model.Screen = "defect-create-form"
			model.defectCreateForm = defectCreateForm{
				description: tc.description,
				storyID:     "story-1", storyLabel: "Story",
				sprintID: "sprint-1", sprintLabel: "Open sprint",
			}
			view := model.View()
			if !strings.Contains(view, "Borrador sin registrar") {
				t.Fatalf("unsubmitted draft copy does not state its status:\n%s", view)
			}
			if strings.Contains(view, "listo para revisión") {
				t.Fatalf("unsubmitted draft claims readiness:\n%s", view)
			}
			if model.Screen != "defect-create-form" || model.defectCreateForm.description != tc.description {
				t.Fatalf("draft description was changed or rejected: screen=%q description=%q", model.Screen, model.defectCreateForm.description)
			}
		})
	}
}

func TestStaleSameProjectDefectSprintResultsAreIgnored(t *testing.T) {
	model := NewModel("http://example.invalid")
	model.activeProjectID = "p-1"
	firstModel, firstCmd := model.loadDefectOpenSprints()
	secondModel, secondCmd := firstModel.loadDefectOpenSprints()
	if firstCmd == nil || secondCmd == nil {
		t.Fatal("selector load did not return commands")
	}
	firstResult := defectOptionsResult{step: "sprints", projectID: "p-1", requestID: firstModel.defectSprintRequestID,
		response: APIResponse{StatusCode: http.StatusOK, Body: []byte(`{"sprints":[{"id":"sprint-stale","sprint_goal":"Stale sprint","is_closed":false}]}`)}}
	secondResult := defectOptionsResult{step: "sprints", projectID: "p-1", requestID: secondModel.defectSprintRequestID,
		response: APIResponse{StatusCode: http.StatusOK, Body: []byte(`{"sprints":[{"id":"sprint-new","sprint_goal":"New sprint","is_closed":false}]}`)}}

	pending, _ := secondModel.applyDefectOptionsResult(firstResult)
	if len(pending.defectSprintSelector.items) != 0 || !pending.defectSprintSelector.loading {
		t.Fatalf("stale result completed the newer request: items=%#v loading=%v", pending.defectSprintSelector.items, pending.defectSprintSelector.loading)
	}
	completed, _ := pending.applyDefectOptionsResult(secondResult)
	items := completed.defectSprintSelector.items
	if len(items) != 1 || items[0].ID != "sprint-new" {
		t.Fatalf("newer result not applied: %#v", items)
	}
}
