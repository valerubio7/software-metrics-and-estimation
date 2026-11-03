package tui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDefectClientListIsReadOnlyAndPreservesResponse(t *testing.T) {
	const responseBody = `{"project_id":"project/1","defects":[{"id":"d-1","description":"Crash on save","severity":"high","status":"open","story_id":"story-1","sprint_id":"sprint-1","resolution_sprint_id":null}]}`
	var gotMethod, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseBody))
	}))
	defer server.Close()

	response, err := NewDefectClient(server.URL).List(context.Background(), "project/1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/projects/project%2F1/defects" {
		t.Fatalf("path = %q, want escaped project-scoped path", gotPath)
	}
	if string(response.Body) != responseBody {
		t.Fatalf("body = %q, want API body unchanged", response.Body)
	}
}

func TestDefectsProjectSelectionLoadsSelectedProject(t *testing.T) {
	var gotMethod, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"project_id":"p-2","defects":[]}`))
	}))
	defer server.Close()

	model := NewModel(server.URL)
	model, cmd := model.openDefects()
	if model.Screen != "project-picker" || model.pickerPurpose != "defects" || cmd == nil {
		t.Fatalf("openDefects() without active project = screen %q, purpose %q, cmd nil %v", model.Screen, model.pickerPurpose, cmd == nil)
	}
	model, cmd = model.selectProject(ProjectSummary{ID: "p-2", Name: "Project Two"})
	if model.Screen != "defects" || cmd == nil {
		t.Fatalf("project selection = screen %q, cmd nil %v", model.Screen, cmd == nil)
	}
	model = updateDefectTestModel(model, cmd())
	if gotMethod != http.MethodGet || gotPath != "/projects/p-2/defects" {
		t.Fatalf("selected project request = %s %s, want GET /projects/p-2/defects", gotMethod, gotPath)
	}
	if model.activeProjectID != "p-2" || len(model.defects) != 0 || model.defectLoading {
		t.Fatalf("selected project list state = project %q, defects %d, loading %v", model.activeProjectID, len(model.defects), model.defectLoading)
	}
}

func TestDefectsListAndRefreshAreReadOnly(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		_, _ = w.Write([]byte(`{"project_id":"p-1","defects":[{"id":"d-1","description":"Crash on save","severity":"high","status":"open","story_id":"story-1","sprint_id":"sprint-1","resolution_sprint_id":null},{"id":"d-2","description":"Wrong total","severity":"low","status":"resolved","story_id":"story-2","sprint_id":"sprint-2","resolution_sprint_id":"sprint-3"}]}`))
	}))
	defer server.Close()

	model := NewModel(server.URL)
	model.activeProjectID, model.activeProjectName = "p-1", "Project One"
	model, cmd := model.openDefects()
	if model.Screen != "defects" || cmd == nil {
		t.Fatalf("openDefects() = screen %q, cmd nil %v", model.Screen, cmd == nil)
	}
	if !model.defectLoading || !strings.Contains(model.View(), "Cargando defectos") {
		t.Fatal("project defect list did not show its loading state")
	}
	result := cmd().(DefectResultMsg)
	model = updateDefectTestModel(model, result)
	if model.defectLoading || len(model.defects) != 2 {
		t.Fatalf("list state loading=%v defects=%d", model.defectLoading, len(model.defects))
	}
	listView := model.View()
	for _, field := range []string{"Crash on save", "high", "open", "story-1", "sprint-1", "d-1", "sprint-3"} {
		if !strings.Contains(listView, field) {
			t.Errorf("list view does not contain %q", field)
		}
	}
	if strings.Contains(listView, "muestra el detalle") {
		t.Fatal("list-only slice advertises a deferred detail view")
	}
	updated, refresh := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	model = updated.(Model)
	if refresh == nil {
		t.Fatal("refresh command is nil")
	}
	model = updateDefectTestModel(model, refresh())
	if got := strings.Join(methods, ","); got != "GET,GET" {
		t.Fatalf("requests = %q, want GET only for list and refresh", got)
	}
}

func TestDefectSearchMatchesEveryFieldCaseInsensitively(t *testing.T) {
	resolutionSprintID := "sprint-fix"
	defect := Defect{
		ID: "DEFECT-42", Description: "Cache invalidation failed", Severity: "Critical",
		Status: "Open", StoryID: "STORY-19", SprintID: "SPRINT-DETECT",
		ResolutionSprintID: &resolutionSprintID,
	}
	tests := []struct {
		name  string
		query string
	}{
		{name: "defect ID", query: "defect-42"},
		{name: "description", query: "CACHE INVALIDATION"},
		{name: "severity", query: "critical"},
		{name: "status", query: "OPEN"},
		{name: "story ID", query: "story-19"},
		{name: "detection Sprint ID", query: "sprint-detect"},
		{name: "resolution Sprint ID", query: "sprint-fix"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewModel("http://example.invalid")
			model.Screen, model.defects = "defects", []Defect{defect}
			model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
			for _, r := range tt.query {
				model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			}
			if visible := model.visibleDefects(); len(visible) != 1 || visible[0].ID != defect.ID {
				t.Fatalf("search for %q returned %#v", tt.query, visible)
			}
			model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyEnter})
			if model.defectFiltering || model.defectFilter != tt.query {
				t.Fatalf("Enter did not preserve and finish search: filtering=%v filter=%q", model.defectFiltering, model.defectFilter)
			}
		})
	}
}

func TestDefectSearchKeepsGlobalShortcutsAsFilterText(t *testing.T) {
	model := NewModel("http://example.invalid")
	model.Screen, model.defects = "defects", []Defect{{ID: "d-1", Description: "Sample"}}
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	for _, key := range []string{"q", "p", "?"} {
		updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		model = updated.(Model)
		if cmd != nil || model.Screen != "defects" || !model.defectFiltering || model.showHelp {
			t.Fatalf("key %q escaped search: screen=%q filtering=%v help=%v cmd nil=%v", key, model.Screen, model.defectFiltering, model.showHelp, cmd == nil)
		}
	}
	if model.defectFilter != "qp?" {
		t.Fatalf("filter = %q, want qp?", model.defectFilter)
	}
}

func TestDefectSearchArrowSelectionAndEscapeClear(t *testing.T) {
	model := NewModel("http://example.invalid")
	model.Screen = "defects"
	model.defects = []Defect{
		{ID: "d-1", Description: "Match first"},
		{ID: "d-2", Description: "Match second"},
		{ID: "d-3", Description: "Other record"},
	}
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	for _, r := range "match" {
		model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyDown})
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyEnter})
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.Screen != "defect-detail" || model.defectDetail.ID != "d-2" {
		t.Fatalf("filtered arrow selection opened screen %q for defect %q", model.Screen, model.defectDetail.ID)
	}
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.Screen != "defects" || model.defectFilter != "match" {
		t.Fatalf("detail Escape did not preserve filter: screen=%q filter=%q", model.Screen, model.defectFilter)
	}
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.Screen != "defects" || model.defectFilter != "" || len(model.visibleDefects()) != 3 {
		t.Fatalf("list Escape did not clear filter: screen=%q filter=%q visible=%d", model.Screen, model.defectFilter, len(model.visibleDefects()))
	}
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	for _, r := range "match" {
		model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.Screen != "defects" || model.defectFiltering || model.defectFilter != "" {
		t.Fatalf("Escape while searching did not clear filter: screen=%q filtering=%v filter=%q", model.Screen, model.defectFiltering, model.defectFilter)
	}
}

func TestDefectDetailUsesSelectedListRecordAndReturnsToFilteredList(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		_, _ = w.Write([]byte(`{"project_id":"p-1","defects":[{"id":"d-alpha","description":"Alpha crash","severity":"high","status":"open","story_id":"story-a","sprint_id":"sprint-a","resolution_sprint_id":null},{"id":"d-beta","description":"Beta mismatch","severity":"low","status":"resolved","story_id":"story-b","sprint_id":"sprint-b","resolution_sprint_id":"sprint-fix"}]}`))
	}))
	defer server.Close()

	model := NewModel(server.URL)
	model.activeProjectID = "p-1"
	model, cmd := model.openDefects()
	model = updateDefectTestModel(model, cmd())
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	for _, r := range "alpha" {
		model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyEnter})
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.Screen != "defect-detail" || model.defectDetail.ID != "d-alpha" {
		t.Fatalf("detail = screen %q, defect %#v", model.Screen, model.defectDetail)
	}
	view := model.View()
	for _, field := range []string{"d-alpha", "Alpha crash", "high", "open", "story-a", "sprint-a", "Sprint de resolución", "—"} {
		if !strings.Contains(view, field) {
			t.Errorf("detail view does not contain %q", field)
		}
	}
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyEsc})
	if model.Screen != "defects" || model.defectFilter != "alpha" || model.defectFiltering {
		t.Fatalf("Escape did not restore filtered list: screen=%q filter=%q filtering=%v", model.Screen, model.defectFilter, model.defectFiltering)
	}
	if view := model.View(); !strings.Contains(view, "d-alpha") || strings.Contains(view, "d-beta") {
		t.Fatalf("filtered list context was not restored:\n%s", view)
	}
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyEsc})
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	for _, r := range "beta" {
		model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyEnter})
	model = updateDefectTestModel(model, tea.KeyMsg{Type: tea.KeyEnter})
	if model.Screen != "defect-detail" || !strings.Contains(model.View(), "sprint-fix") {
		t.Fatalf("non-null resolution Sprint missing from selected detail: screen=%q\n%s", model.Screen, model.View())
	}
	if got := strings.Join(methods, ","); got != "GET" {
		t.Fatalf("detail requests = %q, want GET only", got)
	}
}

func TestChangingProjectFromDefectDetailOpensNewProjectList(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"project_id":"new-project","defects":[]}`))
	}))
	defer server.Close()

	model := NewModel(server.URL)
	model.Screen, model.activeProjectID = "defect-detail", "old-project"
	model.defectDetail = Defect{ID: "old-defect", Description: "Old project record"}
	model.defectFilter = "old-project-filter"
	model, _ = model.openProjectPicker("project")
	model, cmd := model.selectProject(ProjectSummary{ID: "new-project", Name: "New project"})
	if model.Screen != "defects" || cmd == nil {
		t.Fatalf("selecting another project from detail = screen %q, cmd nil %v", model.Screen, cmd == nil)
	}
	model = updateDefectTestModel(model, cmd())
	if gotPath != "/projects/new-project/defects" || model.activeProjectID != "new-project" || model.defectFilter != "" {
		t.Fatalf("new project list request = %q for %q with filter %q", gotPath, model.activeProjectID, model.defectFilter)
	}
}

