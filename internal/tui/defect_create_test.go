package tui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDefectCreationCanBeCancelledWithoutMutation(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/projects/p-1/stories":
			_, _ = w.Write([]byte(`{"stories":[{"id":"story-1","title":"Story"}]}`))
		case "/projects/p-1/sprints/open":
			_, _ = w.Write([]byte(`{"sprints":[{"id":"sprint-1","sprint_goal":"Open","is_closed":false}]}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	model := NewModel(server.URL)
	model.Screen, model.activeProjectID = "defects", "p-1"
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	model = defectFormToStories(t, model)
	updated, sprintsCmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if sprintsCmd == nil {
		t.Fatal("selecting a story produced no open-Sprint request")
	}
	model = updateDefectTestModel(model, sprintsCmd())
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.Screen != "defect-create-form" || !strings.Contains(model.View(), "Story") || !strings.Contains(model.View(), "Open") {
		t.Fatalf("draft did not show selected references: screen=%q\n%s", model.Screen, model.View())
	}
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.Screen != "defects" {
		t.Fatalf("Escape did not cancel the draft: screen = %q", model.Screen)
	}
	if got := strings.Join(methods, "|"); got != "GET /projects/p-1/stories|GET /projects/p-1/sprints/open" {
		t.Fatalf("cancel sequence issued a mutation: %q", got)
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

func defectFormToStories(t *testing.T, model Model) Model {
	t.Helper()
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyTab})
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyTab})
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("submitting the defect details produced no story-list request")
	}
	return updateDefectTestModel(model, cmd())
}

func TestDefectCreationFormOffersAPIRegistrationCatalogs(t *testing.T) {
	model := NewModel("http://example.invalid")
	model.Screen, model.activeProjectID = "defects", "p-1"
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
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

func TestStaleSameProjectDefectSelectorResultsAreIgnored(t *testing.T) {
	for _, tc := range []struct {
		name      string
		start     func(Model) (Model, tea.Cmd)
		apply     func(Model, defectOptionsResult) (Model, tea.Cmd)
		step      string
		body      string
		requestID func(Model) uint64
		items     func(Model) []selectorItem
	}{
		{
			name:  "story options",
			start: func(model Model) (Model, tea.Cmd) { return model.loadDefectStories() },
			apply: func(model Model, result defectOptionsResult) (Model, tea.Cmd) {
				return model.applyDefectOptionsResult(result)
			},
			step: "stories", body: `{"stories":[{"id":"story-new","title":"New story"}]}`,
			requestID: func(model Model) uint64 { return model.defectStoryRequestID },
			items:     func(model Model) []selectorItem { return model.defectStorySelector.items },
		},
		{
			name:  "open Sprint options",
			start: func(model Model) (Model, tea.Cmd) { return model.loadDefectOpenSprints() },
			apply: func(model Model, result defectOptionsResult) (Model, tea.Cmd) {
				return model.applyDefectOptionsResult(result)
			},
			step: "sprints", body: `{"sprints":[{"id":"sprint-new","sprint_goal":"New sprint","is_closed":false}]}`,
			requestID: func(model Model) uint64 { return model.defectSprintRequestID },
			items:     func(model Model) []selectorItem { return model.defectSprintSelector.items },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := NewModel("http://example.invalid")
			model.activeProjectID = "p-1"
			firstModel, firstCmd := tc.start(model)
			secondModel, secondCmd := tc.start(firstModel)
			if firstCmd == nil || secondCmd == nil {
				t.Fatal("selector load did not return commands")
			}
			staleBody := `{"stories":[{"id":"story-stale","title":"Stale story"}],"sprints":[{"id":"sprint-stale","sprint_goal":"Stale sprint","is_closed":false}]}`
			firstResult := defectOptionsResult{step: tc.step, projectID: "p-1", requestID: tc.requestID(firstModel), response: APIResponse{StatusCode: http.StatusOK, Body: []byte(staleBody)}}
			secondResult := defectOptionsResult{step: tc.step, projectID: "p-1", requestID: tc.requestID(secondModel), response: APIResponse{StatusCode: http.StatusOK, Body: []byte(tc.body)}}

			pending, _ := tc.apply(secondModel, firstResult)
			if len(tc.items(pending)) != 0 || !selectorLoading(pending, tc.step) {
				t.Fatalf("stale %s result completed the newer request: items=%#v loading=%v", tc.step, tc.items(pending), selectorLoading(pending, tc.step))
			}
			completed, _ := tc.apply(pending, secondResult)
			items := tc.items(completed)
			if len(items) != 1 || items[0].ID == "story-stale" || items[0].ID == "sprint-stale" {
				t.Fatalf("newer %s result not applied: %#v", tc.step, items)
			}
		})
	}
}

func selectorLoading(model Model, step string) bool {
	if step == "stories" {
		return model.defectStorySelector.loading
	}
	return model.defectSprintSelector.loading
}
