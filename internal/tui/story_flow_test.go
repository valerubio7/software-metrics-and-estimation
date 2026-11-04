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

func TestCurrentTabOfStoryDetailIsBacklog(t *testing.T) {
	m := detailModel()

	if got := m.currentTab(); got != tabBacklog {
		t.Fatalf("currentTab() = %v, want tabBacklog (%v)", got, tabBacklog)
	}
	if m.tabRoot() {
		t.Error("the story detail must not be a tab root: Tab and the digits must not act there")
	}
}
