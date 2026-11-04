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
	m := openSectionWithProject(t, server.URL, "p1", tabEstimation)
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

// pokerDetailServer serves the Planning Poker flow launched from the detail. The
// first GET /stories answers the sample backlog (no story points); every later
// one answers the reload, so a test controls what the detail rereads after the
// estimate was saved.
func pokerDetailServer(t *testing.T, requests *[]string, reload stub) *httptest.Server {
	t.Helper()
	listings := 0
	return pokerFlowServer(t, requests, func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/projects/p1/stories":
			listings++
			if listings == 1 {
				_, _ = w.Write([]byte(storiesBodyForSection(sectionStories())))
				return true
			}
			w.WriteHeader(reload.status)
			_, _ = w.Write([]byte(reload.body))
			return true
		case r.Method == http.MethodGet && r.URL.Path == "/projects/p1/members":
			_, _ = w.Write([]byte(`{"project_id":"p1","members":[{"id":"m1","project_id":"p1","full_name":"Ada Lovelace","email":null}]}`))
			return true
		case r.Method == http.MethodPut && r.URL.Path == "/projects/p1/stories/s1/story-points":
			_, _ = w.Write([]byte(`{"id":"s1","project_id":"p1","title":"Primera historia","story_points":8}`))
			return true
		}
		return false
	})
}

const (
	reloadedWithPoints = `{"project_id":"p1","stories":[{"id":"s1","project_id":"p1","title":"Primera historia","priority":"alta","status":"pendiente","story_points":8,"acceptance_criteria":[]},{"id":"s2","project_id":"p1","title":"Segunda historia","priority":"media","status":"en progreso"}]}`
	reloadedWithoutS1  = `{"project_id":"p1","stories":[{"id":"s2","project_id":"p1","title":"Segunda historia","priority":"media","status":"en progreso"}]}`
)

// pokerConfirmFromDetail launches Estimar from the detail of the first story and
// votes 8 until the explicit confirmation screen.
func pokerConfirmFromDetail(t *testing.T, baseURL string) tui.Model {
	t.Helper()
	return pokerConfirmFrom(t, openStoryDetailWithProject(t, baseURL, "p1", 0))
}

// pokerConfirmFrom does the same from a detail that is already open.
func pokerConfirmFrom(t *testing.T, detail tui.Model) tui.Model {
	t.Helper()
	m, membersCmd := apply(detail, keyMsg("p"))
	if membersCmd == nil {
		t.Fatal("p on the detail should load the participants")
	}
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
	if got := stripANSI(m.View()); !strings.Contains(got, "Confirmar Story Points 8") {
		t.Fatalf("expected the confirmation screen, got %q", got)
	}
	return m
}

// savedPoker confirms the estimate and applies the 200 of the PUT, returning the
// model back on the detail together with its pending backlog reload.
func savedPoker(t *testing.T, m tui.Model) (tui.Model, tea.Cmd) {
	t.Helper()
	m, saveCmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if saveCmd == nil {
		t.Fatal("explicit confirmation should issue the estimate request")
	}
	return apply(m, saveCmd())
}

// assertDetailIgnoresActions presses every action key of the detail and requires
// the screen to stay put without emitting a command.
func assertDetailIgnoresActions(t *testing.T, m tui.Model) {
	t.Helper()
	for name, key := range map[string]tea.KeyMsg{"p": keyMsg("p"), "t": keyMsg("t"), "d": keyMsg("d"), "Enter": {Type: tea.KeyEnter}} {
		if next, cmd := apply(m, key); next.Screen != "story-detail" || cmd != nil {
			t.Errorf("%s: Screen = %q, cmd nil = %v, want the detail ignoring it", name, next.Screen, cmd == nil)
		}
	}
}

