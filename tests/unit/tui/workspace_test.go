package tui_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

// workspaceServer answers GET /projects with projects and every other GET with
// an empty collection that each screen accepts, recording the requested paths.
type workspaceServer struct {
	*httptest.Server
	mu    sync.Mutex
	paths []string
}

func newWorkspaceServer(t *testing.T, projects []tui.ProjectSummary) *workspaceServer {
	t.Helper()
	ws := &workspaceServer{}
	ws.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws.mu.Lock()
		ws.paths = append(ws.paths, r.Method+" "+r.URL.Path)
		ws.mu.Unlock()
		path := r.URL.Path
		switch {
		case path == "/projects":
			_, _ = w.Write([]byte(projectsBody(t, projects)))
		case strings.HasSuffix(path, "/stories"):
			_, _ = w.Write([]byte(`{"stories":[]}`))
		case strings.HasSuffix(path, "/defects"):
			_, _ = w.Write([]byte(`{"defects":[]}`))
		case strings.HasSuffix(path, "/sprints/open"):
			_, _ = w.Write([]byte(`{"sprints":[]}`))
		case strings.HasSuffix(path, "/metrics"):
			_, _ = w.Write([]byte(`{}`))
		default:
			_, _ = w.Write([]byte(`{"id":"p1","name":"Alpha","status":"active"}`))
		}
	}))
	t.Cleanup(ws.Close)
	return ws
}

func (ws *workspaceServer) requested(want string) bool {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	for _, path := range ws.paths {
		if path == want {
			return true
		}
	}
	return false
}

// startTUI builds the model and applies the startup project load, as Bubble Tea
// does with Init.
func startTUI(t *testing.T, url string) tui.Model {
	t.Helper()
	m := tui.NewModel(url)
	return run(t, m, m.Init())
}

// openWorkspace starts the TUI with the sample projects and opens pid with
// Enter, applying the Dashboard load.
func openWorkspace(t *testing.T, url, pid string) tui.Model {
	t.Helper()
	m, cmd := pickProject(startTUI(t, url), pid)
	m = run(t, m, cmd)
	if m.Screen != "dashboard" {
		t.Fatalf("Screen = %q after opening %s, want dashboard", m.Screen, pid)
	}
	return m
}

// pickProject searches pid in the loaded browser and opens the first match.
func pickProject(m tui.Model, pid string) (tui.Model, tea.Cmd) {
	m, _ = apply(m, keyMsg("/"))
	m = typeText(m, pid)
	m = press(m, tea.KeyEnter) // stop searching, keep the filter
	return apply(m, tea.KeyMsg{Type: tea.KeyEnter})
}

// switchTab presses key on a workspace root and applies the tab load.
func switchTab(t *testing.T, m tui.Model, key tea.KeyMsg) tui.Model {
	t.Helper()
	m, cmd := apply(m, key)
	if cmd != nil {
		m = run(t, m, cmd)
	}
	return m
}

func TestStartupOpensTheProjectBrowser(t *testing.T) {
	server := newWorkspaceServer(t, sampleProjects())
	m := tui.NewModel(server.URL)
	if m.Screen != "project-picker" {
		t.Fatalf("initial Screen = %q, want project-picker", m.Screen)
	}
	m = run(t, m, m.Init())
	if !server.requested("GET /projects") {
		t.Fatal("startup did not request GET /projects")
	}
	m, _ = apply(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	view := stripANSI(m.View())
	for _, want := range []string{"Proyectos", "Alpha", "Beta"} {
		if !strings.Contains(view, want) {
			t.Errorf("startup view lacks %q:\n%s", want, view)
		}
	}
	for _, unwanted := range []string{"Panel de control", "Planning Poker", "Salir"} {
		if strings.Contains(view, unwanted) {
			t.Errorf("startup view still shows the old menu entry %q:\n%s", unwanted, view)
		}
	}
}

func TestStartupWithASingleProjectOpensItsDashboard(t *testing.T) {
	server := newWorkspaceServer(t, sampleProjects()[:1])
	m := startTUI(t, server.URL)
	if m.Screen != "dashboard" {
		t.Fatalf("Screen = %q with a single project, want dashboard", m.Screen)
	}
}

func TestLeavingTheSingleProjectDoesNotReopenIt(t *testing.T) {
	server := newWorkspaceServer(t, sampleProjects()[:1])
	m := startTUI(t, server.URL)
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.Screen != "project-picker" {
		t.Fatalf("Esc on the Dashboard: Screen = %q, want project-picker", m.Screen)
	}
	m = run(t, m, cmd)
	if m.Screen != "project-picker" {
		t.Fatalf("reloading the browser reopened the project: Screen = %q", m.Screen)
	}
}

func TestEnterOnAProjectOpensItsWorkspaceTabs(t *testing.T) {
	server := newWorkspaceServer(t, sampleProjects())
	m := openWorkspace(t, server.URL, "p1")
	m, _ = apply(m, tea.WindowSizeMsg{Width: 100, Height: 30})
	view := stripANSI(m.View())
	for _, tab := range []string{"Dashboard", "Backlog", "Sprints", "Defectos"} {
		if !strings.Contains(view, tab) {
			t.Errorf("workspace view lacks the %q tab:\n%s", tab, view)
		}
	}
	if !strings.Contains(view, "Proyectos › Alpha › Dashboard") {
		t.Errorf("breadcrumb does not start at Proyectos › Alpha › Dashboard:\n%s", view)
	}
}

// tabBarLine is the line of a rendered workspace view that holds the tab bar.
func tabBarLine(t *testing.T, view string) string {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "1 Dashboard") {
			return line
		}
	}
	t.Fatalf("view has no tab bar:\n%s", view)
	return ""
}

