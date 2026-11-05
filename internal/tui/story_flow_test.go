package tui

import (
	"net/http"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// detailModel builds a model on the story detail of "Login" (s1) with the
// backlog already loaded, so each test only changes the state it exercises.
func detailModel() Model {
	m := NewModel("http://127.0.0.1:1")
	m.activeProjectID = "p1"
	m.activeProjectName = "Alpha"
	m.Screen = "story-detail"
	m.backlogStories = []Story{
		{ID: "s1", ProjectID: "p1", Title: "Login", Priority: "alta", Status: "pendiente"},
		{ID: "s2", ProjectID: "p1", Title: "Pago", Priority: "media", Status: "pendiente"},
	}
	m.storyDetail = storyDetailState{storyID: "s1"}
	return m
}

func TestStoryDetailStatesGateAvailability(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Model)
		want   bool
	}{
		{"listo", func(m *Model) {}, true},
		{"cargando", func(m *Model) { m.backlogLoading = true }, false},
		{"error de recarga", func(m *Model) { m.backlogMessage = "Error HTTP 500: boom" }, false},
		{"historia ausente", func(m *Model) { m.storyDetail.storyID = "gone" }, false},
		{"la carga prevalece sobre el error", func(m *Model) {
			m.backlogLoading = true
			m.backlogMessage = "Error HTTP 500: boom"
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := detailModel()
			tc.mutate(&m)

			if got := m.storyActionsAvailable(); got != tc.want {
				t.Errorf("storyActionsAvailable() = %v, want %v", got, tc.want)
			}

			// Esc goes back to the backlog in every state, keeping its cursor and
			// filter and without reloading it.
			m.backlogCursor, m.backlogFilter = 1, "pa"
			next, cmd := m.updateStoryDetail(tea.KeyMsg{Type: tea.KeyEsc})
			if next.Screen != "backlog" || cmd != nil {
				t.Errorf("Esc: Screen = %q, cmd nil = %v, want backlog without a command", next.Screen, cmd == nil)
			}
			if next.backlogCursor != 1 || next.backlogFilter != "pa" {
				t.Errorf("Esc: cursor/filter = %d/%q, want 1/\"pa\" kept", next.backlogCursor, next.backlogFilter)
			}
		})
	}
}

func TestStoryDetailViewIndicatesLoadingErrorAndMissing(t *testing.T) {
	const missing = "ya no está en el backlog"
	for _, tc := range []struct {
		name    string
		mutate  func(*Model)
		want    []string
		notWant []string
	}{
		{"cargando", func(m *Model) { m.backlogLoading = true }, []string{"Cargando"}, []string{missing, "Error HTTP 500", "Login"}},
		{"error de recarga", func(m *Model) { m.backlogMessage = "Error HTTP 500: boom" }, []string{"Error HTTP 500: boom"}, []string{missing, "Cargando", "Login"}},
		{"historia ausente", func(m *Model) { m.storyDetail.storyID = "gone" }, []string{missing}, []string{"Cargando", "Error HTTP 500"}},
		{"la carga prevalece sobre el error", func(m *Model) {
			m.backlogLoading = true
			m.backlogMessage = "Error HTTP 500: boom"
		}, []string{"Cargando"}, []string{missing, "Error HTTP 500"}},
		{"listo", func(m *Model) {}, []string{"Login", "alta"}, []string{missing, "Cargando", "Error HTTP 500"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := detailModel()
			tc.mutate(&m)

			view := stripANSI(m.viewStoryDetail())

			for _, want := range tc.want {
				if !strings.Contains(view, want) {
					t.Errorf("view lacks %q:\n%s", want, view)
				}
			}
			for _, notWant := range tc.notWant {
				if strings.Contains(view, notWant) {
					t.Errorf("view must not contain %q:\n%s", notWant, view)
				}
			}
		})
	}
}