func TestPokerSaveFromDetailReturnsToStoryDetailAndReloadsBacklog(t *testing.T) {
	var requests []string
	server := pokerDetailServer(t, &requests, stub{http.StatusOK, reloadedWithPoints})
	m := pokerConfirmFromDetail(t, server.URL)

	m, reloadCmd := savedPoker(t, m)

	if m.Screen != "story-detail" || reloadCmd == nil {
		t.Fatalf("Screen = %q, reload nil = %v, want the detail reloading the backlog", m.Screen, reloadCmd == nil)
	}
	if view := wideView(m); strings.Contains(view, "✓") {
		t.Errorf("estimating shows no toast: %q", view)
	}
	wantBefore := []string{"GET /projects/p1/stories", "GET /projects/p1/members", "PUT /projects/p1/stories/s1/story-points"}
	if !reflect.DeepEqual(requests, wantBefore) {
		t.Fatalf("requests before the reload = %v, want %v", requests, wantBefore)
	}
	m = run(t, m, reloadCmd)
	if want := append(wantBefore, "GET /projects/p1/stories"); !reflect.DeepEqual(requests, want) {
		t.Errorf("requests = %v, want the PUT followed by one GET /stories %v", requests, want)
	}
	if m.Screen != "story-detail" {
		t.Errorf("Screen = %q after the reload, want story-detail", m.Screen)
	}
}

func TestStoryDetailShowsStoryPointsReleidosAfterReload(t *testing.T) {
	var requests []string
	server := pokerDetailServer(t, &requests, stub{http.StatusOK, reloadedWithPoints})
	detail := openStoryDetailWithProject(t, server.URL, "p1", 0)
	if got := fieldValue(t, stripANSI(detail.View()), "Story points"); got != "—" {
		t.Fatalf("Story points before estimating = %q, want a dash", got)
	}
	m := pokerConfirmFrom(t, detail)

	m, reloadCmd := savedPoker(t, m)
	m = run(t, m, reloadCmd)

	if got := fieldValue(t, stripANSI(m.View()), "Story points"); got != "8" {
		t.Errorf("Story points = %q after the reload, want 8 read from the reloaded backlog", got)
	}
}

func TestStoryDetailShowsLoadingWhileBacklogReloads(t *testing.T) {
	var requests []string
	server := pokerDetailServer(t, &requests, stub{http.StatusOK, reloadedWithPoints})
	m := pokerConfirmFromDetail(t, server.URL)

	m, _ = savedPoker(t, m) // the reload is still in flight

	view := stripANSI(m.View())
	if !strings.Contains(view, "Cargando") || strings.Contains(view, "ya no está en el backlog") {
		t.Errorf("the detail should say it is loading, not that the story is gone:\n%s", view)
	}
	assertDetailIgnoresActions(t, m)
}

func TestStoryDetailShowsAPIErrorWhenReloadFails(t *testing.T) {
	var requests []string
	server := pokerDetailServer(t, &requests, stub{http.StatusInternalServerError, `{"error":"internal_error","message":"backlog down"}`})
	m := pokerConfirmFromDetail(t, server.URL)

	m, reloadCmd := savedPoker(t, m)
	m = run(t, m, reloadCmd)

	view := stripANSI(m.View())
	if m.Screen != "story-detail" || !strings.Contains(view, "Error HTTP 500") || strings.Contains(view, "ya no está en el backlog") {
		t.Fatalf("Screen = %q, want the detail showing the API error:\n%s", m.Screen, view)
	}
	assertDetailIgnoresActions(t, m)
	if m = press(m, tea.KeyEsc); m.Screen != "backlog" {
		t.Errorf("Esc: Screen = %q, want backlog", m.Screen)
	}
}

func TestStoryDetailReportsMissingStoryAfterReload(t *testing.T) {
	var requests []string
	server := pokerDetailServer(t, &requests, stub{http.StatusOK, reloadedWithoutS1})
	m := pokerConfirmFromDetail(t, server.URL)

	m, reloadCmd := savedPoker(t, m)
	m = run(t, m, reloadCmd)

	view := stripANSI(m.View())
	if m.Screen != "story-detail" || !strings.Contains(view, "ya no está en el backlog") || strings.Contains(view, "Cargando") {
		t.Fatalf("Screen = %q, want the detail reporting the missing story:\n%s", m.Screen, view)
	}
	assertDetailIgnoresActions(t, m)
	if m = press(m, tea.KeyEsc); m.Screen != "backlog" {
		t.Errorf("Esc: Screen = %q, want backlog", m.Screen)
	}
}

func pokerConfirmModel(t *testing.T, baseURL string) tui.Model {
	t.Helper()
	m := openSectionWithProject(t, baseURL, "p1", tabEstimation)
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