func TestTabBarShowsFourTabsWithoutEstimationOrEffort(t *testing.T) {
	server := newWorkspaceServer(t, sampleProjects())
	m := openWorkspace(t, server.URL, "p1")
	m, _ = apply(m, tea.WindowSizeMsg{Width: 100, Height: 30})

	bar := tabBarLine(t, stripANSI(m.View()))

	if !strings.Contains(bar, "1 Dashboard  2 Backlog  3 Sprints  4 Defectos") {
		t.Errorf("tab bar %q should list the four tabs in order with the digits 1 to 4", bar)
	}
	for _, unwanted := range []string{"Estimación", "Esfuerzo", "5 ", "6 "} {
		if strings.Contains(bar, unwanted) {
			t.Errorf("tab bar %q must not offer %q", bar, unwanted)
		}
	}
}

func TestTabsSwitchWithTabAndDigits(t *testing.T) {
	server := newWorkspaceServer(t, sampleProjects())
	m := openWorkspace(t, server.URL, "p1")

	m = switchTab(t, m, tea.KeyMsg{Type: tea.KeyTab})
	if m.Screen != "backlog" {
		t.Fatalf("Tab from Dashboard: Screen = %q, want backlog", m.Screen)
	}
	if !server.requested("GET /projects/p1/stories") {
		t.Error("opening the Backlog tab did not load the project stories")
	}
	m = switchTab(t, m, keyMsg("3"))
	if m.Screen != "sprints" {
		t.Fatalf("3: Screen = %q, want sprints", m.Screen)
	}
}

func TestDigitFourOpensDefects(t *testing.T) {
	server := newWorkspaceServer(t, sampleProjects())
	m := openWorkspace(t, server.URL, "p1")

	m = switchTab(t, m, keyMsg("4"))

	if m.Screen != "defects" {
		t.Fatalf("4: Screen = %q, want defects", m.Screen)
	}
	if !server.requested("GET /projects/p1/defects") {
		t.Error("opening the Defectos tab did not load the project defects")
	}
}

func TestDigitsFiveAndSixDoNothing(t *testing.T) {
	server := newWorkspaceServer(t, sampleProjects())
	m := openWorkspace(t, server.URL, "p1")

	for _, digit := range []string{"5", "6"} {
		next, cmd := apply(m, keyMsg(digit))
		if next.Screen != "dashboard" || cmd != nil {
			t.Errorf("%s: Screen = %q, command = %v, want the Dashboard unchanged and no request", digit, next.Screen, cmd != nil)
		}
	}
}

func TestTabWrapsFromDefectsToDashboard(t *testing.T) {
	server := newWorkspaceServer(t, sampleProjects())
	m := switchTab(t, openWorkspace(t, server.URL, "p1"), keyMsg("4"))
	if m.Screen != "defects" {
		t.Fatalf("4: Screen = %q, want defects", m.Screen)
	}

	m = switchTab(t, m, tea.KeyMsg{Type: tea.KeyTab})

	if m.Screen != "dashboard" {
		t.Fatalf("Tab from Defectos: Screen = %q, want dashboard", m.Screen)
	}
}