func TestApplyBacklogResultAcceptsStoryDetailScreen(t *testing.T) {
	body := []byte(`{"project_id":"p1","stories":[{"id":"s1","project_id":"p1","title":"Login","story_points":8}]}`)
	for _, tc := range []struct {
		name       string
		screen     string
		wantLoaded bool
	}{
		{"detalle de la historia", "story-detail", true},
		{"lista del backlog", "backlog", true},
		{"otra pantalla descarta la respuesta", "dashboard", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := detailModel()
			m.Screen = tc.screen
			m.backlogStories = nil
			m.backlogLoading = true

			next, cmd := m.applyBacklogResult(BacklogResultMsg{Op: BacklogOpList, Response: APIResponse{StatusCode: 200, Body: body}})

			if cmd != nil || next.Screen != tc.screen {
				t.Errorf("Screen = %q, cmd nil = %v, want %q without a command", next.Screen, cmd == nil, tc.screen)
			}
			if !tc.wantLoaded {
				if len(next.backlogStories) != 0 || !next.backlogLoading {
					t.Errorf("a result on %q must be discarded, got %d stories (loading %v)", tc.screen, len(next.backlogStories), next.backlogLoading)
				}
				return
			}
			if next.backlogLoading || len(next.backlogStories) != 1 || next.backlogStories[0].Title != "Login" {
				t.Fatalf("loading = %v, stories = %+v, want the repopulated backlog", next.backlogLoading, next.backlogStories)
			}
			if story, found := next.detailStory(); tc.screen == "story-detail" && (!found || story.StoryPoints == nil || *story.StoryPoints != 8) {
				t.Errorf("detailStory() = %+v, %v, want the reloaded story with 8 points", story, found)
			}
		})
	}
}

// The actions of the detail are ignored unless it is ready; the ready case keeps
// the table honest, because it is the one that must open the flow.
func TestStoryDetailActionsDoNothingWhileLoading(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mutate    func(*Model)
		wantFlow  bool
		wantStory string
	}{
		{"listo", func(m *Model) {}, true, "s1"},
		{"cargando", func(m *Model) { m.backlogLoading = true }, false, ""},
		{"error de recarga", func(m *Model) { m.backlogMessage = "Error HTTP 500: boom" }, false, ""},
		{"historia ausente", func(m *Model) { m.storyDetail.storyID = "gone" }, false, ""},
	} {
		for name, key := range map[string]tea.KeyMsg{
			"p":     {Type: tea.KeyRunes, Runes: []rune{'p'}},
			"t":     {Type: tea.KeyRunes, Runes: []rune{'t'}},
			"Enter": {Type: tea.KeyEnter},
		} {
			t.Run(tc.name+"/"+name, func(t *testing.T) {
				m := detailModel()
				tc.mutate(&m)

				next, cmd := m.updateStoryDetail(key)

				if !tc.wantFlow {
					if cmd != nil || next.Screen != "story-detail" {
						t.Errorf("Screen = %q, cmd nil = %v, want the detail untouched without a command", next.Screen, cmd == nil)
					}
					return
				}
				if cmd == nil || next.Screen != "section" || next.section.selected.ID != tc.wantStory {
					t.Errorf("Screen = %q, cmd nil = %v, story = %q, want a flow of %q", next.Screen, cmd == nil, next.section.selected.ID, tc.wantStory)
				}
			})
		}
	}
}

// Only the way out is offered while the detail has no actions to run; the ready
// case pins the footer the actions add, so the table cannot pass vacuously.
func TestStoryDetailStatesShowOnlyEscInFooter(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Model)
		want   string
	}{
		{"listo", func(m *Model) {}, "↑/↓ mover · Enter elegir acción · Esc backlog · ? más"},
		{"cargando", func(m *Model) { m.backlogLoading = true }, "Esc backlog · ? más"},
		{"error de recarga", func(m *Model) { m.backlogMessage = "Error HTTP 500: boom" }, "Esc backlog · ? más"},
		{"historia ausente", func(m *Model) { m.storyDetail.storyID = "gone" }, "Esc backlog · ? más"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := detailModel()
			tc.mutate(&m)

			if got := strings.Join(strings.Fields(stripANSI(m.footer())), " "); got != tc.want {
				t.Errorf("footer = %q, want %q", got, tc.want)
			}
		})
	}
}

