package tui_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

// backlogActionCount is the number of actions that precede the stories on the
// backlog screen when a project is active: Crear historia, Cambiar proyecto,
// Refrescar, Volver.
const backlogActionCount = 4

func intPtr(v int) *int { return &v }

// sampleStories returns a backlog whose API order differs from alphabetical order.
func sampleStories() []tui.Story {
	return []tui.Story{
		{ID: "s1", ProjectID: "p1", Title: "Login", Description: "Pantalla de acceso", Priority: "alta", Status: "pendiente",
			AcceptanceCriteria: []string{"A", "B"}, EstimatedHours: floatPtr(5)},
		{ID: "s2", ProjectID: "p1", Title: "Alta de usuarios", Description: "Formulario", Priority: "media", Status: "en curso",
			StoryPoints: intPtr(3), AcceptanceCriteria: []string{}},
	}
}

// sampleProjects is the collection served by GET /projects for the picker.
func sampleProjects() []tui.ProjectSummary {
	return []tui.ProjectSummary{
		{ID: "p1", Name: "Alpha", StartDate: "2026-01-01", PlannedFinishDate: "2026-12-31", Status: "active"},
		{ID: "p2", Name: "Beta", StartDate: "2026-01-01", PlannedFinishDate: "2026-12-31", Status: "planned"},
	}
}

func projectsBody(t *testing.T, projects []tui.ProjectSummary) string {
	t.Helper()
	if projects == nil {
		projects = []tui.ProjectSummary{}
	}
	encoded, err := json.Marshal(struct {
		Projects []tui.ProjectSummary `json:"projects"`
	}{projects})
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

type stub struct {
	status int
	body   string
}

type apiStubs struct {
	projects, list, create, update stub
}

func storiesBody(t *testing.T, stories []tui.Story) string {
	t.Helper()
	if stories == nil {
		stories = []tui.Story{}
	}
	encoded, err := json.Marshal(struct {
		ProjectID string      `json:"project_id"`
		Stories   []tui.Story `json:"stories"`
	}{"p1", stories})
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func storyBody(t *testing.T, story tui.Story) string {
	t.Helper()
	encoded, err := json.Marshal(story)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func defaultStub(s stub, status int, body string) stub {
	if s.status == 0 {
		return stub{status, body}
	}
	return s
}

// fakeBacklogAPI serves the project collection and the story endpoints, answering
// each with its stub (list defaults to sampleStories) and recording every request.
func fakeBacklogAPI(t *testing.T, stubs apiStubs) (*httptest.Server, *recorder) {
	t.Helper()
	rec := &recorder{}
	stubs.projects = defaultStub(stubs.projects, http.StatusOK, projectsBody(t, sampleProjects()))
	stubs.list = defaultStub(stubs.list, http.StatusOK, storiesBody(t, sampleStories()))
	stubs.create = defaultStub(stubs.create, http.StatusCreated, storyBody(t, tui.Story{ID: "s9", Title: "Historia nueva"}))
	stubs.update = defaultStub(stubs.update, http.StatusOK, storyBody(t, tui.Story{ID: "s1", Title: "Historia modificada"}))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		rec.mu.Lock()
		rec.requests = append(rec.requests, recordedRequest{Method: r.Method, Path: r.URL.EscapedPath(), Body: raw})
		rec.mu.Unlock()
		if r.Method == http.MethodGet && r.URL.Path == "/projects" {
			w.WriteHeader(stubs.projects.status)
			_, _ = w.Write([]byte(stubs.projects.body))
			return
		}
		response := map[string]stub{http.MethodGet: stubs.list, http.MethodPost: stubs.create, http.MethodPut: stubs.update}[r.Method]
		w.WriteHeader(response.status)
		_, _ = w.Write([]byte(response.body))
	}))
	t.Cleanup(server.Close)
	return server, rec
}

func countMethod(rec *recorder, method string) int {
	count := 0
	for _, request := range rec.all() {
		if request.Method == method {
			count++
		}
	}
	return count
}

func lastRequest(t *testing.T, rec *recorder) recordedRequest {
	t.Helper()
	requests := rec.all()
	if len(requests) == 0 {
		t.Fatal("no request was recorded")
	}
	return requests[len(requests)-1]
}

// apply feeds msg to the model.
func apply(m tui.Model, msg tea.Msg) (tui.Model, tea.Cmd) {
	updated, cmd := m.Update(msg)
	return updated.(tui.Model), cmd
}

// run executes cmd (a synchronous tea.Cmd) and feeds the resulting message to the model.
func run(t *testing.T, m tui.Model, cmd tea.Cmd) tui.Model {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a command to execute, got nil")
	}
	m, _ = apply(m, cmd())
	return m
}

// startBacklog opens the backlog from the menu (Down, Enter). With no active
// project this lands on the project picker and returns its load command.
func startBacklog(url string) (tui.Model, tea.Cmd) {
	m := tui.NewModel(url)
	m = press(m, tea.KeyDown)
	return apply(m, tea.KeyMsg{Type: tea.KeyEnter})
}

// confirmProjectInPicker runs the picker's load command, searches by pid and
// confirms the first match, returning the model and the command of the next step.
func confirmProjectInPicker(t *testing.T, m tui.Model, load tea.Cmd, pid string) (tui.Model, tea.Cmd) {
	t.Helper()
	m = run(t, m, load)
	m, _ = apply(m, keyMsg("/"))
	m = typeText(m, pid)
	m = press(m, tea.KeyEnter) // stop searching, keep the filter
	return apply(m, tea.KeyMsg{Type: tea.KeyEnter})
}

// loadingBacklog reaches the backlog screen waiting for the list response of pid.
func loadingBacklog(t *testing.T, url, pid string) tui.Model {
	t.Helper()
	m, load := startBacklog(url)
	m, cmd := confirmProjectInPicker(t, m, load, pid)
	if cmd == nil {
		t.Fatal("confirming the project should issue the list command")
	}
	return m
}

