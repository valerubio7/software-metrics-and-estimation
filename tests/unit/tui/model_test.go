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

func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// spaceKey is the message Bubble Tea v1.3.10 emits for the space bar.
func spaceKey() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
}

// press applies a non-rune key to the model.
func press(m tui.Model, keyType tea.KeyType) tui.Model {
	updated, _ := m.Update(tea.KeyMsg{Type: keyType})
	return updated.(tui.Model)
}

// typeText types s rune by rune, sending the space bar as tea.KeySpace like a real terminal.
func typeText(m tui.Model, s string) tui.Model {
	for _, r := range s {
		var msg tea.KeyMsg
		if r == ' ' {
			msg = spaceKey()
		} else {
			msg = keyMsg(string(r))
		}
		updated, _ := m.Update(msg)
		m = updated.(tui.Model)
	}
	return m
}

// isQuit reports whether cmd produces tea.QuitMsg.
func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

// fieldValue extracts the text rendered after "<label>: " on its line of the view.
// The project-id prompt draws a trailing "_" cursor, which is not part of the value.
func fieldValue(t *testing.T, view, label string) string {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if _, value, found := strings.Cut(line, label+": "); found {
			value = strings.TrimRight(value, "\r ")
			value = strings.TrimSuffix(value, "│")
			value = strings.TrimRight(value, " ")
			value = strings.TrimSuffix(value, "_")
			return strings.TrimRight(value, " ")
		}
	}
	t.Fatalf("label %q not found in view %q", label, view)
	return ""
}

func toProjectID(m tui.Model) tui.Model {
	m = press(m, tea.KeyEnter) // menu -> projects
	m = press(m, tea.KeyDown)
	m = press(m, tea.KeyDown)
	return press(m, tea.KeyEnter) // "Consultar estado" -> project-id
}

func toCreateProject(m tui.Model) tui.Model {
	m = press(m, tea.KeyEnter) // menu -> projects
	return press(m, tea.KeyEnter)
}

func toEditProject(m tui.Model, id string) tui.Model {
	m = press(m, tea.KeyEnter) // menu -> projects
	m = press(m, tea.KeyDown)
	m = press(m, tea.KeyDown)
	m = press(m, tea.KeyEnter) // "Modificar proyecto" -> project-id
	m = typeText(m, id)
	return press(m, tea.KeyEnter) // -> edit-project
}

// projectFieldCase describes a project text field reached from the menu.
type projectFieldCase struct {
	screen string
	label  string
	open   func(tui.Model) tui.Model
}

var projectTextFields = []projectFieldCase{
	{"project-id", "ID", toProjectID},
	{"create-project", "Nombre", toCreateProject},
}

// submitProject fills the create form and returns the model and the API command.
func submitProject(t *testing.T, m tui.Model, name string) (tui.Model, tea.Cmd) {
	t.Helper()
	m = toCreateProject(m)
	m = typeText(m, name)
	m = press(m, tea.KeyTab)
	m = typeText(m, "2026-10-01")
	m = press(m, tea.KeyTab)
	m = typeText(m, "2026-12-01")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("submitting the project form should issue an API command")
	}
	return updated.(tui.Model), cmd
}