const effortTasksOfS1 = `{"project_id":"p1","story_id":"s1","tasks":[{"id":"t1","project_id":"p1","story_id":"s1","title":"Diseñar"}]}`

// effortTasksStep opens "Registrar esfuerzo" for the story s1 of the detail
// model and leaves the flow waiting for the answer to its task query.
func effortTasksStep(t *testing.T) Model {
	t.Helper()
	m := detailModel()
	m, cmd := m.openStoryFlow(sectionEffort, m.backlogStories[0])
	if cmd == nil || m.Screen != "section" || m.section.step != sectionStepTasks || !m.section.selector.loading {
		t.Fatalf("Screen = %q, step = %v, loading = %v, want the task selector waiting for its answer", m.Screen, m.section.step, m.section.selector.loading)
	}
	return m
}

func TestSectionResultFromAnotherStoryIsDropped(t *testing.T) {
	m := effortTasksStep(t)
	late := SectionResultMsg{Kind: sectionEffort, Step: sectionStepTasks, StoryID: "s2",
		Response: APIResponse{StatusCode: 200, Body: []byte(effortTasksOfS1)}}

	next, cmd := m.applySectionResult(late)

	if cmd != nil || !next.section.selector.loading || len(next.section.selector.items) != 0 {
		t.Errorf("a result of another story must be dropped: cmd nil = %v, loading = %v, items = %+v",
			cmd == nil, next.section.selector.loading, next.section.selector.items)
	}
}

func TestSectionResultWithMatchingStoryIDStillApplies(t *testing.T) {
	m := effortTasksStep(t)
	current := SectionResultMsg{Kind: sectionEffort, Step: sectionStepTasks, StoryID: "s1",
		Response: APIResponse{StatusCode: 200, Body: []byte(effortTasksOfS1)}}

	next, _ := m.applySectionResult(current)

	items := next.section.selector.items
	if next.section.selector.loading || len(items) != 2 || items[0].Label != wholeStoryLabel || items[1].Label != "Diseñar" {
		t.Errorf("loading = %v, items = %+v, want the whole-story row and Diseñar", next.section.selector.loading, items)
	}
}

func TestSprintStoryDetailAppliesItsTasks(t *testing.T) {
	m := detailModel()
	m.Screen = "story-detail"
	m.sprintStories = []Story{{ID: "s1", Title: "Login"}}
	m.storyDetail = storyDetailState{storyID: "s1", origin: storyDetailFromSprint, tasksLoading: true}

	m, _ = m.applyStoryTasks(storyTasksResult{StoryID: "s1", Response: APIResponse{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"tasks":[{"id":"t1","title":"Diseñar","estimated_hours":2}]}`),
	}})

	if len(m.storyDetail.tasks) != 1 || m.storyDetail.tasks[0].Title != "Diseñar" {
		t.Fatalf("tasks = %+v, want the task of the story", m.storyDetail.tasks)
	}
	if view := m.View(); !strings.Contains(view, "Diseñar") || !strings.Contains(view, "2 h") {
		t.Errorf("the detail should render the tasks:\n%s", view)
	}
}

func TestSprintStoryDetailDiscardsTasksOfAnotherStory(t *testing.T) {
	m := detailModel()
	m.Screen = "story-detail"
	m.storyDetail = storyDetailState{storyID: "s1", origin: storyDetailFromSprint, tasksLoading: true}

	m, _ = m.applyStoryTasks(storyTasksResult{StoryID: "s2", Response: APIResponse{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"tasks":[{"id":"t1","title":"Diseñar"}]}`),
	}})

	if len(m.storyDetail.tasks) != 0 || !m.storyDetail.tasksLoading {
		t.Errorf("tasks = %+v loading = %v, want an unanswered request for the open detail", m.storyDetail.tasks, m.storyDetail.tasksLoading)
	}
}