// openBacklogWithProject goes menu -> Backlog -> picker -> pid -> Enter and applies the list response.
func openBacklogWithProject(t *testing.T, url, pid string) tui.Model {
	t.Helper()
	m, load := startBacklog(url)
	m, cmd := confirmProjectInPicker(t, m, load, pid)
	m = run(t, m, cmd)
	if m.Screen != "backlog" {
		t.Fatalf("Screen = %q after loading, want backlog", m.Screen)
	}
	return m
}

// submitStory opens the create form from the backlog (cursor 0), fills it and presses Enter on the last field.
func submitStory(m tui.Model, title, description, priority, criteria string) (tui.Model, tea.Cmd) {
	m = press(m, tea.KeyEnter)
	for i, value := range []string{title, description, priority, criteria} {
		m = typeText(m, value)
		if i < 3 {
			m = press(m, tea.KeyEnter)
		}
	}
	return apply(m, tea.KeyMsg{Type: tea.KeyEnter})
}

// selectStory moves to the index-th story of the list and opens its update form.
func selectStory(m tui.Model, index int) tui.Model {
	for i := 0; i < backlogActionCount+index; i++ {
		m = press(m, tea.KeyDown)
	}
	return press(m, tea.KeyEnter)
}

// advanceFields moves to the next form field n times.
func advanceFields(m tui.Model, n int) tui.Model {
	for i := 0; i < n; i++ {
		m = press(m, tea.KeyTab)
	}
	return m
}

// submitUpdate presses Enter on the focused (last) field of the update form.
func submitUpdate(m tui.Model) (tui.Model, tea.Cmd) {
	return apply(m, tea.KeyMsg{Type: tea.KeyEnter})
}

func failure(status int, body string) tui.BacklogResultMsg {
	return tui.BacklogResultMsg{Op: tui.BacklogOpCreate, Response: tui.APIResponse{StatusCode: status, Body: []byte(body)}}
}

func TestBacklogMenuOpensPickerWithoutProject(t *testing.T) {
	m, _ := startBacklog("http://localhost:8080")
	if m.Screen != "project-picker" {
		t.Fatalf("Screen = %q, want project-picker", m.Screen)
	}
	if !strings.Contains(m.View(), "Elegí un proyecto") {
		t.Errorf("View() = %q, want the project picker", m.View())
	}
	if strings.Contains(m.View(), "TUI-03") {
		t.Errorf("View() = %q, want the TUI-03 placeholder gone", m.View())
	}
}

func TestBacklogPickerFiltersAndConfirmsProject(t *testing.T) {
	server, _ := fakeBacklogAPI(t, apiStubs{})
	m, load := startBacklog(server.URL)
	m = run(t, m, load)
	m, _ = apply(m, keyMsg("/"))
	m = typeText(m, "Beta")
	view := m.View()
	if !strings.Contains(view, "Beta") || strings.Contains(view, "Alpha") {
		t.Errorf("View() = %q, want only the matching project", view)
	}
	m = press(m, tea.KeyEsc) // clear the search, stay in the browser
	if m.Screen != "project-picker" {
		t.Errorf("Screen = %q after clearing the search, want project-picker", m.Screen)
	}
	m = press(m, tea.KeyEsc) // leave the browser
	if m.Screen != "menu" {
		t.Errorf("Screen = %q after Esc, want menu", m.Screen)
	}
}

func TestBacklogProjectSelectionLoadsBacklogInAPIOrder(t *testing.T) {
	stories := []tui.Story{
		{ID: "s1", Title: "Zeta historia", Priority: "alta", Status: "pendiente"},
		{ID: "s2", Title: "Alfa historia", Priority: "media", Status: "en curso", StoryPoints: intPtr(3)},
		{ID: "s3", Title: "Media historia", Priority: "baja", Status: "hecha", StoryPoints: intPtr(0)},
	}
	server, rec := fakeBacklogAPI(t, apiStubs{list: stub{http.StatusOK, storiesBody(t, stories)}})

	m, load := startBacklog(server.URL)
	m, cmd := confirmProjectInPicker(t, m, load, "p1")
	if m.Screen != "backlog" {
		t.Fatalf("Screen = %q, want backlog", m.Screen)
	}
	if !strings.Contains(m.View(), "Cargando backlog…") {
		t.Errorf("View() = %q, want the loading message", m.View())
	}
	m = run(t, m, cmd)
	if got := lastRequest(t, rec); got.Method != http.MethodGet || got.Path != "/projects/p1/stories" {
		t.Errorf("request = %s %s, want GET /projects/p1/stories", got.Method, got.Path)
	}
	view := m.View()
	if strings.Contains(view, "Cargando backlog…") {
		t.Errorf("View() = %q, want the loading message gone", view)
	}
	previous := -1
	for _, title := range []string{"Zeta historia", "Alfa historia", "Media historia"} {
		index := strings.Index(view, title)
		if index < 0 {
			t.Fatalf("View() = %q, want the story %q", view, title)
		}
		if index < previous {
			t.Errorf("story %q appears before the previous one, want the API order", title)
		}
		previous = index
	}
	if !strings.Contains(view, "Backlog del proyecto Alpha") {
		t.Errorf("View() = %q, want the project header", view)
	}
	if !strings.Contains(view, "puntos 3") {
		t.Errorf("View() = %q, want the read-only story points", view)
	}
}

