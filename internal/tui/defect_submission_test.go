package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDefectCreationPostsOnceAfterExplicitConfirmationAndReloads(t *testing.T) {
	var requests []string
	var posted map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/projects/p-1/stories":
			_, _ = w.Write([]byte(`{"stories":[{"id":"story-1","title":"First story"},{"id":"story-2","title":"Chosen story"}]}`))
		case "/projects/p-1/sprints/open":
			_, _ = w.Write([]byte(`{"sprints":[{"id":"sprint-open","sprint_goal":"Open sprint","is_closed":false},{"id":"sprint-closed","sprint_goal":"Closed sprint","is_closed":true}]}`))
		case "/projects/p-1/defects":
			if r.Method == http.MethodPost {
				if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
					t.Errorf("decode POST body: %v", err)
				}
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"id":"defect-new","project_id":"p-1","story_id":"story-2","sprint_id":"sprint-open","description":"Cache crash","severity":"media","status":"en_correccion"}`))
				return
			}
			_, _ = w.Write([]byte(`{"project_id":"p-1","defects":[{"id":"defect-new","description":"Cache crash","severity":"media","status":"en_correccion","story_id":"story-2","sprint_id":"sprint-open","resolution_sprint_id":null}]}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	model := NewModel(server.URL)
	model.Screen, model.activeProjectID = "defects", "p-1"
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	for _, r := range "Cache crash" {
		model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyTab})
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyRight})
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyTab})
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyRight})
	updated, storiesCmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if storiesCmd == nil {
		t.Fatal("description form did not load project stories")
	}
	model = updateDefectTestModel(model, storiesCmd())
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyDown})
	updated, sprintsCmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if sprintsCmd == nil {
		t.Fatal("story selection did not load detection Sprints")
	}
	model = updateDefectTestModel(model, sprintsCmd())
	if len(model.defectSprintSelector.items) != 1 || model.defectSprintSelector.items[0].ID != "sprint-open" {
		t.Fatalf("detection options = %#v, want open Sprints only", model.defectSprintSelector.items)
	}
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyEnter})
	updated, confirmCmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.Screen != "defect-create-confirm" || confirmCmd != nil {
		t.Fatalf("draft confirmation = screen %q, command nil %v", model.Screen, confirmCmd == nil)
	}
	if got := strings.Join(requests, "|"); got != "GET /projects/p-1/stories|GET /projects/p-1/sprints/open" {
		t.Fatalf("requests before confirmation = %q", got)
	}
	updated, createCmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	model = updated.(Model)
	if model.Screen != "defect-create-confirm" || !model.defectCreateLoading || createCmd == nil {
		t.Fatalf("confirmation state = screen %q loading=%v command nil=%v", model.Screen, model.defectCreateLoading, createCmd == nil)
	}
	updated, duplicate := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	model = updated.(Model)
	if duplicate != nil || len(requests) != 2 {
		t.Fatalf("pending duplicate confirmation sent another request: %q", requests)
	}
	updated, reloadCmd := model.Update(createCmd())
	model = updated.(Model)
	if posted["description"] != "Cache crash" || posted["severity"] != "media" || posted["status"] != "en_correccion" || posted["story_id"] != "story-2" || posted["sprint_id"] != "sprint-open" || len(posted) != 5 {
		t.Fatalf("POST body = %#v", posted)
	}
	if got := strings.Join(requests, "|"); got != "GET /projects/p-1/stories|GET /projects/p-1/sprints/open|POST /projects/p-1/defects" {
		t.Fatalf("post-confirmation requests = %q", got)
	}
	if !strings.Contains(stripANSI(model.View()), "✓ Defecto registrado: Cache crash") {
		t.Fatalf("success view missing the toast:\n%s", stripANSI(model.View()))
	}
	if model.Screen != "defects" || reloadCmd == nil {
		t.Fatalf("success return = %q, reload command nil %v", model.Screen, reloadCmd == nil)
	}
	model = updateDefectTestModel(model, reloadCmd())
	if len(model.defects) != 1 || model.defects[0].ID != "defect-new" {
		t.Fatalf("reloaded defects = %#v", model.defects)
	}
	if got := strings.Join(requests, "|"); got != "GET /projects/p-1/stories|GET /projects/p-1/sprints/open|POST /projects/p-1/defects|GET /projects/p-1/defects" {
		t.Fatalf("final request sequence = %q", got)
	}
}