func TestCompletingFromSprintDetailSendsCompletion(t *testing.T) {
	m := detailModel()
	m.Screen = "story-detail"
	m.activeProjectID = "p1"
	m.selectedSprintID = "sp1"
	m.sprintStories = []Story{{ID: "s1", Title: "Login"}}
	m.storyDetail = storyDetailState{storyID: "s1", origin: storyDetailFromSprint, cursor: 4}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("c on the Sprint detail should send the completion")
	}
	if result, ok := cmd().(SprintResultMsg); !ok || result.Op != "complete" {
		t.Fatalf("command result = %#v, want a complete Sprint result", cmd())
	}

	updated, reload := m.Update(SprintResultMsg{Op: "complete", Response: APIResponse{StatusCode: http.StatusCreated, Body: []byte(`{}`)}})
	m = updated.(Model)
	if m.Screen != "sprint-assigned-list" || reload == nil || !strings.Contains(m.toast, "Historia completada") {
		t.Fatalf("completion = screen %q reload nil %v toast %q, want the Sprint stories with a notice", m.Screen, reload == nil, m.toast)
	}
}

func TestTaskCreationFromSprintDetailReturnsToDetail(t *testing.T) {
	m := detailModel()
	m.Screen = "sprint-task-form"
	m.activeProjectID = "p1"
	m.storyDetail = storyDetailState{storyID: "s1", origin: storyDetailFromSprint}
	m.sprintSaving, m.sprintSavingLabel = true, "Diseñar"

	updated, cmd := m.Update(SprintResultMsg{Op: "tasks", Response: APIResponse{StatusCode: http.StatusCreated, Body: []byte(`{}`)}})
	m = updated.(Model)

	if m.Screen != "story-detail" || !m.storyDetail.tasksLoading || cmd == nil {
		t.Fatalf("task creation = screen %q loading %v cmd nil %v, want the detail reloading its tasks", m.Screen, m.storyDetail.tasksLoading, cmd == nil)
	}
	if !strings.Contains(m.toast, "Tarea creada: Diseñar") {
		t.Errorf("toast = %q, want the task notice", m.toast)
	}
}

func TestSelectProjectResetsStoryDetail(t *testing.T) {
	m := detailModel()
	m.storyDetail.cursor = 1

	next, _ := m.selectProject(ProjectSummary{ID: "p2", Name: "Beta", Status: "planned"})

	if next.storyDetail.storyID != "" || next.storyDetail.origin != storyDetailFromBacklog || next.storyDetail.tasks != nil {
		t.Errorf("storyDetail = %+v, want the zero value after changing project", next.storyDetail)
	}
	if next.activeProjectID != "p2" {
		t.Errorf("activeProjectID = %q, want p2", next.activeProjectID)
	}
}

// flowScreenModel builds the model on screen with the flow state of a story flow
// of kind at step, started from a detail opened from origin.
func flowScreenModel(screen string, kind sectionKind, step sectionStep, origin storyDetailOrigin) Model {
	m := detailModel()
	m.Screen = screen
	m.section.kind, m.section.step = kind, step
	m.storyDetail.origin = origin
	return m
}