func TestBacklogProjectIsKeptAcrossMenuReentry(t *testing.T) {
	server, rec := fakeBacklogAPI(t, apiStubs{})
	m := openBacklogWithProject(t, server.URL, "p1")
	m = press(m, tea.KeyEsc)
	if m.Screen != "menu" {
		t.Fatalf("Screen = %q after Esc, want menu", m.Screen)
	}
	m = press(m, tea.KeyDown)
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != "backlog" {
		t.Fatalf("Screen = %q, want backlog", m.Screen)
	}
	if strings.Contains(m.View(), "Elegí un proyecto") {
		t.Errorf("View() = %q, want no picker when the project is kept", m.View())
	}
	m = run(t, m, cmd)
	if got := lastRequest(t, rec); got.Method != http.MethodGet || got.Path != "/projects/p1/stories" {
		t.Errorf("request = %s %s, want GET of the kept project", got.Method, got.Path)
	}
	if !strings.Contains(m.View(), "Login") {
		t.Errorf("View() = %q, want the reloaded stories", m.View())
	}
}

func TestBacklogChangeProjectUsesPicker(t *testing.T) {
	server, rec := fakeBacklogAPI(t, apiStubs{})
	m := openBacklogWithProject(t, server.URL, "p1")

	m = press(m, tea.KeyDown) // Cambiar proyecto
	m, load := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != "project-picker" {
		t.Fatalf("Screen = %q, want project-picker", m.Screen)
	}
	m = press(m, tea.KeyEsc)
	if m.Screen != "backlog" || !strings.Contains(m.View(), "Backlog del proyecto Alpha") {
		t.Errorf("after Esc: Screen = %q view = %q, want backlog still on Alpha", m.Screen, m.View())
	}

	m, load = apply(m, tea.KeyMsg{Type: tea.KeyEnter}) // Cambiar proyecto again
	m, cmd := confirmProjectInPicker(t, m, load, "p2")
	m = run(t, m, cmd)
	if got := lastRequest(t, rec); got.Method != http.MethodGet || got.Path != "/projects/p2/stories" {
		t.Errorf("request = %s %s, want GET /projects/p2/stories", got.Method, got.Path)
	}
	if !strings.Contains(m.View(), "Backlog del proyecto Beta") {
		t.Errorf("View() = %q, want the new project", m.View())
	}

	_, cmd = submitStory(m, "T", "D", "alta", "")
	if cmd == nil {
		t.Fatal("create should issue a command")
	}
	cmd()
	if got := lastRequest(t, rec); got.Method != http.MethodPost || got.Path != "/projects/p2/stories" {
		t.Errorf("request = %s %s, want POST /projects/p2/stories", got.Method, got.Path)
	}
}

func TestBacklogEmptyListShowsMessageAndKeepsActions(t *testing.T) {
	server, _ := fakeBacklogAPI(t, apiStubs{list: stub{http.StatusOK, storiesBody(t, nil)}})
	m := openBacklogWithProject(t, server.URL, "p1")
	view := m.View()
	for _, want := range []string{"El backlog no tiene historias.", "Crear historia", "Cambiar proyecto"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() = %q, want %q", view, want)
		}
	}
}

func TestBacklogSearchFiltersStories(t *testing.T) {
	server, _ := fakeBacklogAPI(t, apiStubs{})
	m := openBacklogWithProject(t, server.URL, "p1")

	m, _ = apply(m, keyMsg("/"))
	if !strings.Contains(m.View(), "Buscar: _") {
		t.Fatalf("View() = %q, want the search prompt", m.View())
	}
	// q is a normal character while the search box owns the keyboard.
	m, cmd := apply(m, keyMsg("q"))
	if isQuit(cmd) {
		t.Fatal("q must be typed into the search box, not quit")
	}
	m = press(m, tea.KeyBackspace)
	m = typeText(m, "Alta")
	view := m.View()
	if !strings.Contains(view, "Alta de usuarios") || strings.Contains(view, "Login") {
		t.Fatalf("View() = %q, want only the matching story", view)
	}
	m, _ = apply(m, keyMsg("zz"))
	view = m.View()
	if !strings.Contains(view, "Sin coincidencias.") || strings.Contains(view, "Alta de usuarios") {
		t.Fatalf("View() = %q, want the no-match state", view)
	}
	for range 2 {
		m = press(m, tea.KeyBackspace)
	}
	m = press(m, tea.KeyEnter)

	// The filtered story is now the first selectable story, after the actions.
	for i := 0; i < backlogActionCount; i++ {
		m = press(m, tea.KeyDown)
	}
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != "backlog-form" || fieldValue(t, m.View(), "Título") != "Alta de usuarios" {
		t.Fatalf("Screen = %q view = %q, want the filtered story form", m.Screen, m.View())
	}

	m = press(m, tea.KeyEsc) // back to the filtered backlog
	m, _ = apply(m, keyMsg("/"))
	m = press(m, tea.KeyEsc) // clear the search
	if strings.Contains(m.View(), "Buscar:") {
		t.Errorf("View() = %q, want the search box removed after Esc", m.View())
	}
}