func TestShiftTabFromDashboardOpensDefects(t *testing.T) {
	server := newWorkspaceServer(t, sampleProjects())
	m := openWorkspace(t, server.URL, "p1")
	m = switchTab(t, m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.Screen != "defects" {
		t.Fatalf("Shift+Tab from Dashboard: Screen = %q, want defects", m.Screen)
	}
}

func TestShiftTabFromDefectsOpensSprints(t *testing.T) {
	server := newWorkspaceServer(t, sampleProjects())
	m := switchTab(t, openWorkspace(t, server.URL, "p1"), keyMsg("4"))
	if m.Screen != "defects" {
		t.Fatalf("4: Screen = %q, want defects", m.Screen)
	}

	m = switchTab(t, m, tea.KeyMsg{Type: tea.KeyShiftTab})

	if m.Screen != "sprints" {
		t.Fatalf("Shift+Tab from Defectos: Screen = %q, want sprints", m.Screen)
	}
}

func TestDigitsAreTypedIntoAFilter(t *testing.T) {
	server := newWorkspaceServer(t, sampleProjects())
	m := switchTab(t, openWorkspace(t, server.URL, "p1"), keyMsg("2"))
	if m.Screen != "backlog" {
		t.Fatalf("2: Screen = %q, want backlog", m.Screen)
	}
	m, _ = apply(m, keyMsg("/"))
	m, cmd := apply(m, keyMsg("3"))
	if m.Screen != "backlog" || cmd != nil {
		t.Fatalf("3 inside the filter changed the tab: Screen = %q", m.Screen)
	}
	if view := stripANSI(m.View()); !strings.Contains(view, "3") {
		t.Errorf("the digit was not typed into the filter:\n%s", view)
	}
}

func TestEscOnATabRootReturnsToProjects(t *testing.T) {
	server := newWorkspaceServer(t, sampleProjects())
	m := switchTab(t, openWorkspace(t, server.URL, "p1"), keyMsg("2"))
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.Screen != "project-picker" || cmd == nil {
		t.Fatalf("Esc on the Backlog root: Screen = %q (cmd %v), want project-picker reloading", m.Screen, cmd != nil)
	}
}

func TestEscInsideAFlowStaysInTheWorkspace(t *testing.T) {
	server := newWorkspaceServer(t, sampleProjects())
	m := switchTab(t, openWorkspace(t, server.URL, "p1"), keyMsg("2"))
	m = typeText(m, "n")
	if m.Screen != "backlog-form" {
		t.Fatalf("n: Screen = %q, want backlog-form", m.Screen)
	}
	m = press(m, tea.KeyEsc)
	if m.Screen != "backlog" {
		t.Fatalf("Esc in the story form: Screen = %q, want backlog", m.Screen)
	}
}

func TestChangingProjectRetargetsEveryTab(t *testing.T) {
	server := newWorkspaceServer(t, sampleProjects())
	m := openWorkspace(t, server.URL, "p1")
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEsc})
	m, cmd = confirmProjectInPicker(t, m, cmd, "p2")
	m = run(t, m, cmd)
	m = switchTab(t, m, keyMsg("2"))
	if !server.requested("GET /projects/p2/stories") {
		t.Error("the Backlog tab did not load the stories of the new project")
	}
	m, _ = apply(m, tea.WindowSizeMsg{Width: 100, Height: 30})
	if view := stripANSI(m.View()); !strings.Contains(view, "Proyectos › Beta › Backlog") {
		t.Errorf("breadcrumb does not show the new project:\n%s", view)
	}
}

func TestBrowserEscWithoutFilterStaysAndQQuits(t *testing.T) {
	server := newWorkspaceServer(t, sampleProjects())
	m := startTUI(t, server.URL)
	m = press(m, tea.KeyEsc)
	if m.Screen != "project-picker" {
		t.Fatalf("Esc without a filter: Screen = %q, want project-picker", m.Screen)
	}
	if _, cmd := apply(m, keyMsg("q")); !isQuit(cmd) {
		t.Error("q on the project browser did not quit")
	}
}

func TestOldProjectShortcutsAreGone(t *testing.T) {
	server := newWorkspaceServer(t, sampleProjects())
	m := switchTab(t, openWorkspace(t, server.URL, "p1"), keyMsg("2"))
	m, _ = apply(m, keyMsg("p"))
	if m.Screen != "backlog" {
		t.Errorf("p on the Backlog: Screen = %q, want backlog (search, not the picker)", m.Screen)
	}
	m = press(m, tea.KeyEsc)
	m, _ = apply(m, keyMsg("c"))
	if m.Screen != "backlog" {
		t.Errorf("c on the Backlog: Screen = %q, want backlog", m.Screen)
	}
}
