package tui

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// defectStoryAPI is the API behind the defect flows that start from the story
// detail: open Sprints and the defect POST. Any other request (stories, defect
// list) is a test failure, so a flow that reloads something it must not is caught.
type defectStoryAPI struct {
	server *httptest.Server

	mu         sync.Mutex
	requests   []string
	posted     map[string]string
	postStatus int
	postBody   string
	defectsTab bool
}

func newDefectStoryAPI(t *testing.T) *defectStoryAPI {
	t.Helper()
	api := &defectStoryAPI{postStatus: http.StatusCreated, postBody: `{"id":"defect-new"}`}
	api.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.mu.Lock()
		defer api.mu.Unlock()
		api.requests = append(api.requests, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/projects/p-1/sprints/open":
			_, _ = w.Write([]byte(`{"sprints":[{"id":"sp-1","sprint_goal":"Sprint uno","is_closed":false}]}`))
		case api.defectsTab && r.Method == http.MethodGet && r.URL.Path == "/projects/p-1/stories":
			_, _ = w.Write([]byte(`{"stories":[{"id":"s-1","title":"Login"}]}`))
		case api.defectsTab && r.Method == http.MethodGet && r.URL.Path == "/projects/p-1/defects":
			_, _ = w.Write([]byte(`{"project_id":"p-1","defects":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/projects/p-1/defects":
			if err := json.NewDecoder(r.Body).Decode(&api.posted); err != nil {
				t.Errorf("decode POST body: %v", err)
			}
			w.WriteHeader(api.postStatus)
			_, _ = w.Write([]byte(api.postBody))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(api.server.Close)
	return api
}

func (a *defectStoryAPI) seen() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.requests...)
}

// defectDetailModel is the detail of "Login" (s-1) of project p-1, ready to run
// its actions.
func defectDetailModel(url string) Model {
	m := NewModel(url)
	m.activeProjectID, m.activeProjectName = "p-1", "Alpha"
	m.Screen = "story-detail"
	m.backlogStories = []Story{
		{ID: "s-1", ProjectID: "p-1", Title: "Login", Priority: "alta", Status: "pendiente"},
		{ID: "s-2", ProjectID: "p-1", Title: "Pago", Priority: "media", Status: "pendiente"},
	}
	m.storyDetail = storyDetailState{storyID: "s-1"}
	return m
}

func runeKey(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

// storyStep sends one message and keeps the command.
func storyStep(m Model, msg tea.Msg) (Model, tea.Cmd) {
	updated, cmd := m.Update(msg)
	return updated.(Model), cmd
}

// storyType types text into the focused field of the form.
func storyType(m Model, text string) Model {
	for _, r := range text {
		m, _ = storyStep(m, runeKey(string(r)))
	}
	return m
}

// openDefectForm presses d on the detail and requires the preloaded form.
func openDefectForm(t *testing.T, m Model) Model {
	t.Helper()
	m, cmd := storyStep(m, runeKey("d"))
	if m.Screen != "defect-create-form" || cmd != nil {
		t.Fatalf("Screen = %q, cmd nil = %v, want the defect form without a command", m.Screen, cmd == nil)
	}
	return m
}

// defectFormToSprints fills the draft and presses Enter on its last field, which
// opens the Sprint selector, and applies the answer of the Sprint query.
func defectFormToSprints(t *testing.T, m Model, description string) Model {
	t.Helper()
	m = storyType(m, description)
	m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyTab})
	m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyTab})
	m, cmd := storyStep(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != "defect-sprint-select" || cmd == nil {
		t.Fatalf("Screen = %q, cmd nil = %v, want the Sprint selector waiting for its answer", m.Screen, cmd == nil)
	}
	m, _ = storyStep(m, cmd())
	return m
}

// defectConfirmation chooses the first open Sprint and opens the confirmation.
func defectConfirmation(t *testing.T, m Model) Model {
	t.Helper()
	m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyEnter}) // choose Sprint uno
	if m.Screen != "defect-create-form" {
		t.Fatalf("Screen = %q after choosing the Sprint, want the form", m.Screen)
	}
	m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != "defect-create-confirm" {
		t.Fatalf("Screen = %q, want the confirmation", m.Screen)
	}
	return m
}

func TestDefectFromStoryOpensFormWithPreloadedStory(t *testing.T) {
	api := newDefectStoryAPI(t)
	m := defectDetailModel(api.server.URL)

	m = openDefectForm(t, m)

	if view := stripANSI(m.View()); !strings.Contains(view, "Historia relacionada: Login") || strings.Contains(view, "Elegir historia") {
		t.Errorf("the form should show the preloaded story and no story selector:\n%s", view)
	}
	if form := m.defectCreateForm; form.storyID != "s-1" || form.storyLabel != "Login" || form.sprintID != "" {
		t.Errorf("form = %+v, want story s-1 Login preloaded and no Sprint yet", form)
	}
	if got := api.seen(); len(got) != 0 {
		t.Errorf("requests = %v, want none: no story query is needed", got)
	}
}

func TestDefectFromStoryEnterOnLastFieldOpensSprintSelector(t *testing.T) {
	api := newDefectStoryAPI(t)
	m := openDefectForm(t, defectDetailModel(api.server.URL))

	m = defectFormToSprints(t, m, "No guarda")

	view := stripANSI(m.View())
	if !strings.Contains(view, "Sprint de detección") || !strings.Contains(view, "Sprint uno") || strings.Contains(view, "Elegí la historia relacionada") {
		t.Errorf("the Sprint selector should list the open Sprints and no story selector:\n%s", view)
	}
	if got := strings.Join(api.seen(), "|"); got != "GET /projects/p-1/sprints/open" {
		t.Errorf("requests = %q, want only the open Sprint query", got)
	}
}

func TestDefectFromStoryRowEnterAndShortcutAreEquivalent(t *testing.T) {
	api := newDefectStoryAPI(t)
	byShortcut := openDefectForm(t, defectDetailModel(api.server.URL))

	byRow := defectDetailModel(api.server.URL)
	byRow, _ = storyStep(byRow, tea.KeyMsg{Type: tea.KeyDown})
	byRow, _ = storyStep(byRow, tea.KeyMsg{Type: tea.KeyDown})
	byRow = openDefectEnter(t, byRow)

	if byRow.defectCreateForm != byShortcut.defectCreateForm || byRow.returnTo != byShortcut.returnTo {
		t.Errorf("row+Enter opened %+v (returnTo %v), d opened %+v (returnTo %v)",
			byRow.defectCreateForm, byRow.returnTo, byShortcut.defectCreateForm, byShortcut.returnTo)
	}
	if view := stripANSI(byRow.View()); !strings.Contains(view, "Historia relacionada: Login") {
		t.Errorf("the row should preload the story too:\n%s", view)
	}
}

// defectDraftToConfirmation fills the draft of "No guarda" from the detail, picks
// the Sprint and stops on the confirmation, with only the Sprint query issued.
func defectDraftToConfirmation(t *testing.T, api *defectStoryAPI) Model {
	t.Helper()
	m := openDefectForm(t, defectDetailModel(api.server.URL))
	return defectConfirmation(t, defectFormToSprints(t, m, "No guarda"))
}

// submitDefect presses s on the confirmation and requires the POST to be in flight.
func submitDefect(t *testing.T, m Model) (Model, tea.Cmd) {
	t.Helper()
	m, cmd := storyStep(m, runeKey("s"))
	if m.Screen != "defect-create-confirm" || !m.defectCreateLoading || cmd == nil {
		t.Fatalf("Screen = %q, loading = %v, cmd nil = %v, want the confirmation with the POST in flight", m.Screen, m.defectCreateLoading, cmd == nil)
	}
	return m, cmd
}

func TestDefectFromStoryEscOnFormReturnsToDetailAndDiscardsDraft(t *testing.T) {
	api := newDefectStoryAPI(t)
	m := openDefectForm(t, defectDetailModel(api.server.URL))
	m = storyType(m, "Borrador")

	m, cmd := storyStep(m, tea.KeyMsg{Type: tea.KeyEsc})

	if m.Screen != "story-detail" || cmd != nil {
		t.Fatalf("Screen = %q, cmd nil = %v, want the detail without a command", m.Screen, cmd == nil)
	}
	if got := fieldValueOf(stripANSI(m.View()), "Título"); got != "Login" {
		t.Errorf("the detail should show Login, got %q", got)
	}
	m = openDefectForm(t, m)
	if form := m.defectCreateForm; form.description != "" || form.storyID != "s-1" || form.storyLabel != "Login" {
		t.Errorf("reopened form = %+v, want an empty description and Login preloaded", form)
	}
	if got := api.seen(); len(got) != 0 {
		t.Errorf("requests = %v, want none", got)
	}
}

// fieldValueOf returns the text after "label:" on the first line that has it.
func fieldValueOf(view, label string) string {
	for _, line := range strings.Split(view, "\n") {
		line = strings.Trim(line, "│ ")
		if strings.HasPrefix(line, label+": ") {
			return strings.TrimSpace(strings.TrimPrefix(line, label+": "))
		}
	}
	return ""
}

func TestDefectFromStoryEscOnSprintSelectorReturnsToFormKeepingDraft(t *testing.T) {
	api := newDefectStoryAPI(t)
	m := openDefectForm(t, defectDetailModel(api.server.URL))
	m = storyType(m, "No guarda")
	m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyTab})
	m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyRight}) // severity: media
	m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyTab})
	m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyRight}) // status: en_correccion
	m, cmd := storyStep(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != "defect-sprint-select" || cmd == nil {
		t.Fatalf("Screen = %q, cmd nil = %v, want the Sprint selector", m.Screen, cmd == nil)
	}
	m, _ = storyStep(m, cmd())

	m, cmd = storyStep(m, tea.KeyMsg{Type: tea.KeyEsc})

	if m.Screen != "defect-create-form" || cmd != nil {
		t.Fatalf("Screen = %q, cmd nil = %v, want the form (never a story selector) without a command", m.Screen, cmd == nil)
	}
	form := m.defectCreateForm
	if form.description != "No guarda" || form.severity() != "media" || form.status() != "en_correccion" || form.storyID != "s-1" || form.storyLabel != "Login" {
		t.Errorf("form = %+v, want the draft and the story kept", form)
	}
	if view := stripANSI(m.View()); !strings.Contains(view, "Historia relacionada: Login") {
		t.Errorf("the form should still show the story:\n%s", view)
	}
}

func TestDefectFromStoryCancelWithNReturnsToDetailWithoutPost(t *testing.T) {
	api := newDefectStoryAPI(t)
	m := defectDraftToConfirmation(t, api)

	m, cmd := storyStep(m, runeKey("n"))

	if m.Screen != "story-detail" || cmd != nil {
		t.Fatalf("Screen = %q, cmd nil = %v, want the detail without a command", m.Screen, cmd == nil)
	}
	if got := strings.Join(api.seen(), "|"); got != "GET /projects/p-1/sprints/open" {
		t.Errorf("requests = %q, want no defect registration", got)
	}
	if view := stripANSI(m.View()); strings.Contains(view, "Defecto registrado") {
		t.Errorf("cancelling must not announce a registration:\n%s", view)
	}
}

func TestDefectFromStorySuccessReturnsToDetailWithToast(t *testing.T) {
	api := newDefectStoryAPI(t)
	m := defectDraftToConfirmation(t, api)
	m, post := submitDefect(t, m)

	m, reload := storyStep(m, post())

	if m.Screen != "story-detail" || reload != nil {
		t.Fatalf("Screen = %q, reload nil = %v, want the detail without reloading the defects", m.Screen, reload == nil)
	}
	view := stripANSI(m.View())
	if !strings.Contains(view, "✓ Defecto registrado: No guarda") || !strings.Contains(view, "Detalle de la historia") || strings.Contains(view, "Registrar defecto") {
		t.Errorf("the detail should carry the toast and not the defect list:\n%s", view)
	}
	if got := strings.Join(api.seen(), "|"); got != "GET /projects/p-1/sprints/open|POST /projects/p-1/defects" {
		t.Errorf("requests = %q, want no defect list query", got)
	}
	// The next key clears the toast and keeps its normal effect.
	m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyDown})
	if view := stripANSI(m.View()); strings.Contains(view, "Defecto registrado") {
		t.Errorf("the toast survived the next key:\n%s", view)
	}
}

func TestDefectFromStorySendsPreloadedStoryIDInBody(t *testing.T) {
	api := newDefectStoryAPI(t)
	m := defectDraftToConfirmation(t, api)
	m, post := submitDefect(t, m)

	_, _ = storyStep(m, post())

	want := map[string]string{"story_id": "s-1", "sprint_id": "sp-1", "description": "No guarda", "severity": "alta", "status": "detectado"}
	if len(api.posted) != len(want) {
		t.Errorf("POST body = %v, want exactly %v", api.posted, want)
	}
	for field, value := range want {
		if api.posted[field] != value {
			t.Errorf("POST body %q = %q, want %q", field, api.posted[field], value)
		}
	}
	if got := strings.Join(api.seen(), "|"); got != "GET /projects/p-1/sprints/open|POST /projects/p-1/defects" {
		t.Errorf("requests = %q, want one Sprint query and one POST", got)
	}
}

// [caracterización] Born green: the confirmation already goes back to the draft.
func TestDefectFromStoryEscOnConfirmReturnsToFormKeepingDraft(t *testing.T) {
	api := newDefectStoryAPI(t)
	m := defectDraftToConfirmation(t, api)

	m, cmd := storyStep(m, tea.KeyMsg{Type: tea.KeyEsc})

	if m.Screen != "defect-create-form" || cmd != nil {
		t.Fatalf("Screen = %q, cmd nil = %v, want the form without a command", m.Screen, cmd == nil)
	}
	if form := m.defectCreateForm; form.description != "No guarda" || form.storyID != "s-1" || form.sprintID != "sp-1" {
		t.Errorf("form = %+v, want the full draft kept", form)
	}
}

// [caracterización] Born green: a rejection already stays on the confirmation.
func TestDefectFromStoryRejectionKeepsConfirmationWithValues(t *testing.T) {
	for _, tc := range []struct {
		name   string
		result func(Model) defectCreateResult
		want   string
	}{
		{"HTTP 422", func(m Model) defectCreateResult {
			return defectCreateResult{ProjectID: "p-1", RequestID: m.defectCreateRequestID,
				Response: APIResponse{StatusCode: http.StatusUnprocessableEntity, Body: []byte(`story not valid`)}}
		}, "Error HTTP 422: story not valid"},
		{"error de conexión", func(m Model) defectCreateResult {
			return defectCreateResult{ProjectID: "p-1", RequestID: m.defectCreateRequestID, Err: errors.New("connection refused")}
		}, "Error de conexión: connection refused"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := newDefectStoryAPI(t)
			m, _ := submitDefect(t, defectDraftToConfirmation(t, api))

			m, cmd := storyStep(m, tc.result(m))

			view := stripANSI(m.View())
			if m.Screen != "defect-create-confirm" || m.defectCreateLoading || cmd != nil || m.toast != "" {
				t.Fatalf("Screen = %q, loading = %v, toast = %q, want the confirmation without success", m.Screen, m.defectCreateLoading, m.toast)
			}
			for _, want := range []string{tc.want, "No guarda", "Login", "Sprint uno"} {
				if !strings.Contains(view, want) {
					t.Errorf("the confirmation lacks %q:\n%s", want, view)
				}
			}
			// The user can retry with s or go back to the draft with Esc.
			if _, retry := storyStep(m, runeKey("s")); retry == nil {
				t.Error("s should send the registration again after a rejection")
			}
			if back, _ := storyStep(m, tea.KeyMsg{Type: tea.KeyEsc}); back.Screen != "defect-create-form" {
				t.Errorf("Esc: Screen = %q, want the draft", back.Screen)
			}
		})
	}
}

// [caracterización] Born green: the in-flight POST already blocks s, n and Esc.
func TestDefectFromStoryDoubleSubmitEmitsSingleRequest(t *testing.T) {
	api := newDefectStoryAPI(t)
	m, post := submitDefect(t, defectDraftToConfirmation(t, api))

	for name, key := range map[string]tea.KeyMsg{"s": runeKey("s"), "n": runeKey("n"), "Esc": {Type: tea.KeyEsc}} {
		next, cmd := storyStep(m, key)
		if cmd != nil || next.Screen != "defect-create-confirm" || !next.defectCreateLoading {
			t.Errorf("%s: Screen = %q, loading = %v, cmd nil = %v, want the confirmation still waiting", name, next.Screen, next.defectCreateLoading, cmd == nil)
		}
	}
	_, _ = storyStep(m, post())
	if got := strings.Join(api.seen(), "|"); got != "GET /projects/p-1/sprints/open|POST /projects/p-1/defects" {
		t.Errorf("requests = %q, want a single POST", got)
	}
}

// [caracterización] Born green: stale answers are already discarded by screen,
// request and project.
func TestDefectFromStoryLateResponsesAreDiscarded(t *testing.T) {
	t.Run("Sprints tras Esc", func(t *testing.T) {
		api := newDefectStoryAPI(t)
		m := openDefectForm(t, defectDetailModel(api.server.URL))
		m = storyType(m, "No guarda")
		m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyTab})
		m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyTab})
		m, query := storyStep(m, tea.KeyMsg{Type: tea.KeyEnter})
		m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyEsc})
		if m.Screen != "defect-create-form" {
			t.Fatalf("Screen = %q after Esc, want the form", m.Screen)
		}

		late, cmd := storyStep(m, query())

		if late.Screen != "defect-create-form" || cmd != nil || len(late.defectSprintSelector.items) != 0 {
			t.Errorf("Screen = %q, items = %d, want the form untouched by the late answer", late.Screen, len(late.defectSprintSelector.items))
		}
		if late.defectCreateForm != m.defectCreateForm {
			t.Errorf("form = %+v, want %+v unchanged", late.defectCreateForm, m.defectCreateForm)
		}
	})
	t.Run("respuesta de un proyecto anterior", func(t *testing.T) {
		api := newDefectStoryAPI(t)
		m, post := submitDefect(t, defectDraftToConfirmation(t, api))
		m, _ = m.selectProject(ProjectSummary{ID: "p-2", Name: "Beta", Status: "planned"})
		screen := m.Screen

		late, cmd := storyStep(m, post())

		if late.Screen != screen || cmd != nil || late.toast != "" {
			t.Errorf("Screen = %q (was %q), toast = %q, want the answer of p-1 discarded", late.Screen, screen, late.toast)
		}
	})
}

// openDefectEnter presses Enter on the highlighted row and requires the form.
func openDefectEnter(t *testing.T, m Model) Model {
	t.Helper()
	m, cmd := storyStep(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != "defect-create-form" || cmd != nil {
		t.Fatalf("Screen = %q, cmd nil = %v, want the defect form without a command", m.Screen, cmd == nil)
	}
	return m
}

// serveDefectsTab also answers the story list and the defect list, which only
// the flow that starts from the Defectos tab needs.
func (a *defectStoryAPI) serveDefectsTab() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.defectsTab = true
}

// [caracterización] Born green: the failure of the Sprint query is already shown
// inside the selector, with retry and a way back to the form.
func TestDefectFromStorySprintQueryFailureShowsErrorAndAllowsRetry(t *testing.T) {
	api := newDefectStoryAPI(t)
	m := openDefectForm(t, defectDetailModel(api.server.URL))
	m = storyType(m, "No guarda")
	m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyTab})
	m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyTab})
	m, query := storyStep(m, tea.KeyMsg{Type: tea.KeyEnter})
	if query == nil || m.Screen != "defect-sprint-select" {
		t.Fatalf("Screen = %q, query nil = %v, want the Sprint selector loading", m.Screen, query == nil)
	}

	m, _ = storyStep(m, defectOptionsResult{step: "sprints", projectID: "p-1", requestID: m.defectSprintRequestID,
		response: APIResponse{StatusCode: http.StatusInternalServerError, Body: []byte(`sprints down`)}})

	if view := stripANSI(m.View()); m.Screen != "defect-sprint-select" || !strings.Contains(view, "Error HTTP 500: sprints down") {
		t.Fatalf("Screen = %q, want the error inside the selector:\n%s", m.Screen, view)
	}
	if _, retry := storyStep(m, runeKey("r")); retry == nil {
		t.Error("r should query the Sprints again")
	}
	if back, _ := storyStep(m, tea.KeyMsg{Type: tea.KeyEsc}); back.Screen != "defect-create-form" || back.defectCreateForm.description != "No guarda" {
		t.Errorf("Esc: Screen = %q, form = %+v, want the form with its draft", back.Screen, back.defectCreateForm)
	}
}

func TestDefectFormHelpDependsOnWhereTheStoryComesFrom(t *testing.T) {
	for _, tc := range []struct {
		name    string
		form    defectCreateForm
		want    string
		notWant string
	}{
		{"historia precargada desde el detalle", defectCreateForm{storyID: "s-1", storyLabel: "Login"},
			"Enter en el último campo elige el Sprint de detección.", "elige la historia"},
		{"alta desde la pestaña Defectos", defectCreateForm{},
			"Enter en el último campo elige la historia.", "elige el Sprint"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := NewModel("http://example.invalid")
			m.Screen, m.activeProjectID, m.defectCreateForm = "defect-create-form", "p-1", tc.form
			m.width = 200 // keep the hint on one line

			view := stripANSI(m.viewDefectCreateForm())

			if !strings.Contains(view, tc.want) || strings.Contains(view, tc.notWant) {
				t.Errorf("hint should contain %q and not %q:\n%s", tc.want, tc.notWant, view)
			}
		})
	}
}

func TestNInDefectsResetsResidualReturnTo(t *testing.T) {
	m := NewModel("http://example.invalid")
	m.Screen, m.activeProjectID = "defects", "p-1"
	m.returnTo = returnStoryDetail // left over from a flow of the detail

	m, _ = storyStep(m, runeKey("n"))

	if m.Screen != "defect-create-form" || m.returnTo != returnDefault {
		t.Fatalf("Screen = %q, returnTo = %v, want the form that goes back to the list", m.Screen, m.returnTo)
	}
	m.defectCreateForm.storyID, m.defectCreateForm.sprintID, m.defectCreateForm.description = "s-1", "sp-1", "Cache crash"
	m, _ = m.openDefectCreateConfirm()
	m, post := m.submitDefectCreate()
	if post == nil {
		t.Fatal("the confirmation should send the registration")
	}

	m, reload := storyStep(m, defectCreateResult{ProjectID: "p-1", RequestID: m.defectCreateRequestID, Response: APIResponse{StatusCode: http.StatusCreated, Body: []byte(`{"id":"defect-new"}`)}})

	if m.Screen != "defects" || !m.defectLoading || reload == nil {
		t.Errorf("Screen = %q, loading = %v, reload nil = %v, want the defect list reloading", m.Screen, m.defectLoading, reload == nil)
	}
}

// defectsTabDraft starts a defect with n from the Defectos tab, walks the story
// and Sprint selectors and stops on the confirmation.
func defectsTabDraft(t *testing.T, api *defectStoryAPI, stopAt string) Model {
	t.Helper()
	api.serveDefectsTab()
	m := NewModel(api.server.URL)
	m.Screen, m.activeProjectID = "defects", "p-1"
	m, _ = storyStep(m, runeKey("n"))
	m = storyType(m, "Cache crash")
	m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyTab})
	m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyTab})
	m, stories := storyStep(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen != "defect-story-select" || stories == nil {
		t.Fatalf("Screen = %q, want the story selector of the Defectos flow", m.Screen)
	}
	if stopAt == "defect-story-select" {
		return m
	}
	m, _ = storyStep(m, stories())
	m, sprints := storyStep(m, tea.KeyMsg{Type: tea.KeyEnter}) // choose the story
	m, _ = storyStep(m, sprints())
	if stopAt == "defect-sprint-select" {
		return m
	}
	m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyEnter}) // choose the Sprint
	m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyEnter}) // open the confirmation
	if m.Screen != "defect-create-confirm" {
		t.Fatalf("Screen = %q, want the confirmation", m.Screen)
	}
	return m
}

// [caracterización] Born green: the flow of the Defectos tab must not change.
func TestDefectCreationFromDefectsTabKeepsStoryPickerAndReturnsToList(t *testing.T) {
	t.Run("Enter en el último campo abre el selector de historias", func(t *testing.T) {
		m := defectsTabDraft(t, newDefectStoryAPI(t), "defect-story-select")

		if view := stripANSI(m.View()); !strings.Contains(view, "Defecto · Historia") {
			t.Errorf("want the story selector:\n%s", view)
		}
		if m.returnTo != returnDefault {
			t.Errorf("returnTo = %v, want returnDefault", m.returnTo)
		}
	})
	t.Run("Esc del selector de Sprints vuelve al selector de historias", func(t *testing.T) {
		m := defectsTabDraft(t, newDefectStoryAPI(t), "defect-sprint-select")

		m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyEsc})

		if m.Screen != "defect-story-select" {
			t.Errorf("Screen = %q, want the story selector", m.Screen)
		}
	})
	t.Run("Esc del formulario vuelve a la lista", func(t *testing.T) {
		m := defectsTabDraft(t, newDefectStoryAPI(t), "defect-create-confirm")
		m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyEsc})

		m, _ = storyStep(m, tea.KeyMsg{Type: tea.KeyEsc})

		if m.Screen != "defects" {
			t.Errorf("Screen = %q, want the defect list", m.Screen)
		}
	})
	t.Run("n en la confirmación vuelve a la lista", func(t *testing.T) {
		m := defectsTabDraft(t, newDefectStoryAPI(t), "defect-create-confirm")

		m, _ = storyStep(m, runeKey("n"))

		if m.Screen != "defects" {
			t.Errorf("Screen = %q, want the defect list", m.Screen)
		}
	})
	t.Run("el éxito vuelve a la lista recargada", func(t *testing.T) {
		m := defectsTabDraft(t, newDefectStoryAPI(t), "defect-create-confirm")
		m, post := submitDefect(t, m)

		m, reload := storyStep(m, post())

		if m.Screen != "defects" || !m.defectLoading || reload == nil {
			t.Fatalf("Screen = %q, loading = %v, reload nil = %v, want the list reloading", m.Screen, m.defectLoading, reload == nil)
		}
		if view := stripANSI(m.View()); !strings.Contains(view, "✓ Defecto registrado: Cache crash") {
			t.Errorf("the list lacks the toast:\n%s", view)
		}
	})
}