func TestBacklogCreateStorySendsFourKeysAndShowsSuccess(t *testing.T) {
	created := tui.Story{ID: "s9", Title: "Respuesta de la API", Priority: "alta", Status: "pendiente"}
	server, rec := fakeBacklogAPI(t, apiStubs{create: stub{http.StatusCreated, storyBody(t, created)}})
	m := openBacklogWithProject(t, server.URL, "p1")

	m = press(m, tea.KeyEnter) // Crear historia
	if m.Screen != "backlog-form" {
		t.Fatalf("Screen = %q, want backlog-form", m.Screen)
	}
	m = typeText(m, "Como usuario quiero ingresar")
	if got := fieldValue(t, m.View(), "Título"); got != "Como usuario quiero ingresar" {
		t.Errorf("title field = %q, want the spaces kept", got)
	}
	m = press(m, tea.KeyEnter)
	m = typeText(m, "Para acceder")
	m = press(m, tea.KeyEnter)
	m = typeText(m, "alta")
	m = press(m, tea.KeyEnter)
	m = typeText(m, "Inicia sesión")
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(t, m, cmd)

	got := lastRequest(t, rec)
	if got.Method != http.MethodPost || got.Path != "/projects/p1/stories" {
		t.Errorf("request = %s %s, want POST /projects/p1/stories", got.Method, got.Path)
	}
	keys := bodyKeys(t, got.Body)
	if want := []string{"acceptance_criteria", "description", "priority", "title"}; !equalStrings(sortedKeys(keys), want) {
		t.Errorf("body keys = %v, want %v", sortedKeys(keys), want)
	}
	var title string
	_ = json.Unmarshal(keys["title"], &title)
	if title != "Como usuario quiero ingresar" {
		t.Errorf("title = %q, want the full title with spaces", title)
	}
	if m.Screen != "backlog-result" {
		t.Fatalf("Screen = %q, want backlog-result", m.Screen)
	}
	view := m.View()
	for _, want := range []string{"201", "creada", "Respuesta de la API"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() = %q, want %q", view, want)
		}
	}
	if strings.Contains(view, "Error") {
		t.Errorf("View() = %q, want no error text on success", view)
	}
}

func TestBacklogCreateSplitsCriteriaBySemicolon(t *testing.T) {
	cases := []struct {
		name, typed string
		want        []string
	}{
		{"trimmed-and-filtered", "  Inicia sesión ; Cierra sesión ;; ", []string{"Inicia sesión", "Cierra sesión"}},
		{"empty", "", []string{}},
		{"only-separators", " ; ;; ", []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server, rec := fakeBacklogAPI(t, apiStubs{})
			m := openBacklogWithProject(t, server.URL, "p1")
			m, cmd := submitStory(m, "T", "D", "alta", tc.typed)
			run(t, m, cmd)
			keys := bodyKeys(t, lastRequest(t, rec).Body)
			var criteria []string
			if err := json.Unmarshal(keys["acceptance_criteria"], &criteria); err != nil {
				t.Fatalf("acceptance_criteria = %s: %v", keys["acceptance_criteria"], err)
			}
			if !equalStrings(criteria, tc.want) || (len(tc.want) == 0 && string(keys["acceptance_criteria"]) != "[]") {
				t.Errorf("acceptance_criteria = %s, want %q", keys["acceptance_criteria"], tc.want)
			}
		})
	}
}

func TestBacklogCreateDoesNotValidatePriorityLocally(t *testing.T) {
	for _, priority := range []string{"urgentísima", ""} {
		t.Run("priority="+priority, func(t *testing.T) {
			server, rec := fakeBacklogAPI(t, apiStubs{})
			m := openBacklogWithProject(t, server.URL, "p1")
			m, cmd := submitStory(m, "T", "D", priority, "")
			run(t, m, cmd)
			keys := bodyKeys(t, lastRequest(t, rec).Body)
			var sent string
			_ = json.Unmarshal(keys["priority"], &sent)
			if sent != priority {
				t.Errorf("priority = %q, want %q sent as typed", sent, priority)
			}
		})
	}
}

func TestBacklogUpdateFormIsPrefilledFromSelectedStory(t *testing.T) {
	cases := []struct {
		name   string
		index  int
		fields map[string]string
	}{
		{"first-story", 0, map[string]string{
			"Título": "Login", "Descripción": "Pantalla de acceso", "Prioridad": "alta", "Estado": "pendiente",
			"Criterios de aceptación": "A; B", "Horas estimadas": "5",
		}},
		{"second-story", 1, map[string]string{
			"Título": "Alta de usuarios", "Descripción": "Formulario", "Prioridad": "media", "Estado": "en curso",
			"Criterios de aceptación": "", "Horas estimadas": "",
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server, _ := fakeBacklogAPI(t, apiStubs{})
			m := openBacklogWithProject(t, server.URL, "p1")
			if tc.index == 1 && !strings.Contains(m.View(), "puntos 3") {
				t.Errorf("list View() = %q, want the read-only story points 3", m.View())
			}
			m = selectStory(m, tc.index)
			if m.Screen != "backlog-form" {
				t.Fatalf("Screen = %q, want backlog-form", m.Screen)
			}
			view := m.View()
			for label, want := range tc.fields {
				if got := fieldValue(t, view, label); got != want {
					t.Errorf("field %q = %q, want %q", label, got, want)
				}
			}
			lower := strings.ToLower(view)
			if strings.Contains(lower, "story_points") {
				t.Errorf("View() = %q, want no editable story points field", view)
			}
		})
	}
}