func TestCurrentTabAndCrumbPerScreenAndOrigin(t *testing.T) {
	const (
		backlogFlow = "Proyectos › Alpha › Backlog › Detalle"
		sprintFlow  = "Proyectos › Alpha › Sprints › Historias › Detalle"
	)
	for _, tc := range []struct {
		name     string
		m        Model
		wantTab  workspaceTab
		wantPath string
	}{
		{"detalle desde Backlog", flowScreenModel("story-detail", "", sectionStepNone, storyDetailFromBacklog), tabBacklog, backlogFlow},
		{"detalle desde Sprints", flowScreenModel("story-detail", "", sectionStepNone, storyDetailFromSprint), tabSprints, sprintFlow},
		{"Estimar participantes desde el detalle", flowScreenModel("section", sectionPlanningPoker, sectionStepMembers, storyDetailFromBacklog), tabBacklog, backlogFlow + " › Estimación › Participantes"},
		{"Planning Poker desde el detalle", flowScreenModel("planning-poker", sectionPlanningPoker, sectionStepMembers, storyDetailFromBacklog), tabBacklog, backlogFlow + " › Estimación › Participantes"},
		{"Esfuerzo tareas desde el detalle", flowScreenModel("section", sectionEffort, sectionStepTasks, storyDetailFromBacklog), tabBacklog, backlogFlow + " › Esfuerzo › Tareas"},
		{"Esfuerzo integrantes desde el detalle", flowScreenModel("section", sectionEffort, sectionStepMembers, storyDetailFromBacklog), tabBacklog, backlogFlow + " › Esfuerzo › Integrante"},
		{"formulario de esfuerzo desde el detalle", flowScreenModel("effort-form", sectionEffort, sectionStepMembers, storyDetailFromBacklog), tabBacklog, backlogFlow + " › Esfuerzo › Registrar"},
		{"formulario de defecto desde el detalle", flowScreenModel("defect-create-form", "", sectionStepNone, storyDetailFromBacklog), tabBacklog, backlogFlow + " › Registrar defecto"},
		{"selector de Sprints del defecto desde el detalle", flowScreenModel("defect-sprint-select", "", sectionStepNone, storyDetailFromBacklog), tabBacklog, backlogFlow + " › Elegir Sprint de detección"},
		{"confirmación del defecto desde el detalle", flowScreenModel("defect-create-confirm", "", sectionStepNone, storyDetailFromBacklog), tabBacklog, backlogFlow + " › Confirmar registro"},
		{"Estimar desde un detalle de Sprints", flowScreenModel("section", sectionPlanningPoker, sectionStepMembers, storyDetailFromSprint), tabSprints, sprintFlow + " › Estimación › Participantes"},
		{"formulario de esfuerzo desde un detalle de Sprints", flowScreenModel("effort-form", sectionEffort, sectionStepMembers, storyDetailFromSprint), tabSprints, sprintFlow + " › Esfuerzo › Registrar"},
		{"formulario de defecto desde un detalle de Sprints", flowScreenModel("defect-create-form", "", sectionStepNone, storyDetailFromSprint), tabSprints, sprintFlow + " › Registrar defecto"},
		{"lista de defectos", flowScreenModel("defects", "", sectionStepNone, storyDetailFromBacklog), tabDefects, "Proyectos › Alpha › Defectos"},
		{"detalle de un defecto", flowScreenModel("defect-detail", "", sectionStepNone, storyDetailFromBacklog), tabDefects, "Proyectos › Alpha › Defectos › Detalle"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.m.currentTab(); got != tc.wantTab {
				t.Errorf("currentTab() = %v (%q), want %v (%q)", got, got.title(), tc.wantTab, tc.wantTab.title())
			}
			if got := strings.Join(tc.m.crumbParts(), " › "); got != tc.wantPath {
				t.Errorf("breadcrumb = %q, want %q", got, tc.wantPath)
			}
		})
	}
}

func TestCurrentTabOfStoryDetailFollowsOrigin(t *testing.T) {
	for _, tc := range []struct {
		origin storyDetailOrigin
		want   workspaceTab
	}{
		{storyDetailFromBacklog, tabBacklog},
		{storyDetailFromSprint, tabSprints},
	} {
		m := detailModel()
		m.storyDetail.origin = tc.origin

		if got := m.currentTab(); got != tc.want {
			t.Errorf("origin %v: currentTab() = %v, want %v", tc.origin, got, tc.want)
		}
		if m.tabRoot() {
			t.Error("the story detail must not be a tab root: Tab and the digits must not act there")
		}
	}
}