func TestDefectCreationCancelAtConfirmationDoesNotPost(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  tea.KeyMsg
		want string
	}{
		{name: "cancel draft", key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}, want: "defects"},
		{name: "back to sprint choice", key: tea.KeyMsg{Type: tea.KeyEsc}, want: "defect-create-form"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := NewModel("http://example.invalid")
			model.Screen, model.activeProjectID = "defect-create-confirm", "p-1"
			updated, cmd := model.Update(tc.key)
			model = updated.(Model)
			if cmd != nil || model.Screen != tc.want {
				t.Fatalf("cancel/back = screen %q command nil %v, want %q without a request", model.Screen, cmd == nil, tc.want)
			}
		})
	}
}

func TestDefectCreateRejectionPreservesAPIStatusAndBody(t *testing.T) {
	const responseBody = `{"error":"validation_failed","message":"one or more fields are invalid","fields":{"description":"is required"}}`
	model := NewModel("http://example.invalid")
	model.Screen, model.activeProjectID, model.defectCreateLoading = "defect-create-confirm", "p-1", true
	updated, _ := model.Update(defectCreateResult{ProjectID: "p-1", Response: APIResponse{StatusCode: http.StatusUnprocessableEntity, Body: []byte(responseBody)}})
	model = updated.(Model)
	model.width = 240
	view := model.View()
	for _, fragment := range []string{"Error HTTP 422", responseBody} {
		if !strings.Contains(view, fragment) {
			t.Errorf("result view does not preserve %q:\n%s", fragment, view)
		}
	}
	if model.Screen != "defect-create-confirm" || model.toast != "" {
		t.Fatalf("API validation failure left the confirmation or reported success: screen=%q toast=%q", model.Screen, model.toast)
	}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if cmd != nil || model.Screen != "defect-create-form" {
		t.Fatalf("API error did not allow returning to the editable draft: screen=%q command nil=%v", model.Screen, cmd == nil)
	}
}

func TestStaleDefectCreateResultCannotCompleteNewerSubmission(t *testing.T) {
	model := NewModel("http://example.invalid")
	model.activeProjectID = "p-1"
	model.defectCreateForm = defectCreateForm{severityIndex: 0}
	firstModel, _ := model.submitDefectCreate()
	secondModel, _ := firstModel.submitDefectCreate()
	firstResult := defectCreateResult{ProjectID: "p-1", RequestID: firstModel.defectCreateRequestID, Response: APIResponse{StatusCode: http.StatusBadRequest, Body: []byte("stale result")}}
	secondResult := defectCreateResult{ProjectID: "p-1", RequestID: secondModel.defectCreateRequestID, Response: APIResponse{StatusCode: http.StatusCreated, Body: []byte("new result")}}

	updated, _ := secondModel.Update(firstResult)
	model = updated.(Model)
	if !model.defectCreateLoading || model.defectCreateMessage != "" {
		t.Fatalf("stale submission result completed the newer pending operation: loading=%v message=%q", model.defectCreateLoading, model.defectCreateMessage)
	}
	updated, _ = model.Update(secondResult)
	model = updated.(Model)
	if model.defectCreateLoading || model.Screen != "defects" || !strings.HasPrefix(model.toast, "Defecto registrado") {
		t.Fatalf("newer submission result was not applied: loading=%v screen=%q toast=%q", model.defectCreateLoading, model.Screen, model.toast)
	}
}