func TestBacklogUpdateWithoutChangesSendsSixKeys(t *testing.T) {
	cases := []struct {
		name      string
		index     int
		path      string
		wantTitle string
		wantHours string
		wantCrit  string
	}{
		{"first-story", 0, "/projects/p1/stories/s1", "Login", "5", `["A","B"]`},
		{"second-story", 1, "/projects/p1/stories/s2", "Alta de usuarios", "null", `[]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server, rec := fakeBacklogAPI(t, apiStubs{})
			m := selectStory(openBacklogWithProject(t, server.URL, "p1"), tc.index)
			m = advanceFields(m, 5)
			m, cmd := submitUpdate(m)
			m = run(t, m, cmd)

			got := lastRequest(t, rec)
			if got.Method != http.MethodPut || got.Path != tc.path {
				t.Errorf("request = %s %s, want PUT %s", got.Method, got.Path, tc.path)
			}
			keys := bodyKeys(t, got.Body)
			want := []string{"acceptance_criteria", "description", "estimated_hours", "priority", "status", "title"}
			if !equalStrings(sortedKeys(keys), want) {
				t.Errorf("body keys = %v, want %v", sortedKeys(keys), want)
			}
			if string(keys["title"]) != `"`+tc.wantTitle+`"` || string(keys["estimated_hours"]) != tc.wantHours || string(keys["acceptance_criteria"]) != tc.wantCrit {
				t.Errorf("body = %s, want title %q hours %s criteria %s", got.Body, tc.wantTitle, tc.wantHours, tc.wantCrit)
			}
			if m.Screen != "backlog-result" || !strings.Contains(m.View(), "modificada") {
				t.Errorf("Screen = %q view = %q, want the update success result", m.Screen, m.View())
			}
		})
	}
}

func TestBacklogUpdateEmptyHoursSendsNull(t *testing.T) {
	server, rec := fakeBacklogAPI(t, apiStubs{})
	m := selectStory(openBacklogWithProject(t, server.URL, "p1"), 0)
	m = advanceFields(m, 5)
	m = press(m, tea.KeyBackspace)
	if got := fieldValue(t, m.View(), "Horas estimadas"); got != "" {
		t.Fatalf("hours field = %q, want it emptied", got)
	}
	m, cmd := submitUpdate(m)
	run(t, m, cmd)
	keys := bodyKeys(t, lastRequest(t, rec).Body)
	raw, present := keys["estimated_hours"]
	if !present || string(raw) != "null" {
		t.Errorf("estimated_hours = %q (present=%v), want the explicit key with null", raw, present)
	}
}

func TestBacklogUpdateChangingTitleKeepsRest(t *testing.T) {
	server, rec := fakeBacklogAPI(t, apiStubs{})
	m := selectStory(openBacklogWithProject(t, server.URL, "p1"), 0)
	for range "Login" {
		m = press(m, tea.KeyBackspace)
	}
	m = typeText(m, "Acceso web")
	m = advanceFields(m, 5)
	m, cmd := submitUpdate(m)
	run(t, m, cmd)

	var sent tui.StoryUpdateInput
	if err := json.Unmarshal(lastRequest(t, rec).Body, &sent); err != nil {
		t.Fatal(err)
	}
	if sent.Title != "Acceso web" {
		t.Errorf("title = %q, want Acceso web", sent.Title)
	}
	if sent.Description != "Pantalla de acceso" || sent.Priority != "alta" || sent.Status != "pendiente" ||
		!equalStrings(sent.AcceptanceCriteria, []string{"A", "B"}) || sent.EstimatedHours == nil || *sent.EstimatedHours != 5 {
		t.Errorf("body = %+v, want the remaining values unchanged", sent)
	}
}

func TestBacklogUpdateHoursNumericParsing(t *testing.T) {
	for _, tc := range []struct{ typed, want string }{{"8", "8"}, {"2.5", "2.5"}, {"-3", "-3"}} {
		t.Run(tc.typed, func(t *testing.T) {
			server, rec := fakeBacklogAPI(t, apiStubs{})
			m := selectStory(openBacklogWithProject(t, server.URL, "p1"), 0)
			m = advanceFields(m, 5)
			m = press(m, tea.KeyBackspace)
			m = typeText(m, tc.typed)
			m, cmd := submitUpdate(m)
			run(t, m, cmd)
			keys := bodyKeys(t, lastRequest(t, rec).Body)
			if got := string(keys["estimated_hours"]); got != tc.want {
				t.Errorf("estimated_hours = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestBacklogNonNumericHoursShowsErrorWithoutRequest(t *testing.T) {
	for _, typed := range []string{"abc", "NaN", "Inf"} {
		t.Run(typed, func(t *testing.T) {
			server, rec := fakeBacklogAPI(t, apiStubs{})
			m := selectStory(openBacklogWithProject(t, server.URL, "p1"), 0)
			m = advanceFields(m, 5)
			m = press(m, tea.KeyBackspace)
			m = typeText(m, typed)
			before := len(rec.all())
			m, cmd := submitUpdate(m)
			if cmd != nil {
				cmd()
			}
			if len(rec.all()) != before {
				t.Errorf("requests = %+v, want none after a non-numeric value", rec.all()[before:])
			}
			if m.Screen != "backlog-form" {
				t.Errorf("Screen = %q, want to stay in backlog-form", m.Screen)
			}
			view := m.View()
			if !strings.Contains(view, "Horas estimadas debe ser un número") {
				t.Errorf("View() = %q, want the numeric error", view)
			}
			if got := fieldValue(t, view, "Horas estimadas"); got != typed {
				t.Errorf("hours field = %q, want %q kept", got, typed)
			}
			if got := fieldValue(t, view, "Título"); got != "Login" {
				t.Errorf("title field = %q, want Login kept", got)
			}
		})
	}
}

func TestBacklogSelectingStoryWithEmptyListOpensNoForm(t *testing.T) {
	for downs := 0; downs < 6; downs++ {
		t.Run(string(rune('0'+downs))+"-downs", func(t *testing.T) {
			server, _ := fakeBacklogAPI(t, apiStubs{list: stub{http.StatusOK, storiesBody(t, nil)}})
			m := openBacklogWithProject(t, server.URL, "p1")
			for i := 0; i < downs; i++ {
				m = press(m, tea.KeyDown)
			}
			m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
			if cmd != nil {
				cmd()
			}
			if strings.Contains(m.View(), "Modificar historia") {
				t.Errorf("View() = %q, want no update form for an empty list", m.View())
			}
		})
	}
}

func TestBacklogFormEscReturnsToBacklogWithoutRequest(t *testing.T) {
	server, rec := fakeBacklogAPI(t, apiStubs{})
	forms := map[string]func(tui.Model) tui.Model{
		"create": func(m tui.Model) tui.Model { return press(m, tea.KeyEnter) },
		"update": func(m tui.Model) tui.Model { return selectStory(m, 0) },
	}
	for name, open := range forms {
		t.Run(name, func(t *testing.T) {
			m := open(openBacklogWithProject(t, server.URL, "p1"))
			if m.Screen != "backlog-form" {
				t.Fatalf("Screen = %q, want backlog-form", m.Screen)
			}
			before := len(rec.all())
			m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEsc})
			if cmd != nil {
				cmd()
			}
			if m.Screen != "backlog" {
				t.Errorf("Screen = %q after Esc, want backlog", m.Screen)
			}
			if len(rec.all()) != before {
				t.Errorf("requests = %+v, want none", rec.all()[before:])
			}
		})
	}
}

func TestBacklogRendersValidationErrorsAsList(t *testing.T) {
	server, _ := fakeBacklogAPI(t, apiStubs{})
	m := openBacklogWithProject(t, server.URL, "p1")
	m, _ = apply(m, failure(http.StatusUnprocessableEntity,
		`{"error":"validation_failed","message":"datos inválidos","fields":{"title":"es obligatorio","priority":"valor inválido"}}`))
	view := m.View()
	if m.Screen != "backlog-result" {
		t.Errorf("Screen = %q, want backlog-result", m.Screen)
	}
	if !strings.Contains(view, "La API rechazó los datos (422)") {
		t.Errorf("View() = %q, want the 422 heading", view)
	}
	first := strings.Index(view, "- priority: valor inválido")
	second := strings.Index(view, "- title: es obligatorio")
	if first < 0 || second < 0 || first > second {
		t.Errorf("View() = %q, want one list line per field ordered by key", view)
	}
	for _, unwanted := range []string{"creada", "modificada", "correctamente"} {
		if strings.Contains(view, unwanted) {
			t.Errorf("View() = %q, must not communicate success (%q)", view, unwanted)
		}
	}
}

func TestBacklogRendersNotFoundWithGenericMessageAndAPIDetail(t *testing.T) {
	cases := []struct {
		name string
		body string
		want []string
	}{
		{"story", `{"error":"story_not_found","message":"story not found"}`, []string{"no existe (404)", "story_not_found", "story not found"}},
		{"project", `{"error":"project_not_found","message":"project not found"}`, []string{"no existe (404)", "project_not_found", "project not found"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server, _ := fakeBacklogAPI(t, apiStubs{})
			m := openBacklogWithProject(t, server.URL, "p1")
			m, _ = apply(m, tui.BacklogResultMsg{Op: tui.BacklogOpUpdate, Response: tui.APIResponse{StatusCode: 404, Body: []byte(tc.body)}})
			view := m.View()
			for _, want := range tc.want {
				if !strings.Contains(view, want) {
					t.Errorf("View() = %q, want %q", view, want)
				}
			}
		})
	}
}

func TestBacklogRendersNonJSONErrorWithStatusCode(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   []string
	}{
		{"404-plain", 404, "404 page not found\n", []string{"Error HTTP 404", "404 page not found"}},
		{"405-plain", 405, "Method Not Allowed\n", []string{"Error HTTP 405", "Method Not Allowed"}},
		{"405-empty", 405, "", []string{"Error HTTP 405", "sin detalle"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server, _ := fakeBacklogAPI(t, apiStubs{})
			m := openBacklogWithProject(t, server.URL, "p1")
			m, cmd := apply(m, failure(tc.status, tc.body))
			if isQuit(cmd) {
				t.Error("a non-JSON error must not quit the TUI")
			}
			view := m.View()
			for _, want := range tc.want {
				if !strings.Contains(view, want) {
					t.Errorf("View() = %q, want %q", view, want)
				}
			}
		})
	}
}

func TestBacklogListFailureReturnsToPicker(t *testing.T) {
	for _, key := range []tea.KeyType{tea.KeyEnter, tea.KeyEsc} {
		t.Run(key.String(), func(t *testing.T) {
			server, _ := fakeBacklogAPI(t, apiStubs{list: stub{http.StatusUnprocessableEntity,
				`{"error":"validation_failed","message":"identificador inválido","fields":{"project_id":"debe ser un UUID"}}`}})
			m := loadingBacklog(t, server.URL, "p1")
			m, _ = apply(m, tui.BacklogResultMsg{Op: tui.BacklogOpList, Response: tui.APIResponse{
				StatusCode: 422, Body: []byte(`{"error":"validation_failed","message":"identificador inválido","fields":{"project_id":"debe ser un UUID"}}`),
			}})
			view := m.View()
			if m.Screen != "backlog-result" || !strings.Contains(view, "- project_id: debe ser un UUID") {
				t.Fatalf("Screen = %q view = %q, want the 422 fields list", m.Screen, view)
			}
			m, cmd := apply(m, tea.KeyMsg{Type: key})
			if m.Screen != "project-picker" {
				t.Fatalf("Screen = %q after %s, want project-picker", m.Screen, key)
			}
			if cmd == nil {
				t.Fatal("returning to the picker should reload the project collection")
			}
		})
	}
}

func TestBacklogTransportErrorDoesNotQuit(t *testing.T) {
	server, _ := fakeBacklogAPI(t, apiStubs{})
	ops := []tui.BacklogOp{tui.BacklogOpList, tui.BacklogOpCreate, tui.BacklogOpUpdate}
	for _, op := range ops {
		t.Run(string(op), func(t *testing.T) {
			m := loadingBacklog(t, server.URL, "p1")
			m, cmd := apply(m, tui.BacklogResultMsg{Op: op, Err: errors.New("connection refused")})
			if isQuit(cmd) {
				t.Error("a transport error must not quit the TUI")
			}
			view := m.View()
			for _, want := range []string{"Error de conexión", "connection refused"} {
				if !strings.Contains(view, want) {
					t.Errorf("View() = %q, want %q", view, want)
				}
			}
		})
	}
}

func TestBacklogResultReturnDestinations(t *testing.T) {
	server, rec := fakeBacklogAPI(t, apiStubs{})

	t.Run("create-success-reloads-backlog", func(t *testing.T) {
		m := openBacklogWithProject(t, server.URL, "p1")
		m, cmd := submitStory(m, "T", "D", "alta", "")
		m = run(t, m, cmd)
		if m.Screen != "backlog-result" {
			t.Fatalf("Screen = %q, want backlog-result", m.Screen)
		}
		before := countMethod(rec, http.MethodGet)
		m, cmd = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
		if m.Screen != "backlog" {
			t.Errorf("Screen = %q, want backlog", m.Screen)
		}
		m = run(t, m, cmd)
		if got := countMethod(rec, http.MethodGet); got != before+1 {
			t.Errorf("GET count = %d, want %d (reload)", got, before+1)
		}
		if !strings.Contains(m.View(), "Login") {
			t.Errorf("View() = %q, want the reloaded list", m.View())
		}
	})

	t.Run("update-success-reloads-backlog", func(t *testing.T) {
		m := selectStory(openBacklogWithProject(t, server.URL, "p1"), 0)
		m = advanceFields(m, 5)
		m, cmd := submitUpdate(m)
		m = run(t, m, cmd)
		before := countMethod(rec, http.MethodGet)
		m, cmd = apply(m, tea.KeyMsg{Type: tea.KeyEsc})
		if m.Screen != "backlog" {
			t.Errorf("Screen = %q, want backlog", m.Screen)
		}
		run(t, m, cmd)
		if got := countMethod(rec, http.MethodGet); got != before+1 {
			t.Errorf("GET count = %d, want %d (reload)", got, before+1)
		}
	})

	t.Run("create-failure-returns-to-form-with-values", func(t *testing.T) {
		m := openBacklogWithProject(t, server.URL, "p1")
		m, _ = submitStory(m, "Título escrito", "Descripción escrita", "alta", "uno; dos")
		m, _ = apply(m, failure(http.StatusUnprocessableEntity, `{"error":"validation_failed","message":"x","fields":{"title":"mal"}}`))
		m = press(m, tea.KeyEnter)
		if m.Screen != "backlog-form" {
			t.Fatalf("Screen = %q, want backlog-form", m.Screen)
		}
		for label, want := range map[string]string{"Título": "Título escrito", "Descripción": "Descripción escrita", "Prioridad": "alta", "Criterios de aceptación": "uno; dos"} {
			if got := fieldValue(t, m.View(), label); got != want {
				t.Errorf("field %q = %q, want %q kept", label, got, want)
			}
		}
	})

	t.Run("update-failure-returns-to-form-with-values", func(t *testing.T) {
		m := selectStory(openBacklogWithProject(t, server.URL, "p1"), 0)
		m = advanceFields(m, 5)
		m, _ = submitUpdate(m)
		m, _ = apply(m, tui.BacklogResultMsg{Op: tui.BacklogOpUpdate, Response: tui.APIResponse{StatusCode: 404, Body: []byte(`{"error":"story_not_found","message":"nope"}`)}})
		m = press(m, tea.KeyEsc)
		if m.Screen != "backlog-form" {
			t.Fatalf("Screen = %q, want backlog-form", m.Screen)
		}
		if got := fieldValue(t, m.View(), "Título"); got != "Login" {
			t.Errorf("title = %q, want Login kept", got)
		}
		if got := fieldValue(t, m.View(), "Horas estimadas"); got != "5" {
			t.Errorf("hours = %q, want 5 kept", got)
		}
	})

	t.Run("list-failure-returns-to-picker", func(t *testing.T) {
		server, _ := fakeBacklogAPI(t, apiStubs{list: stub{http.StatusNotFound, `{"error":"project_not_found","message":"x"}`}})
		m := loadingBacklog(t, server.URL, "p1")
		m, _ = apply(m, tui.BacklogResultMsg{Op: tui.BacklogOpList, Response: tui.APIResponse{StatusCode: 404, Body: []byte(`{"error":"project_not_found","message":"x"}`)}})
		m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
		if m.Screen != "project-picker" {
			t.Errorf("Screen = %q, want project-picker", m.Screen)
		}
	})
}

func TestBacklogLateListResultIsDiscarded(t *testing.T) {
	server, _ := fakeBacklogAPI(t, apiStubs{})
	intruder := tui.BacklogResultMsg{Op: tui.BacklogOpList, Response: tui.APIResponse{
		StatusCode: 200, Body: []byte(storiesBody(t, []tui.Story{{ID: "x", Title: "Intrusa"}})),
	}}

	t.Run("at-the-menu", func(t *testing.T) {
		m := press(openBacklogWithProject(t, server.URL, "p1"), tea.KeyEsc)
		m, _ = apply(m, intruder)
		if m.Screen != "menu" {
			t.Errorf("Screen = %q, want menu untouched by a late list result", m.Screen)
		}
	})

	t.Run("when-not-loading", func(t *testing.T) {
		m := openBacklogWithProject(t, server.URL, "p1")
		m, _ = apply(m, intruder)
		if m.Screen != "backlog" || strings.Contains(m.View(), "Intrusa") || !strings.Contains(m.View(), "Login") {
			t.Errorf("Screen = %q view = %q, want the loaded list kept", m.Screen, m.View())
		}
	})

	t.Run("create-and-update-always-apply", func(t *testing.T) {
		for _, op := range []tui.BacklogOp{tui.BacklogOpCreate, tui.BacklogOpUpdate} {
			m := press(openBacklogWithProject(t, server.URL, "p1"), tea.KeyEsc)
			m, _ = apply(m, tui.BacklogResultMsg{Op: op, Response: tui.APIResponse{StatusCode: 200, Body: []byte(`{"id":"s1","title":"Aplicada"}`)}})
			if m.Screen != "backlog-result" {
				t.Errorf("op %s: Screen = %q, want backlog-result", op, m.Screen)
			}
		}
	})
}

func TestBacklogBrowsingDoesNotMutate(t *testing.T) {
	server, rec := fakeBacklogAPI(t, apiStubs{})
	m := openBacklogWithProject(t, server.URL, "p1")
	m = press(m, tea.KeyEsc)
	m = press(m, tea.KeyDown)
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	run(t, m, cmd)

	requests := rec.all()
	if len(requests) < 3 {
		t.Fatalf("requests = %+v, want the project load, the stories load and the re-entry", requests)
	}
	for _, request := range requests {
		if request.Method != http.MethodGet {
			t.Errorf("request %s %s, want only GET while browsing", request.Method, request.Path)
		}
	}
}

// backlogScreens builds a model on each backlog screen of interest.
func backlogScreens(t *testing.T, url string) []struct {
	screen string
	label  string // text field label, empty for non-text screens
	model  tui.Model
} {
	t.Helper()
	loaded := openBacklogWithProject(t, url, "p1")
	result, _ := apply(loaded, failure(http.StatusUnprocessableEntity, `{"error":"validation_failed","message":"x","fields":{"title":"mal"}}`))
	picker, _ := startBacklog(url)
	return []struct {
		screen string
		label  string
		model  tui.Model
	}{
		{"project-picker", "", picker},
		{"backlog-form", "Título", press(loaded, tea.KeyEnter)},
		{"backlog", "", loaded},
		{"backlog-result", "", result},
	}
}

func TestQuitOnQIsScopedInBacklogScreens(t *testing.T) {
	server, _ := fakeBacklogAPI(t, apiStubs{})
	for _, tc := range backlogScreens(t, server.URL) {
		for _, char := range []string{"q", "Q"} {
			t.Run(tc.screen+"/"+char, func(t *testing.T) {
				if tc.model.Screen != tc.screen {
					t.Fatalf("Screen = %q, want %q", tc.model.Screen, tc.screen)
				}
				m, cmd := apply(tc.model, keyMsg(char))
				if tc.label == "" {
					if !isQuit(cmd) {
						t.Errorf("%q from %s did not quit", char, tc.screen)
					}
					return
				}
				if isQuit(cmd) {
					t.Fatalf("%q quit the TUI from %s, want it typed into the field", char, tc.screen)
				}
				if m.Screen != tc.screen {
					t.Errorf("Screen = %q, want %q", m.Screen, tc.screen)
				}
				if got := fieldValue(t, m.View(), tc.label); got != char {
					t.Errorf("field %q = %q, want %q", tc.label, got, char)
				}
			})
		}
	}
}

func TestCtrlCQuitsFromEveryBacklogScreen(t *testing.T) {
	server, _ := fakeBacklogAPI(t, apiStubs{})
	for _, tc := range backlogScreens(t, server.URL) {
		t.Run(tc.screen, func(t *testing.T) {
			if tc.model.Screen != tc.screen {
				t.Fatalf("Screen = %q, want %q", tc.model.Screen, tc.screen)
			}
			_, cmd := tc.model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
			if !isQuit(cmd) {
				t.Errorf("Ctrl+C from %s did not quit", tc.screen)
			}
		})
	}
}

func TestSpaceAndBackspaceByRuneInBacklogFields(t *testing.T) {
	server, rec := fakeBacklogAPI(t, apiStubs{})
	fields := []struct {
		screen, label string
		open          func(t *testing.T) tui.Model
	}{
		{"project-picker", "Buscar", func(t *testing.T) tui.Model {
			m, _ := startBacklog(server.URL)
			m, _ = apply(m, keyMsg("/"))
			return m
		}},
		{"backlog-form", "Título", func(t *testing.T) tui.Model {
			return press(openBacklogWithProject(t, server.URL, "p1"), tea.KeyEnter)
		}},
	}
	for _, field := range fields {
		t.Run(field.screen+"/space", func(t *testing.T) {
			m := typeText(field.open(t), "a b")
			if got := fieldValue(t, m.View(), field.label); got != "a b" {
				t.Errorf("field %q = %q, want %q", field.label, got, "a b")
			}
		})
		t.Run(field.screen+"/accented-backspace", func(t *testing.T) {
			m := typeText(field.open(t), "Canción")
			m = press(m, tea.KeyBackspace)
			if got := fieldValue(t, m.View(), field.label); got != "Canció" {
				t.Errorf("field %q = %q, want %q", field.label, got, "Canció")
			}
		})
		t.Run(field.screen+"/multibyte-until-empty", func(t *testing.T) {
			m := typeText(field.open(t), "añá")
			for i, want := range []string{"añ", "a", ""} {
				m = press(m, tea.KeyBackspace)
				got := fieldValue(t, m.View(), field.label)
				if !utf8.ValidString(got) || got != want {
					t.Errorf("backspace %d: field = %q (valid=%v), want %q", i+1, got, utf8.ValidString(got), want)
				}
			}
		})
	}

	t.Run("title-sent-as-valid-utf8", func(t *testing.T) {
		m := press(openBacklogWithProject(t, server.URL, "p1"), tea.KeyEnter)
		m = typeText(m, "Gestión añ")
		m = press(m, tea.KeyBackspace)
		m = typeText(m, "o")
		for i := 0; i < 3; i++ {
			m = press(m, tea.KeyEnter)
		}
		m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
		run(t, m, cmd)
		got := lastRequest(t, rec)
		if !utf8.Valid(got.Body) {
			t.Errorf("request body is not valid UTF-8: %q", got.Body)
		}
		var sent tui.StoryCreateInput
		if err := json.Unmarshal(got.Body, &sent); err != nil {
			t.Fatal(err)
		}
		if sent.Title != "Gestión ao" {
			t.Errorf("title = %q, want %q", sent.Title, "Gestión ao")
		}
	})
}
