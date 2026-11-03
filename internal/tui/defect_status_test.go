package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDefectDetailOffersExplicitStatusUpdateAction(t *testing.T) {
	model := NewModel("http://example.invalid")
	model.Screen, model.activeProjectID = "defect-detail", "p-1"
	model.defectDetail = Defect{ID: "d-1", Status: "detectado"}
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	model = updated.(Model)
	if model.Screen != "defect-status-select" || cmd != nil {
		t.Fatalf("explicit update action = screen %q, command nil %v; want status choice and no mutation", model.Screen, cmd == nil)
	}
}

func TestDefectListStatusActionUsesHighlightedDefectWithoutMutating(t *testing.T) {
	model := NewModel("http://example.invalid")
	model.Screen, model.activeProjectID = "defects", "p-1"
	model.defects = []Defect{{ID: "d-1"}, {ID: "d-2"}}
	model.defectCursor = 1
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	model = updated.(Model)
	if model.Screen != "defect-status-select" || model.defectDetail.ID != "d-2" || cmd != nil {
		t.Fatalf("list status action = screen %q defect %q command nil %v; want status choice for highlighted defect without mutation", model.Screen, model.defectDetail.ID, cmd == nil)
	}
}

func TestDefectStatusUpdatePatchesSupportedStatesAndReloads(t *testing.T) {
	tests := []struct {
		name       string
		index      int
		visible    string
		apiStatus  string
		wantSprint string
		wantCalls  string
	}{
		{name: "open clears resolution", visible: "Open", apiStatus: "detectado", wantCalls: "PATCH /projects/p-1/defects/d-1|GET /projects/p-1/defects"},
		{name: "in progress clears resolution", index: 1, visible: "In Progress", apiStatus: "en_correccion", wantCalls: "PATCH /projects/p-1/defects/d-1|GET /projects/p-1/defects"},
		{name: "resolved includes selected resolution Sprint", index: 2, visible: "Resolved", apiStatus: "resuelto", wantSprint: "closed-sprint", wantCalls: "GET /projects/p-1/sprints/open|GET /projects/p-1/sprints|PATCH /projects/p-1/defects/d-1|GET /projects/p-1/defects"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var requests []string
			var patch map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.Method+" "+r.URL.Path)
				if r.Method == http.MethodPatch {
					if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
						t.Errorf("decode PATCH body: %v", err)
					}
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"id":"d-1","status":"` + tc.apiStatus + `"}`))
					return
				}
				if r.Method == http.MethodGet && r.URL.Path == "/projects/p-1/sprints/open" {
					_, _ = w.Write([]byte(`{"sprints":[{"id":"open-sprint","sprint_goal":"Open sprint","is_closed":false}]}`))
					return
				}
				if r.Method == http.MethodGet && r.URL.Path == "/projects/p-1/sprints" {
					_, _ = w.Write([]byte(`{"project_id":"p-1","sprints":[{"id":"closed-sprint","sprint_goal":"Closed sprint","is_closed":true}]}`))
					return
				}
				if r.Method == http.MethodGet && r.URL.Path == "/projects/p-1/defects" {
					_, _ = w.Write([]byte(`{"project_id":"p-1","defects":[{"id":"d-1","description":"Updated defect","severity":"media","status":"` + tc.apiStatus + `","story_id":"story-1","sprint_id":"detect-sprint","resolution_sprint_id":null}]}`))
					return
				}
				t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				http.NotFound(w, r)
			}))
			defer server.Close()

			model := NewModel(server.URL)
			model.Screen, model.activeProjectID = "defect-detail", "p-1"
			model.defectDetail = Defect{ID: "d-1", Status: "detectado", ResolutionSprintID: stringPointer("old-resolution")}
			model = updateStatusTestModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
			for i := 0; i < tc.index; i++ {
				model = updateStatusTestModel(model, tea.KeyMsg{Type: tea.KeyDown})
			}
			if !strings.Contains(model.View(), tc.visible) {
				t.Fatalf("status options do not show %q:\n%s", tc.visible, model.View())
			}
			if strings.Contains(model.View(), "Closed") {
				t.Fatalf("unsupported Closed state was offered:\n%s", model.View())
			}
			updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
			model = updated.(Model)
			if tc.index == 2 {
				if cmd == nil {
					t.Fatal("Resolved selection did not load resolution Sprints")
				}
				model = updateStatusTestModel(model, cmd())
				if len(model.defectResolutionSelector.items) != 2 {
					t.Fatalf("resolution options = %#v, want active project's open and closed Sprints", model.defectResolutionSelector.items)
				}
				if model.defectResolutionSelector.items[0].ID != "open-sprint" || model.defectResolutionSelector.items[1].ID != "closed-sprint" {
					t.Fatalf("resolution options = %#v", model.defectResolutionSelector.items)
				}
				model = updateStatusTestModel(model, tea.KeyMsg{Type: tea.KeyDown})
				updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
				model = updated.(Model)
			}
			if cmd == nil {
				t.Fatal("status selection did not submit PATCH")
			}
			updated, reload := model.Update(cmd())
			model = updated.(Model)
			if patch["status"] != tc.apiStatus {
				t.Fatalf("PATCH body = %#v, status want %q", patch, tc.apiStatus)
			}
			if tc.wantSprint != "" {
				if patch["resolution_sprint_id"] != tc.wantSprint {
					t.Fatalf("PATCH resolution = %#v, want %q", patch["resolution_sprint_id"], tc.wantSprint)
				}
			} else if _, exists := patch["resolution_sprint_id"]; exists {
				t.Fatalf("non-resolved PATCH included resolution Sprint: %#v", patch)
			}
			if model.Screen != "defects" || !model.defectLoading || !model.defectStatusOK || !strings.Contains(model.defectStatusMessage, "HTTP 200") || reload == nil {
				t.Fatalf("success did not confirm and reload list: screen=%q loading=%v ok=%v message=%q cmd nil=%v", model.Screen, model.defectLoading, model.defectStatusOK, model.defectStatusMessage, reload == nil)
			}
			model = updateStatusTestModel(model, reload())
			if len(model.defects) != 1 || model.defects[0].Status != tc.apiStatus {
				t.Fatalf("reloaded defects = %#v", model.defects)
			}
			if got := strings.Join(requests, "|"); got != tc.wantCalls {
				t.Fatalf("request sequence = %q, want %q", got, tc.wantCalls)
			}
		})
	}
}

func TestDefectStatusCancelAndDuplicateSubmitDoNotMutate(t *testing.T) {
	patchCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			patchCount++
		}
		_, _ = w.Write([]byte(`{"id":"d-1","status":"detectado"}`))
	}))
	defer server.Close()
	model := NewModel(server.URL)
	model.Screen, model.activeProjectID = "defect-detail", "p-1"
	model.defectDetail = Defect{ID: "d-1"}
	model = updateStatusTestModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	model = updateStatusTestModel(model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.Screen != "defect-detail" || model.defectStatusLoading || patchCount != 0 {
		t.Fatalf("cancel state = screen %q loading=%v PATCH count=%d", model.Screen, model.defectStatusLoading, patchCount)
	}
	model = updateStatusTestModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	updated, first := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if first == nil {
		t.Fatal("first status submit did not start PATCH")
	}
	updated, duplicate := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if duplicate != nil || !model.defectStatusLoading {
		t.Fatalf("duplicate submission started a request: cmd nil=%v loading=%v", duplicate == nil, model.defectStatusLoading)
	}
	_ = first()
	if patchCount != 1 {
		t.Fatalf("pending submission sent %d PATCH requests, want exactly one", patchCount)
	}
}

func TestDefectStatusFailureShowsAPIStatusAndBody(t *testing.T) {
	const responseBody = `{"error":"validation_failed","message":"resolution Sprint predates detection"}`
	model := NewModel("http://example.invalid")
	model.Screen, model.activeProjectID, model.defectStatusLoading = "defect-status-result", "p-1", true
	model.defectStatusRequestID = 5
	model = updateStatusTestModel(model, defectStatusResult{ProjectID: "p-1", RequestID: 5, Response: APIResponse{StatusCode: http.StatusUnprocessableEntity, Body: []byte(responseBody)}})
	model.width = 240
	view := model.View()
	for _, fragment := range []string{"Error HTTP 422", responseBody} {
		if !strings.Contains(view, fragment) {
			t.Errorf("failure view does not preserve %q:\n%s", fragment, view)
		}
	}
}

func TestStaleSameProjectDefectStatusResultsAreIgnored(t *testing.T) {
	model := NewModel("http://example.invalid")
	model.Screen, model.activeProjectID = "defect-status-select", "p-1"
	model.defectDetail = Defect{ID: "d-1"}
	first, _ := model.loadDefectResolutionSprints()
	second, _ := first.loadDefectResolutionSprints()
	updated, _ := second.Update(defectStatusOptionsResult{ProjectID: "p-1", RequestID: first.defectStatusOptionsRequestID})
	model = updated.(Model)
	if !model.defectStatusOptionsLoading || model.defectStatusOptionsRequestID != second.defectStatusOptionsRequestID {
		t.Fatal("stale same-project selector result completed the newer operation")
	}
	firstUpdate, _ := model.submitDefectStatus("detectado", "")
	firstUpdate.defectStatusLoading = false
	firstUpdate.Screen = "defect-status-select"
	secondUpdate, _ := firstUpdate.submitDefectStatus("en_correccion", "")
	updated, _ = secondUpdate.Update(defectStatusResult{ProjectID: "p-1", RequestID: firstUpdate.defectStatusRequestID, Response: APIResponse{StatusCode: http.StatusOK, Body: []byte("stale")}})
	model = updated.(Model)
	if !model.defectStatusLoading || model.defectStatusMessage != "" {
		t.Fatal("stale same-project PATCH result completed the newer operation")
	}
}

func updateStatusTestModel(model Model, msg tea.Msg) Model {
	updated, _ := model.Update(msg)
	return updated.(Model)
}

func stringPointer(value string) *string { return &value }