func TestDefectsIgnoreListResponseFromPreviousProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"project_id":"old-project","defects":[{"id":"old-defect","description":"Previous project defect","severity":"high","status":"open","story_id":"old-story","sprint_id":"old-sprint","resolution_sprint_id":null}]}`))
	}))
	defer server.Close()

	model := NewModel(server.URL)
	model.Screen, model.activeProjectID = "defects", "old-project"
	model, cmd := model.startDefectLoad()
	model.activeProjectID = "new-project"
	result := cmd().(DefectResultMsg)
	model = updateDefectTestModel(model, result)
	if len(model.defects) != 0 || strings.Contains(model.View(), "Previous project defect") {
		t.Fatalf("stale defect response populated the new project's list: %#v", model.defects)
	}
}

func updateDefectTestModel(model Model, msg tea.Msg) Model {
	updated, _ := model.Update(msg)
	return updated.(Model)
}

func TestDefectsListEmptyAndAPIErrorStates(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantScreen string
		wantText   string
	}{
		{name: "empty list", status: http.StatusOK, body: `{"project_id":"p-1","defects":[]}`, wantScreen: "defects", wantText: "No hay defectos registrados."},
		{name: "api error", status: http.StatusUnprocessableEntity, body: `{"error":"validation_error","message":"project unavailable"}`, wantScreen: "defects", wantText: "unavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewModel("http://example.invalid")
			model.Screen, model.activeProjectID = "defects", "p-1"
			model.defectLoading = true
			model = updateDefectTestModel(model, DefectResultMsg{ProjectID: "p-1", Response: APIResponse{StatusCode: tt.status, Body: []byte(tt.body)}})
			if model.Screen != tt.wantScreen {
				t.Fatalf("screen = %q, want %q", model.Screen, tt.wantScreen)
			}
			if view := model.View(); !strings.Contains(view, tt.wantText) {
				t.Fatalf("view does not contain %q:\n%s", tt.wantText, view)
			}
		})
	}
}

func TestDefectListResponseDecodesAllReferenceFields(t *testing.T) {
	var listing struct {
		ProjectID string   `json:"project_id"`
		Defects   []Defect `json:"defects"`
	}
	if err := json.Unmarshal([]byte(`{"project_id":"p-1","defects":[{"id":"d-1","description":"Crash","severity":"high","status":"open","story_id":"story-1","sprint_id":"sprint-1","resolution_sprint_id":null}]}`), &listing); err != nil {
		t.Fatal(err)
	}
	if listing.ProjectID != "p-1" || len(listing.Defects) != 1 || listing.Defects[0].ResolutionSprintID != nil {
		t.Fatalf("decoded listing = %#v", listing)
	}
}
