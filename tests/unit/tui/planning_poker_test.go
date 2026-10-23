package tui_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

func TestPlanningPokerExplicitConfirmationSavesThenRefreshesBacklog(t *testing.T) {
	var requests []string
	saved := false
	server := pokerFlowServer(t, &requests, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/projects/p1/stories":
			if saved {
				_, _ = w.Write([]byte(`{"project_id":"p1","stories":[{"id":"s1","project_id":"p1","title":"Primera historia","priority":"alta","status":"pendiente","story_points":8,"acceptance_criteria":[]}]}`))
			} else {
				_, _ = w.Write([]byte(storiesBodyForSection(sectionStories())))
			}
			return true
		case r.Method == http.MethodGet && r.URL.Path == "/projects/p1/members":
			_, _ = w.Write([]byte(`{"project_id":"p1","members":[{"id":"m1","project_id":"p1","full_name":"Ada Lovelace","email":null}]}`))
			return true
		case r.Method == http.MethodPut && r.URL.Path == "/projects/p1/stories/s1/story-points":
			saved = true
			_, _ = w.Write([]byte(`{"id":"s1","project_id":"p1","title":"Primera historia","story_points":8}`))
			return true
		default:
			return false
		}
	})

	m := pokerConfirmModel(t, server.URL)
	if got := stripANSI(m.View()); !strings.Contains(got, "Confirmar Story Points 8") {
		t.Fatalf("expected explicit confirmation screen, got %q", got)
	}
	if !reflect.DeepEqual(requests, []string{"GET /projects/p1/stories", "GET /projects/p1/members"}) {
		t.Fatalf("requests before confirmation=%v; estimate must not be sent early", requests)
	}

	m, saveCmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if saveCmd == nil {
		t.Fatal("explicit confirmation should issue the estimate request")
	}
	m, refreshCmd := apply(m, saveCmd())
	if refreshCmd == nil {
		t.Fatalf("successful PUT should refresh the backlog; requests=%v view=%q", requests, stripANSI(m.View()))
	}
	if !reflect.DeepEqual(requests, []string{"GET /projects/p1/stories", "GET /projects/p1/members", "PUT /projects/p1/stories/s1/story-points"}) {
		t.Fatalf("requests before refresh=%v", requests)
	}
	m = run(t, m, refreshCmd)
	if !strings.Contains(stripANSI(m.View()), "puntos 8") {
		t.Fatalf("refreshed backlog does not show updated points: %q", stripANSI(m.View()))
	}
	want := []string{"GET /projects/p1/stories", "GET /projects/p1/members", "PUT /projects/p1/stories/s1/story-points", "GET /projects/p1/stories"}
	if !reflect.DeepEqual(requests, want) {
		t.Fatalf("requests=%v, want PUT then GET refresh %v", requests, want)
	}
}

func TestPlanningPokerEmptyMemberListHasClearStateAndCannotStartVoting(t *testing.T) {
	server := pokerFlowServer(t, new([]string), func(w http.ResponseWriter, r *http.Request) bool {
		switch r.URL.Path {
		case "/projects/p1/stories":
			_, _ = w.Write([]byte(storiesBodyForSection(sectionStories())))
			return true
		case "/projects/p1/members":
			_, _ = w.Write([]byte(`{"project_id":"p1","members":[]}`))
			return true
		default:
			return false
		}
	})
	m := openSectionWithProject(t, server.URL, "p1", menuPlanning)
	m, membersCmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(t, m, membersCmd)
	if got := stripANSI(m.View()); !strings.Contains(got, "No hay participantes disponibles.") {
		t.Fatalf("empty participant state is unclear: %q", got)
	}
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.Screen == "planning-poker" {
		t.Fatal("empty participant list must not start a vote or index an absent participant")
	}
}

func TestPlanningPokerDisplaysEstimateAPIErrorsWithoutRefreshingOrClaimingSuccess(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		status           int
	}{
		{name: "not found", status: http.StatusNotFound, body: `{"error":"story_not_found","message":"story missing"}`, want: "story missing"},
		{name: "conflict", status: http.StatusConflict, body: `{"error":"conflict","message":"estimate conflict"}`, want: "estimate conflict"},
		{name: "validation", status: http.StatusUnprocessableEntity, body: `{"error":"validation_failed","message":"invalid points","fields":{"story_points":"must be allowed"}}`, want: "must be allowed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests []string
			server := pokerFlowServer(t, &requests, func(w http.ResponseWriter, r *http.Request) bool {
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/projects/p1/stories":
					_, _ = w.Write([]byte(storiesBodyForSection(sectionStories())))
					return true
				case r.Method == http.MethodGet && r.URL.Path == "/projects/p1/members":
					_, _ = w.Write([]byte(`{"project_id":"p1","members":[{"id":"m1","project_id":"p1","full_name":"Ada Lovelace","email":null}]}`))
					return true
				case r.Method == http.MethodPut && r.URL.Path == "/projects/p1/stories/s1/story-points":
					w.WriteHeader(tc.status)
					_, _ = w.Write([]byte(tc.body))
					return true
				default:
					return false
				}
			})

			m := pokerConfirmModel(t, server.URL)
			m, saveCmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
			if saveCmd == nil {
				t.Fatal("explicit confirmation should issue the estimate request")
			}
			m = run(t, m, saveCmd)
			view := stripANSI(m.View())
			if !strings.Contains(view, tc.want) {
				t.Errorf("view=%q, want API detail %q", view, tc.want)
			}
			if m.Screen != "planning-poker" {
				t.Errorf("screen=%q after failed save, want planning-poker", m.Screen)
			}
			if strings.Contains(view, "puntos 8") {
				t.Errorf("failed save claims estimate was applied: %q", view)
			}
			want := []string{"GET /projects/p1/stories", "GET /projects/p1/members", "PUT /projects/p1/stories/s1/story-points"}
			if !reflect.DeepEqual(requests, want) {
				t.Errorf("failed save requests=%v, want no backlog refresh: %v", requests, want)
			}
		})
	}
}

func pokerConfirmModel(t *testing.T, baseURL string) tui.Model {
	t.Helper()
	m := openSectionWithProject(t, baseURL, "p1", menuPlanning)
	m, membersCmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(t, m, membersCmd)
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeySpace})
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'8'}})
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter}) // finish the vote and offer reveal
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter}) // reveal
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter}) // choose agreed estimate
	for i := 0; i < 5; i++ {                        // 8 points in the numeric scale
		m, _ = apply(m, tea.KeyMsg{Type: tea.KeyDown})
	}
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter}) // explicit confirmation screen
	return m
}

func pokerFlowServer(t *testing.T, requests *[]string, handler func(http.ResponseWriter, *http.Request) bool) *httptest.Server {
	t.Helper()
	return sprintServer(t, func(w http.ResponseWriter, r *http.Request) bool {
		*requests = append(*requests, r.Method+" "+r.URL.Path)
		return handler(w, r)
	})
}