// capturingProjectServer records the raw body of every POST /projects.
func capturingProjectServer(t *testing.T) (*httptest.Server, *[][]byte) {
	t.Helper()
	var bodies [][]byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"p1","name":"ok","status":"active"}`))
	}))
	t.Cleanup(server.Close)
	return server, &bodies
}

func TestMenuShowsFourOptions(t *testing.T) {
	m := tui.NewModel("http://localhost:8080")
	view := m.View()

	for _, want := range []string{"proyectos", "backlog", "sprints", "salir"} {
		if !strings.Contains(strings.ToLower(view), want) {
			t.Errorf("View() = %q, want option %q", view, want)
		}
	}
	if strings.Contains(view, "Ayuda") {
		t.Errorf("View() = %q, want no Ayuda menu option", view)
	}
}

func TestCursorMovesDownAndUp(t *testing.T) {
	m := tui.NewModel("http://localhost:8080")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(tui.Model)
	if m.Cursor != 1 {
		t.Fatalf("Cursor = %d, want 1 after down", m.Cursor)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(tui.Model)
	if m.Cursor != 0 {
		t.Fatalf("Cursor = %d, want 0 after up", m.Cursor)
	}
}

func TestEnterProjectsAndBack(t *testing.T) {
	m := tui.NewModel("http://localhost:8080")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tui.Model)
	if m.Screen == "menu" {
		t.Fatalf("Screen = menu after enter, want placeholder")
	}
	if !strings.Contains(m.View(), "Crear proyecto") {
		t.Errorf("project View() = %q, want project actions", m.View())
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(tui.Model)
	if m.Screen != "menu" {
		t.Errorf("Screen = %q after esc, want menu", m.Screen)
	}
}

func TestProjectScreenOffersCreateEditAndStatus(t *testing.T) {
	m := tui.NewModel("http://localhost:8080")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tui.Model)
	view := strings.ToLower(m.View())
	for _, want := range []string{"crear", "modificar", "estado"} {
		if !strings.Contains(view, want) {
			t.Errorf("project View() = %q, want %q", view, want)
		}
	}
}

func TestProjectAPIErrorIsShownWithoutSuccess(t *testing.T) {
	m := tui.NewModel("http://localhost:8080")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tui.Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tui.Model)
	for i, value := range []string{"Alpha", "2026-10-01", "2026-12-01"} {
		for _, r := range value {
			updated, _ = m.Update(keyMsg(string(r)))
			m = updated.(tui.Model)
		}
		if i < 2 {
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
			m = updated.(tui.Model)
		}
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tui.Model)
	if cmd == nil {
		t.Fatal("submitting form should issue API command")
	}
	result := tui.ProjectResultMsg{Response: tui.APIResponse{StatusCode: 422, Body: []byte(`{"error":"validation_failed","message":"invalid dates"}`)}}
	updated, _ = m.Update(result)
	m = updated.(tui.Model)
	view := m.View()
	if !strings.Contains(view, "422") || !strings.Contains(view, "invalid dates") || strings.Contains(view, "creado correctamente") {
		t.Errorf("error view = %q", view)
	}
}

func TestProjectNotFoundIsExplicitAndPreservesAPIDetail(t *testing.T) {
	m := tui.NewModel("http://localhost:8080")
	updated, _ := m.Update(tui.ProjectResultMsg{Response: tui.APIResponse{StatusCode: 404, Body: []byte(`{"error":"project_not_found","message":"project not found"}`)}})
	m = updated.(tui.Model)
	view := m.View()
	if !strings.Contains(view, "no existe") || !strings.Contains(view, "project_not_found") {
		t.Errorf("404 view = %q, want explicit not-found message and API detail", view)
	}
}

func TestProjectNavigationPreservesOtherSectionsAndQuit(t *testing.T) {
	m := tui.NewModel("http://localhost:8080")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.(tui.Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tui.Model)
	if m.Screen != "project-picker" {
		t.Fatalf("screen = %q, want project-picker", m.Screen)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(tui.Model)
	if m.Screen != "menu" {
		t.Fatalf("screen after escape = %q, want menu", m.Screen)
	}
	_, cmd := m.Update(keyMsg("q"))
	if cmd == nil {
		t.Fatal("q should quit from menu")
	}
}

func TestQuitOnQAndCtrlC(t *testing.T) {
	for _, msg := range []tea.Msg{keyMsg("q"), tea.KeyMsg{Type: tea.KeyCtrlC}} {
		m := tui.NewModel("http://localhost:8080")
		_, cmd := m.Update(msg)
		if cmd == nil {
			t.Fatalf("Update(%v) cmd = nil, want quit command", msg)
		}
		var quitMsg tea.QuitMsg
		if got := cmd(); got != quitMsg {
			_ = got
		}
	}
}

func TestQuitOnQDoesNotExitInProjectTextFields(t *testing.T) {
	for _, field := range projectTextFields {
		for _, char := range []string{"q", "Q"} {
			t.Run(field.screen+"/"+char, func(t *testing.T) {
				m := field.open(tui.NewModel("http://localhost:8080"))
				updated, cmd := m.Update(keyMsg(char))
				m = updated.(tui.Model)
				if isQuit(cmd) {
					t.Fatalf("%q quit the TUI from %s, want it typed into the field", char, field.screen)
				}
				if m.Screen != field.screen {
					t.Errorf("Screen = %q after %q, want %q", m.Screen, char, field.screen)
				}
				if got := fieldValue(t, m.View(), field.label); got != char {
					t.Errorf("field %q = %q, want %q", field.label, got, char)
				}
			})
		}
	}
}

func TestCtrlCQuitsFromProjectForms(t *testing.T) {
	screens := []struct {
		screen string
		open   func(tui.Model) tui.Model
	}{
		{"project-id", toProjectID},
		{"create-project", toCreateProject},
		{"edit-project", func(m tui.Model) tui.Model { return toEditProject(m, "p1") }},
	}
	for _, tc := range screens {
		t.Run(tc.screen, func(t *testing.T) {
			m := tc.open(tui.NewModel("http://localhost:8080"))
			if m.Screen != tc.screen {
				t.Fatalf("Screen = %q, want %q", m.Screen, tc.screen)
			}
			_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
			if !isQuit(cmd) {
				t.Errorf("Ctrl+C from %s did not quit", tc.screen)
			}
		})
	}
}

func TestQuitOnQFromMenuAndProjectLists(t *testing.T) {
	screens := []struct {
		screen string
		open   func(tui.Model) tui.Model
	}{
		{"menu", func(m tui.Model) tui.Model { return m }},
		{"projects", func(m tui.Model) tui.Model { return press(m, tea.KeyEnter) }},
		{"project-result", func(m tui.Model) tui.Model {
			updated, _ := m.Update(tui.ProjectResultMsg{Err: errors.New("connection refused")})
			return updated.(tui.Model)
		}},
	}
	for _, tc := range screens {
		for _, char := range []string{"q", "Q"} {
			t.Run(tc.screen+"/"+char, func(t *testing.T) {
				m := tc.open(tui.NewModel("http://localhost:8080"))
				if m.Screen != tc.screen {
					t.Fatalf("Screen = %q, want %q", m.Screen, tc.screen)
				}
				_, cmd := m.Update(keyMsg(char))
				if !isQuit(cmd) {
					t.Errorf("%q from %s did not quit", char, tc.screen)
				}
			})
		}
	}
}

func TestSpaceIsAcceptedInProjectTextFields(t *testing.T) {
	for _, field := range projectTextFields {
		t.Run(field.screen, func(t *testing.T) {
			m := field.open(tui.NewModel("http://localhost:8080"))
			m = typeText(m, "a b")
			if got := fieldValue(t, m.View(), field.label); got != "a b" {
				t.Errorf("field %q = %q, want %q", field.label, got, "a b")
			}
		})
	}
}

func TestSpaceInProjectNameIsSentToAPI(t *testing.T) {
	server, bodies := capturingProjectServer(t)
	_, cmd := submitProject(t, tui.NewModel(server.URL), "Proyecto Alfa")
	cmd()
	if len(*bodies) != 1 {
		t.Fatalf("POST /projects count = %d, want 1", len(*bodies))
	}
	var sent map[string]string
	if err := json.Unmarshal((*bodies)[0], &sent); err != nil {
		t.Fatal(err)
	}
	if sent["name"] != "Proyecto Alfa" {
		t.Errorf("name = %q, want %q", sent["name"], "Proyecto Alfa")
	}
}

func TestBackspaceRemovesWholeRuneInProjectFields(t *testing.T) {
	steps := []struct {
		name  string
		typed string
		want  []string // value after each backspace
	}{
		{"accented-last-rune", "Canción", []string{"Canció"}},
		{"accented-final-rune", "Canció", []string{"Canci"}},
		{"multibyte-until-empty", "añá", []string{"añ", "a", ""}},
	}
	for _, field := range projectTextFields {
		for _, step := range steps {
			t.Run(field.screen+"/"+step.name, func(t *testing.T) {
				m := field.open(tui.NewModel("http://localhost:8080"))
				m = typeText(m, step.typed)
				for i, want := range step.want {
					m = press(m, tea.KeyBackspace)
					got := fieldValue(t, m.View(), field.label)
					if !utf8.ValidString(got) || strings.Contains(got, "�") {
						t.Fatalf("backspace %d left invalid UTF-8: %q", i+1, got)
					}
					if got != want {
						t.Errorf("backspace %d: field = %q, want %q", i+1, got, want)
					}
				}
			})
		}
	}
}

func TestBackspaceOnEmptyFieldIsNoop(t *testing.T) {
	for _, field := range projectTextFields {
		t.Run(field.screen, func(t *testing.T) {
			m := field.open(tui.NewModel("http://localhost:8080"))
			m = press(m, tea.KeyBackspace)
			if m.Screen != field.screen {
				t.Errorf("Screen = %q, want %q", m.Screen, field.screen)
			}
			if got := fieldValue(t, m.View(), field.label); got != "" {
				t.Errorf("field %q = %q, want empty", field.label, got)
			}
			m = typeText(m, "x")
			if got := fieldValue(t, m.View(), field.label); got != "x" {
				t.Errorf("field %q after typing = %q, want %q", field.label, got, "x")
			}
		})
	}
}

func TestBackspaceThenTypeSendsValidUTF8Name(t *testing.T) {
	server, bodies := capturingProjectServer(t)
	m := tui.NewModel(server.URL)
	m = toCreateProject(m)
	m = typeText(m, "Gestión añ")
	m = press(m, tea.KeyBackspace)
	m = typeText(m, "o")
	m = press(m, tea.KeyTab)
	m = typeText(m, "2026-10-01")
	m = press(m, tea.KeyTab)
	m = typeText(m, "2026-12-01")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("submitting the project form should issue an API command")
	}
	cmd()
	if len(*bodies) != 1 {
		t.Fatalf("POST /projects count = %d, want 1", len(*bodies))
	}
	if !utf8.Valid((*bodies)[0]) {
		t.Errorf("request body is not valid UTF-8: %q", (*bodies)[0])
	}
	var sent map[string]string
	if err := json.Unmarshal((*bodies)[0], &sent); err != nil {
		t.Fatal(err)
	}
	if sent["name"] != "Gestión ao" {
		t.Errorf("name = %q, want %q", sent["name"], "Gestión ao")
	}
}
