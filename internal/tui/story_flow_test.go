package tui

import (
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

func TestSelectProjectResetsStoryDetailAndReturnTo(t *testing.T) {
	m := detailModel()
	m.storyDetail.cursor = 1
	m.returnTo = returnStoryDetail

	next, _ := m.selectProject(ProjectSummary{ID: "p2", Name: "Beta", Status: "planned"})

	if next.storyDetail != (storyDetailState{}) || next.returnTo != returnDefault {
		t.Errorf("storyDetail = %+v, returnTo = %v, want the zero values after changing project", next.storyDetail, next.returnTo)
	}
	if next.activeProjectID != "p2" {
		t.Errorf("activeProjectID = %q, want p2", next.activeProjectID)
	}
}

func TestOpenTabResetsReturnTo(t *testing.T) {
	for _, tab := range workspaceTabs {
		t.Run(tab.title(), func(t *testing.T) {
			m := detailModel()
			m.returnTo = returnStoryDetail

			next, _ := m.openTab(tab)

			if next.returnTo != returnDefault {
				t.Errorf("returnTo = %v after opening %s, want returnDefault", next.returnTo, tab.title())
			}
		})
	}
}

// P and E on the backlog still start the old flows until the tabs are retired;
// a returnTo left over from a story flow must not make them go back to the detail.
func TestBacklogShortcutsResetAResidualReturnTo(t *testing.T) {
	for _, key := range []rune{'P', 'E'} {
		t.Run(string(key), func(t *testing.T) {
			m := detailModel()
			m.Screen = "backlog"
			m.returnTo = returnStoryDetail

			next, _ := m.updateBacklog(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})

			if next.Screen != "section" || next.returnTo != returnDefault {
				t.Errorf("Screen = %q, returnTo = %v, want a section that goes back to its own destination", next.Screen, next.returnTo)
			}
		})
	}
}

// flowScreenModel builds the model on screen with the flow state of a story flow
// of kind at step, started from origin.
func flowScreenModel(screen string, kind sectionKind, step sectionStep, origin returnTarget) Model {
	m := detailModel()
	m.Screen = screen
	m.section.kind, m.section.step = kind, step
	m.returnTo = origin
	return m
}

func TestCurrentTabAndCrumbPerScreenAndOrigin(t *testing.T) {
	const flow = "Proyectos › Alpha › Backlog › Detalle"
	for _, tc := range []struct {
		name     string
		m        Model
		wantTab  workspaceTab
		wantPath string
	}{
		{"detalle", flowScreenModel("story-detail", "", 0, returnDefault), tabBacklog, flow},
		{"Estimar participantes desde el detalle", flowScreenModel("section", sectionPlanningPoker, sectionStepMembers, returnStoryDetail), tabBacklog, flow + " › Estimación › Participantes"},
		{"Planning Poker desde el detalle", flowScreenModel("planning-poker", sectionPlanningPoker, sectionStepMembers, returnStoryDetail), tabBacklog, flow + " › Estimación › Participantes"},
		{"Esfuerzo tareas desde el detalle", flowScreenModel("section", sectionEffort, sectionStepTasks, returnStoryDetail), tabBacklog, flow + " › Esfuerzo › Tareas"},
		{"Esfuerzo integrantes desde el detalle", flowScreenModel("section", sectionEffort, sectionStepMembers, returnStoryDetail), tabBacklog, flow + " › Esfuerzo › Integrante"},
		{"formulario de esfuerzo desde el detalle", flowScreenModel("effort-form", sectionEffort, sectionStepMembers, returnStoryDetail), tabBacklog, flow + " › Esfuerzo › Registrar"},
		// [caracterización] The old tabs keep their own tab and breadcrumb until
		// they are retired; the origin is what tells the two apart.
		{"pestaña Estimación", flowScreenModel("section", sectionPlanningPoker, sectionStepPrimary, returnDefault), tabEstimation, "Proyectos › Alpha › Estimación"},
		{"pestaña Esfuerzo", flowScreenModel("section", sectionEffort, sectionStepPrimary, returnDefault), tabEffort, "Proyectos › Alpha › Esfuerzo"},
		{"tareas con la historia elegida desde el Backlog", flowScreenModel("section", sectionEffort, sectionStepTasks, returnDefault), tabEffort, "Proyectos › Alpha › Esfuerzo › Tareas"},
		{"Planning Poker de la pestaña", flowScreenModel("planning-poker", sectionPlanningPoker, sectionStepMembers, returnDefault), tabEstimation, "Proyectos › Alpha › Estimación › Participantes"},
		{"formulario de esfuerzo de la pestaña", flowScreenModel("effort-form", sectionEffort, sectionStepMembers, returnDefault), tabEffort, "Proyectos › Alpha › Esfuerzo › Registrar"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Assumption S1 (unconfirmed by the user): the flows that start from the
			// detail highlight Backlog; the rule lives only in currentTab.
			if got := tc.m.currentTab(); got != tc.wantTab {
				t.Errorf("currentTab() = %v (%q), want %v (%q)", got, got.title(), tc.wantTab, tc.wantTab.title())
			}
			if got := strings.Join(tc.m.crumbParts(), " › "); got != tc.wantPath {
				t.Errorf("breadcrumb = %q, want %q", got, tc.wantPath)
			}
		})
	}
}

func TestCurrentTabOfStoryDetailIsBacklog(t *testing.T) {
	m := detailModel()

	if got := m.currentTab(); got != tabBacklog {
		t.Fatalf("currentTab() = %v, want tabBacklog (%v)", got, tabBacklog)
	}
	if m.tabRoot() {
		t.Error("the story detail must not be a tab root: Tab and the digits must not act there")
	}
}
