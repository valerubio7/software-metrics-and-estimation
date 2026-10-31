package tui_test

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

var effortFormDate = regexp.MustCompile(`Fecha:\s+(\d{4}|AAAA)\s+(\d{2}|MM)\s+(\d{2}|DD)`)

func TestEffortFormOpensWithUTCDateFromInjectedClock(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)

	if m.Screen != "effort-form" {
		t.Fatalf("Screen = %q, want effort-form", m.Screen)
	}
	view := stripANSI(m.View())
	date := effortFormDate.FindStringSubmatch(view)
	if date == nil || date[1] != "2026" || date[2] != "11" || date[3] != "01" {
		t.Fatalf("the date should open as the UTC day 2026-11-01, got %v in %q", date, view)
	}
	if line := focusedLine(view); !strings.Contains(line, "Fecha") {
		t.Fatalf("the focus should start on Fecha: %q", line)
	}
	for _, label := range []string{"Actividad", "Horas"} {
		if value := fieldValue(t, view, label); value != "" {
			t.Errorf("%s should start empty, got %q", label, value)
		}
	}
	for _, want := range []string{"Historia: Primera historia", "Tarea: Diseñar", "Integrante: Ada Lovelace"} {
		if !strings.Contains(view, want) {
			t.Errorf("the form should show the read-only %q: %q", want, view)
		}
	}
	for _, request := range rec.all() {
		if request.Method != http.MethodGet || strings.Contains(request.Path, "effort-records") {
			t.Errorf("opening the form emitted %s %s, want only navigation GETs", request.Method, request.Path)
		}
	}
}

func TestEffortFormShowsWholeStoryLabel(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 0, 0)

	if view := stripANSI(m.View()); m.Screen != "effort-form" || !strings.Contains(view, "Tarea: toda la historia") {
		t.Fatalf("the whole-story choice should read 'Tarea: toda la historia' (screen %q): %q", m.Screen, view)
	}
}

func TestEffortMemberSingleSelectionOpensFormForSecondMember(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openEffortMembers(t, server.URL, 0, 1).WithClock(fixedClock)

	m = press(m, tea.KeyDown)
	m, _ = apply(m, spaceKey())
	if view := stripANSI(m.View()); strings.Contains(view, "[x]") || strings.Contains(view, "[ ]") {
		t.Fatalf("Space must not mark a member: %q", view)
	}
	m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if view := stripANSI(m.View()); m.Screen != "effort-form" || !strings.Contains(view, "Integrante: Grace Hopper") {
		t.Fatalf("Enter on the second member should open the form for Grace Hopper (screen %q): %q", m.Screen, view)
	}
}

// effortDate reads the three rendered date segments (hints included).
func effortDate(t *testing.T, view string) [3]string {
	t.Helper()
	match := effortFormDate.FindStringSubmatch(view)
	if match == nil {
		t.Fatalf("the form shows no date row: %q", view)
	}
	return [3]string{match[1], match[2], match[3]}
}

// assertFocus fails unless the focused row is the one labelled label.
func assertFocus(t *testing.T, m interface{ View() string }, label string) {
	t.Helper()
	if line := focusedLine(stripANSI(m.View())); !strings.Contains(line, label+":") {
		t.Fatalf("focus should be on %s, got %q", label, line)
	}
}

func TestEffortFormEditsDateBySegments(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)

	m = press(m, tea.KeyRight)
	m = press(m, tea.KeyRight) // day segment
	for i := 0; i < 7; i++ {   // "1" + "11" + "2026" digits, stepping back through the segments
		m = press(m, tea.KeyBackspace)
	}
	if got := effortDate(t, stripANSI(m.View())); got != [3]string{"AAAA", "MM", "DD"} {
		t.Fatalf("backspace should erase the whole date, got %v", got)
	}

	m = typeText(m, "2026")
	m = typeText(m, "1")
	m = press(m, tea.KeyBackspace) // lands on the month only if the focus advanced after the year
	if got := effortDate(t, stripANSI(m.View())); got != [3]string{"2026", "MM", "DD"} {
		t.Fatalf("the focus should have moved to the month after the year was complete, got %v", got)
	}

	m = typeText(m, "1005")
	if got := effortDate(t, stripANSI(m.View())); got != [3]string{"2026", "10", "05"} {
		t.Fatalf("typing 20261005 should fill year, month and day, got %v", got)
	}
}

func TestEffortFormEnterAdvancesWithoutRequest(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)
	before := len(rec.all())

	assertFocus(t, m, "Fecha")
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("Enter on Fecha must not emit a command")
	}
	assertFocus(t, m, "Actividad")
	m, cmd = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("Enter on Actividad must not emit a command")
	}
	assertFocus(t, m, "Horas")
	if after := len(rec.all()); after != before {
		t.Fatalf("Enter emitted %d requests, want none", after-before)
	}
}

func TestEffortFormTabAndArrowsWrapAround(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)

	steps := []struct {
		name string
		key  tea.KeyType
		want string
	}{
		{"Tab", tea.KeyTab, "Actividad"},
		{"Tab", tea.KeyTab, "Horas"},
		{"Tab wraps to the first field", tea.KeyTab, "Fecha"},
		{"Shift+Tab wraps to the last field", tea.KeyShiftTab, "Horas"},
		{"Up", tea.KeyUp, "Actividad"},
		{"Down", tea.KeyDown, "Horas"},
		{"Down wraps to the first field", tea.KeyDown, "Fecha"},
		{"Up wraps to the last field", tea.KeyUp, "Horas"},
	}
	for _, step := range steps {
		m = press(m, step.key)
		if line := focusedLine(stripANSI(m.View())); !strings.Contains(line, step.want+":") {
			t.Fatalf("after %s the focus should be on %s, got %q", step.name, step.want, line)
		}
	}
}

func TestEffortFormTextFieldsAcceptSpacesAndRuneBackspace(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)

	m = press(m, tea.KeyTab) // Actividad
	m = typeText(m, "Revisión de diseño")
	if got := fieldValue(t, stripANSI(m.View()), "Actividad"); got != "Revisión de diseño" {
		t.Fatalf("Actividad = %q, want the typed text with its spaces", got)
	}
	m = press(m, tea.KeyBackspace)
	got := fieldValue(t, stripANSI(m.View()), "Actividad")
	if got != "Revisión de diseñ" || !utf8.ValidString(got) {
		t.Fatalf("Actividad = %q, want 'Revisión de diseñ' as valid UTF-8", got)
	}
}

func TestEffortFormTreatsGlobalKeysAsTextCharacters(t *testing.T) {
	for _, field := range []struct {
		label string
		tabs  int
	}{{"Actividad", 1}, {"Horas", 2}} {
		t.Run(field.label, func(t *testing.T) {
			server, rec := effortFlowServer(t, effortFlowOptions{})
			m := openEffortFormFor(t, server.URL, 0, 1, 0)
			for i := 0; i < field.tabs; i++ {
				m = press(m, tea.KeyTab)
			}
			before := len(rec.all())

			for _, r := range "qQ?pr" {
				var cmd tea.Cmd
				m, cmd = apply(m, keyMsg(string(r)))
				if isQuit(cmd) {
					t.Fatalf("%q must not quit the TUI inside the form", r)
				}
			}
			view := stripANSI(m.View())
			if got := fieldValue(t, view, field.label); got != "qQ?pr" {
				t.Fatalf("%s = %q, want qQ?pr", field.label, got)
			}
			if m.Screen != "effort-form" || strings.Contains(view, "Atajos de teclado") {
				t.Fatalf("the form must stay open without help (screen %q): %q", m.Screen, view)
			}
			if after := len(rec.all()); after != before {
				t.Fatalf("typing emitted %d requests, want none", after-before)
			}
			if _, cmd := apply(m, tea.KeyMsg{Type: tea.KeyCtrlC}); !isQuit(cmd) {
				t.Fatal("Ctrl+C must still quit")
			}
		})
	}
}

func TestEffortFormFooterOnlyOffersFormKeys(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)

	view := strings.Join(strings.Fields(stripANSI(m.View())), " ")
	for _, want := range []string{"Tab/↑/↓ campo", "Enter siguiente/enviar", "Esc volver"} {
		if !strings.Contains(view, want) {
			t.Errorf("footer lacks %q: %q", want, view)
		}
	}
	for _, banned := range []string{"q salir", "p proyecto", "r refrescar", "? ayuda"} {
		if strings.Contains(view, banned) {
			t.Errorf("footer must not offer %q: %q", banned, view)
		}
	}
}

func TestEffortFormCrumbAndTitle(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)

	view := stripANSI(m.View())
	for _, want := range []string{"Esfuerzo › Registrar", "Proyecto: Alpha", "Registrar esfuerzo"} {
		if !strings.Contains(view, want) {
			t.Errorf("form lacks %q: %q", want, view)
		}
	}
}

// valueColumn returns the display column where the value after "label:" starts.
func valueColumn(t *testing.T, view, label string) int {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if before, after, found := strings.Cut(line, label+":"); found {
			rest := strings.TrimLeft(after, " ")
			return utf8.RuneCountInString(before) + utf8.RuneCountInString(label+":") + utf8.RuneCountInString(after) - utf8.RuneCountInString(rest)
		}
	}
	t.Fatalf("label %q not found in view %q", label, view)
	return 0
}

func TestEffortFormAlignsTheValueColumn(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)
	m = press(m, tea.KeyTab)
	m = typeText(m, "x")
	m = press(m, tea.KeyTab)
	m = typeText(m, "1")

	view := stripANSI(m.View())
	activity, hours, date := valueColumn(t, view, "Actividad"), valueColumn(t, view, "Horas"), valueColumn(t, view, "Fecha")
	if activity != hours || hours+1 != date {
		t.Fatalf("value columns differ: Fecha %d (segments start one cell later), Actividad %d, Horas %d: %q", date, activity, hours, view)
	}
}

// fillEffortForm replaces the date the form opened with by the given year,
// month and day segments (an empty segment stays blank), then types the
// activity and the hours. The focus ends on Horas, ready to submit.
func fillEffortForm(m tui.Model, date [3]string, description, hours string) tui.Model {
	m = press(press(m, tea.KeyRight), tea.KeyRight)
	for i := 0; i < 12; i++ { // more than the 7 digits of the opening date
		m = press(m, tea.KeyBackspace)
	}
	for i, segment := range date {
		m = typeText(m, segment)
		if widths := [3]int{4, 2, 2}; len(segment) < widths[i] {
			m = press(m, tea.KeyRight) // a full segment already moved the focus on
		}
	}
	m = press(m, tea.KeyTab)
	m = typeText(m, description)
	m = press(m, tea.KeyTab)
	return typeText(m, hours)
}

// effortPosts returns the POST requests recorded so far.
func effortPosts(rec *recorder) []recordedRequest {
	var posts []recordedRequest
	for _, request := range rec.all() {
		if request.Method == http.MethodPost {
			posts = append(posts, request)
		}
	}
	return posts
}

// submitEffort presses Enter on the focused Horas field, runs the command it
// returns and reports the model after the response with the single new POST.
func submitEffort(t *testing.T, m tui.Model, rec *recorder) (tui.Model, recordedRequest) {
	t.Helper()
	before := len(effortPosts(rec))
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter on Horas should submit the effort")
	}
	m = run(t, m, cmd)
	posts := effortPosts(rec)
	if len(posts) != before+1 {
		t.Fatalf("POSTs = %+v, want exactly 1 new one after the %d already sent", posts, before)
	}
	return m, posts[before]
}

// assertEffortBody compares the request body key by key with its raw JSON value.
func assertEffortBody(t *testing.T, body []byte, want map[string]string) {
	t.Helper()
	keys := bodyKeys(t, body)
	if len(keys) != len(want) {
		t.Fatalf("body keys = %v, want %d keys %v: %s", sortedKeys(keys), len(want), want, body)
	}
	for name, raw := range want {
		if got, ok := keys[name]; !ok || string(got) != raw {
			t.Errorf("body[%q] = %s, want %s (body %s)", name, got, raw, body)
		}
	}
}

func TestEffortSubmitSendsExactBodyForTask(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)
	m = fillEffortForm(m, [3]string{"2026", "10", "05"}, "Implementar login", "3.5")

	_, post := submitEffort(t, m, rec)
	if post.Path != "/projects/p1/stories/s1/effort-records" {
		t.Errorf("path = %q, want /projects/p1/stories/s1/effort-records", post.Path)
	}
	assertEffortBody(t, post.Body, map[string]string{
		"member_id": `"m1"`, "task_id": `"t1"`, "work_date": `"2026-10-05"`,
		"description": `"Implementar login"`, "hours": `3.5`,
	})
}

func TestEffortSubmitOmitsTaskIDForWholeStory(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 0, 1) // whole-story row, second member
	m = fillEffortForm(m, [3]string{"2026", "10", "05"}, "Revisión", "2")

	_, post := submitEffort(t, m, rec)
	assertEffortBody(t, post.Body, map[string]string{
		"member_id": `"m2"`, "work_date": `"2026-10-05"`, "description": `"Revisión"`, "hours": `2`,
	})
	if strings.Contains(string(post.Body), "task_id") {
		t.Errorf("a whole-story registration must omit task_id, not send null: %s", post.Body)
	}
}

func TestEffortSubmitSendsDescriptionUntrimmed(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)
	m = fillEffortForm(m, [3]string{"2026", "10", "05"}, "  Implementar login  ", "3.5")

	_, post := submitEffort(t, m, rec)
	if got := bodyKeys(t, post.Body)["description"]; string(got) != `"  Implementar login  "` {
		t.Errorf("description = %s, want the text exactly as typed", got)
	}
}

func TestEffortSubmitNormalizesWorkDate(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)
	m = fillEffortForm(m, [3]string{"2026", "3", "7"}, "Implementar login", "1")

	_, post := submitEffort(t, m, rec)
	if got := bodyKeys(t, post.Body)["work_date"]; string(got) != `"2026-03-07"` {
		t.Errorf("work_date = %s, want the zero-padded 2026-03-07", got)
	}
}

func TestEffortFormLocalValidationBlocksRequest(t *testing.T) {
	const hoursNumberMessage = "Las horas deben ser un número mayor que 0"
	const hoursPositiveMessage = "Las horas deben ser mayores que 0"
	tests := []struct {
		name      string
		date      [3]string
		hours     string
		want      []string // fragments the error must contain
		notWanted string   // a message that must not appear
		shown     [3]string
	}{
		{"incomplete date", [3]string{"2026", "10", ""}, "2", []string{"fecha", "incompleta"}, "", [3]string{"2026", "10", "DD"}},
		{"impossible date", [3]string{"2026", "02", "30"}, "2", []string{"fecha", "válida"}, "", [3]string{"2026", "02", "30"}},
		{"empty hours", [3]string{"2026", "10", "05"}, "", []string{hoursNumberMessage}, "", [3]string{"2026", "10", "05"}},
		{"non numeric hours", [3]string{"2026", "10", "05"}, "abc", []string{hoursNumberMessage}, "", [3]string{"2026", "10", "05"}},
		{"comma decimal", [3]string{"2026", "10", "05"}, "1,5", []string{hoursNumberMessage}, "", [3]string{"2026", "10", "05"}},
		{"NaN", [3]string{"2026", "10", "05"}, "NaN", []string{hoursNumberMessage}, "", [3]string{"2026", "10", "05"}},
		{"Inf", [3]string{"2026", "10", "05"}, "Inf", []string{hoursNumberMessage}, "", [3]string{"2026", "10", "05"}},
		{"+Inf", [3]string{"2026", "10", "05"}, "+Inf", []string{hoursNumberMessage}, "", [3]string{"2026", "10", "05"}},
		{"zero hours", [3]string{"2026", "10", "05"}, "0", []string{hoursPositiveMessage}, hoursNumberMessage, [3]string{"2026", "10", "05"}},
		{"negative hours", [3]string{"2026", "10", "05"}, "-2", []string{hoursPositiveMessage}, hoursNumberMessage, [3]string{"2026", "10", "05"}},
		{"date is reported before hours", [3]string{"2026", "10", ""}, "abc", []string{"fecha", "incompleta"}, "Las horas", [3]string{"2026", "10", "DD"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, rec := effortFlowServer(t, effortFlowOptions{})
			m := openEffortFormFor(t, server.URL, 0, 1, 0)
			m = fillEffortForm(m, tt.date, "Revisión", tt.hours)

			m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
			if cmd != nil {
				t.Fatal("an invalid form must not emit a command")
			}
			if posts := effortPosts(rec); len(posts) != 0 {
				t.Fatalf("an invalid form sent %+v", posts)
			}
			view := stripANSI(m.View())
			for _, want := range tt.want {
				if !strings.Contains(view, want) {
					t.Errorf("the form should say %q: %q", want, view)
				}
			}
			if tt.notWanted != "" && strings.Contains(view, tt.notWanted) {
				t.Errorf("the form must not say %q: %q", tt.notWanted, view)
			}
			if got := fieldValue(t, view, "Horas"); got != tt.hours {
				t.Errorf("Horas = %q, want the typed %q", got, tt.hours)
			}
			if got := fieldValue(t, view, "Actividad"); got != "Revisión" {
				t.Errorf("Actividad = %q, want the typed text", got)
			}
			if got := effortDate(t, view); got != tt.shown {
				t.Errorf("date = %v, want %v kept", got, tt.shown)
			}
			if m.Screen != "effort-form" {
				t.Errorf("Screen = %q, want to stay on the form", m.Screen)
			}
		})
	}
}

func TestEffortFormLocalErrorClearsOnEdit(t *testing.T) {
	edits := []struct {
		name string
		edit func(m tui.Model) tui.Model
	}{
		{"hours", func(m tui.Model) tui.Model { return press(m, tea.KeyBackspace) }},
		{"activity", func(m tui.Model) tui.Model { return typeText(press(m, tea.KeyShiftTab), "x") }},
		{"date", func(m tui.Model) tui.Model { return press(press(press(m, tea.KeyUp), tea.KeyUp), tea.KeyBackspace) }},
	}
	for _, tt := range edits {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := effortFlowServer(t, effortFlowOptions{})
			m := openEffortFormFor(t, server.URL, 0, 1, 0)
			m = fillEffortForm(m, [3]string{"2026", "10", "05"}, "Revisión", "0")
			m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
			if view := stripANSI(m.View()); !strings.Contains(view, "Las horas deben ser mayores que 0") {
				t.Fatalf("setup: expected the zero-hours error: %q", view)
			}

			m = tt.edit(m)
			if view := stripANSI(m.View()); strings.Contains(view, "Las horas deben") {
				t.Fatalf("editing %s should clear the local error: %q", tt.name, view)
			}
		})
	}
}

func TestEffortFormConfirmsAfterCorrectingTheLocalError(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)
	m = fillEffortForm(m, [3]string{"2026", "10", "05"}, "Revisión", "0")
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("zero hours must not emit a command")
	}

	m = typeText(press(m, tea.KeyBackspace), "2")
	_, post := submitEffort(t, m, rec)
	if got := bodyKeys(t, post.Body)["hours"]; string(got) != "2" {
		t.Errorf("hours = %s, want the corrected 2", got)
	}
}

// [caracterización] born green: only the checks the spec lists are local, so the
// range, the decimals, an empty activity and a future date reach the API.
func TestEffortFormLeavesRangeDecimalsAndDescriptionToAPI(t *testing.T) {
	tests := []struct {
		name        string
		date        [3]string
		description string
		hours       string
		wantHours   string
		wantDate    string
	}{
		{"hours above a day", [3]string{"2026", "10", "05"}, "Revisión", "25", "25", "2026-10-05"},
		{"many decimals", [3]string{"2026", "10", "05"}, "Revisión", "1.234", "1.234", "2026-10-05"},
		{"empty activity and future date", [3]string{"2099", "01", "01"}, "", "2", "2", "2099-01-01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, rec := effortFlowServer(t, effortFlowOptions{})
			m := openEffortFormFor(t, server.URL, 0, 1, 0)
			m = fillEffortForm(m, tt.date, tt.description, tt.hours)

			_, post := submitEffort(t, m, rec)
			keys := bodyKeys(t, post.Body)
			if string(keys["hours"]) != tt.wantHours || string(keys["work_date"]) != `"`+tt.wantDate+`"` {
				t.Errorf("body = %s, want hours %s on %s", post.Body, tt.wantHours, tt.wantDate)
			}
			if want := `"` + tt.description + `"`; string(keys["description"]) != want {
				t.Errorf("description = %s, want %s", keys["description"], want)
			}
		})
	}
}

// startSaving fills a valid form and presses Enter once, leaving the POST in
// flight: the returned command has not been executed yet.
func startSaving(t *testing.T, url string) (tui.Model, tea.Cmd) {
	t.Helper()
	m := openEffortFormFor(t, url, 0, 1, 0)
	m = fillEffortForm(m, [3]string{"2026", "10", "05"}, "Implementar login", "3.5")
	m, cmd := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("a valid form should emit the POST command")
	}
	return m, cmd
}

func TestEffortDoubleEnterEmitsSingleRequest(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m, first := startSaving(t, server.URL)
	if view := stripANSI(m.View()); !strings.Contains(view, "Registrando…") {
		t.Fatalf("the form should show the saving indicator while the POST is in flight: %q", view)
	}

	m, second := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
	if second != nil {
		t.Fatal("a second Enter while saving must not emit another command")
	}
	m = run(t, m, first)
	if posts := effortPosts(rec); len(posts) != 1 {
		t.Fatalf("POSTs = %d (%+v), want exactly 1", len(posts), posts)
	}
	if m.Screen != "effort-result" {
		t.Errorf("Screen = %q, want effort-result after the 201", m.Screen)
	}
}

func TestEffortEscIgnoredWhileSaving(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m, cmd := startSaving(t, server.URL)

	m, escCmd := apply(m, tea.KeyMsg{Type: tea.KeyEsc})
	view := stripANSI(m.View())
	if escCmd != nil || m.Screen != "effort-form" || !strings.Contains(view, "Registrando…") {
		t.Fatalf("Esc while saving must keep the form with the indicator (screen %q, cmd %v): %q", m.Screen, escCmd != nil, view)
	}
	m = run(t, m, cmd)
	if view := stripANSI(m.View()); m.Screen != "effort-result" || !strings.Contains(view, "Esfuerzo registrado") {
		t.Fatalf("the response should still be presented after the ignored Esc (screen %q): %q", m.Screen, view)
	}
}

func TestEffortOtherKeysIgnoredWhileSaving(t *testing.T) {
	server, _ := effortFlowServer(t, effortFlowOptions{})
	m, _ := startSaving(t, server.URL)
	before := stripANSI(m.View())
	if footer := strings.Join(strings.Fields(before), " "); !strings.Contains(footer, "Ctrl+C salir") || strings.Contains(footer, "Enter siguiente/enviar") || strings.Contains(footer, "Esc volver") {
		t.Fatalf("while saving the footer should offer only Ctrl+C: %q", footer)
	}

	keys := []tea.Msg{keyMsg("x"), keyMsg("q"), keyMsg("?"), spaceKey(), tea.KeyMsg{Type: tea.KeyBackspace}, tea.KeyMsg{Type: tea.KeyTab}, tea.KeyMsg{Type: tea.KeyShiftTab}, tea.KeyMsg{Type: tea.KeyLeft}}
	for _, key := range keys {
		var cmd tea.Cmd
		m, cmd = apply(m, key)
		if cmd != nil {
			t.Fatalf("%v while saving must not emit a command", key)
		}
	}
	if after := stripANSI(m.View()); after != before {
		t.Fatalf("keys while saving must not change the form:\nbefore %q\nafter  %q", before, after)
	}
	if _, cmd := apply(m, tea.KeyMsg{Type: tea.KeyCtrlC}); !isQuit(cmd) {
		t.Fatal("Ctrl+C must still quit while saving")
	}
}

// assertFormKeptAfterFailure checks that a rejected or unconfirmed save leaves
// the form open with every typed value, no saving indicator and no success.
func assertFormKeptAfterFailure(t *testing.T, m tui.Model) {
	t.Helper()
	view := stripANSI(m.View())
	if m.Screen != "effort-form" {
		t.Fatalf("Screen = %q, want to stay on the form: %q", m.Screen, view)
	}
	if strings.Contains(view, "Registrando…") || strings.Contains(view, "Esfuerzo registrado") {
		t.Errorf("a failed save must neither keep saving nor report success: %q", view)
	}
	if got := effortDate(t, view); got != [3]string{"2026", "10", "05"} {
		t.Errorf("date = %v, want 2026-10-05 kept", got)
	}
	if got := fieldValue(t, view, "Actividad"); got != "Implementar login" {
		t.Errorf("Actividad = %q, want the typed text kept", got)
	}
	if got := fieldValue(t, view, "Horas"); got != "3.5" {
		t.Errorf("Horas = %q, want 3.5 kept", got)
	}
	if !strings.Contains(strings.Join(strings.Fields(view), " "), "Enter siguiente/enviar") {
		t.Errorf("the form footer should come back after the answer: %q", view)
	}
}

func TestEffortFormShowsAPIErrorsKeepingData(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   []string
	}{
		{"422 with fields", 422, `{"error":"validation_failed","message":"one or more fields are invalid","fields":{"hours":"hours must be greater than 0","description":"description is required"}}`,
			[]string{"La API rechazó los datos (422)", "- description: description is required", "- hours: hours must be greater than 0"}},
		{"400 invalid request", 400, `{"error":"invalid_request","message":"request body must be a valid JSON object"}`,
			[]string{"Error HTTP 400", "invalid_request"}},
		{"404 member not found", 404, `{"error":"member_not_found","message":"member not found"}`,
			[]string{"El recurso solicitado no existe (404)", "member_not_found"}},
		{"500 internal error", 500, `{"error":"internal_error","message":"internal server error"}`,
			[]string{"Error HTTP 500", "internal_error"}},
		// D5: only 201 is success. These rows were born green ([caracterización]).
		{"200 is not the registration contract", 200, effortRecordBody,
			[]string{"Error HTTP 200"}},
		{"204 without a body", 204, ``,
			[]string{"Error HTTP 204", "sin detalle"}},
		{"409 conflict", 409, `{"error":"conflict","message":"already registered"}`,
			[]string{"Error HTTP 409", "conflict"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, rec := effortFlowServer(t, effortFlowOptions{postStatus: tt.status, postBody: tt.body})
			m, cmd := startSaving(t, server.URL)
			m = run(t, m, cmd)

			view := stripANSI(m.View())
			for _, want := range tt.want {
				if !strings.Contains(view, want) {
					t.Errorf("the form should show %q: %q", want, view)
				}
			}
			assertFormKeptAfterFailure(t, m)
			if posts := effortPosts(rec); len(posts) != 1 {
				t.Errorf("POSTs = %d, want 1", len(posts))
			}
		})
	}
}

func TestEffortFormTransportErrorWarnsUnconfirmed(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m, cmd := startSaving(t, server.URL)
	server.Close() // the connection fails before the answer arrives

	m = run(t, m, cmd)
	view := stripANSI(m.View())
	for _, want := range []string{"No se pudo confirmar el registro", "conexión", "verificar"} {
		if !strings.Contains(view, want) {
			t.Errorf("the form should say %q: %q", want, view)
		}
	}
	assertFormKeptAfterFailure(t, m)
	if posts := effortPosts(rec); len(posts) != 0 {
		t.Errorf("the closed server recorded %d POSTs, want none", len(posts))
	}
	if _, cmd := apply(m, keyMsg("x")); isQuit(cmd) {
		t.Error("the TUI must stay open after a transport error")
	}
}

func TestEffortFormCorrectAndResubmitAfterRejection(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{
		postStatus: 422, postFailures: 1,
		postBody: `{"error":"validation_failed","message":"one or more fields are invalid","fields":{"hours":"hours must be at most 24"}}`,
	})
	m, cmd := startSaving(t, server.URL)
	m = run(t, m, cmd)
	if view := stripANSI(m.View()); m.Screen != "effort-form" || !strings.Contains(view, "- hours: hours must be at most 24") {
		t.Fatalf("setup: the rejection should be listed on the form (screen %q): %q", m.Screen, view)
	}

	for i := 0; i < 3; i++ { // erase 3.5
		m = press(m, tea.KeyBackspace)
	}
	m = typeText(m, "2")
	m, post := submitEffort(t, m, rec)
	if got := bodyKeys(t, post.Body)["hours"]; string(got) != "2" {
		t.Errorf("hours = %s, want the corrected 2", got)
	}
	if m.Screen != "effort-result" {
		t.Errorf("Screen = %q, want effort-result after the 201", m.Screen)
	}
}

// saveMessage runs a save command without applying its result, so a test can
// deliver it to the model later (or twice).
func saveMessage(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a save command, got nil")
	}
	return cmd()
}

func TestEffortSaveResultWithoutPendingSaveIsDiscarded(t *testing.T) {
	rejection := `{"error":"validation_failed","message":"one or more fields are invalid","fields":{"hours":"hours must be at most 24"}}`
	options := effortFlowOptions{postStatus: 422, postFailures: 1, postBody: rejection}

	t.Run("a delivered answer is not applied again", func(t *testing.T) {
		server, _ := effortFlowServer(t, options)
		m, cmd := startSaving(t, server.URL)
		msg := saveMessage(t, cmd)
		m, _ = apply(m, msg)
		m = press(m, tea.KeyBackspace) // editing clears the error
		want := stripANSI(m.View())

		m, _ = apply(m, msg)
		if got := stripANSI(m.View()); got != want || m.Screen != "effort-form" {
			t.Fatalf("a repeated answer must change nothing (screen %q):\nwant %q\ngot  %q", m.Screen, want, got)
		}
	})
	t.Run("an old answer does not touch a newer save", func(t *testing.T) {
		server, _ := effortFlowServer(t, options)
		m, first := startSaving(t, server.URL)
		old := saveMessage(t, first)
		m, _ = apply(m, old)
		m, second := apply(m, tea.KeyMsg{Type: tea.KeyEnter}) // resubmit while the form is back
		if second == nil {
			t.Fatal("resubmitting should emit a new POST")
		}
		pending := stripANSI(m.View())

		m, _ = apply(m, old)
		if got := stripANSI(m.View()); got != pending || m.Screen != "effort-form" {
			t.Fatalf("an old answer must not replace the pending save (screen %q):\nwant %q\ngot  %q", m.Screen, pending, got)
		}
		m, _ = apply(m, saveMessage(t, second))
		if m.Screen != "effort-result" {
			t.Errorf("the current answer should still be applied, Screen = %q", m.Screen)
		}
	})
	t.Run("an answer from a form that was left and reopened", func(t *testing.T) {
		server, _ := effortFlowServer(t, options)
		m, first := startSaving(t, server.URL)
		old := saveMessage(t, first)
		m, _ = apply(m, old)
		m = press(m, tea.KeyEsc) // back to the members
		m, _ = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
		m = fillEffortForm(m, [3]string{"2026", "10", "05"}, "Implementar login", "3.5")
		m, second := apply(m, tea.KeyMsg{Type: tea.KeyEnter})
		if second == nil {
			t.Fatal("the reopened form should emit its own POST")
		}
		pending := stripANSI(m.View())

		m, _ = apply(m, old)
		if got := stripANSI(m.View()); got != pending || m.Screen != "effort-form" {
			t.Fatalf("an answer of the previous form must be dropped (screen %q):\nwant %q\ngot  %q", m.Screen, pending, got)
		}
	})
	t.Run("an answer after success", func(t *testing.T) {
		server, _ := effortFlowServer(t, effortFlowOptions{})
		m, cmd := startSaving(t, server.URL)
		msg := saveMessage(t, cmd)
		m, _ = apply(m, msg)
		want := stripANSI(m.View())

		m, _ = apply(m, msg)
		if got := stripANSI(m.View()); got != want || m.Screen != "effort-result" {
			t.Fatalf("a repeated answer must change nothing (screen %q):\nwant %q\ngot  %q", m.Screen, want, got)
		}
	})
}

// [caracterización] born green: Enter on Fecha and Actividad only advances, even
// when the form is already valid and the last field would send it.
func TestEffortFormEnterOnEarlierFieldsNeverSendsAValidForm(t *testing.T) {
	server, rec := effortFlowServer(t, effortFlowOptions{})
	m := openEffortFormFor(t, server.URL, 0, 1, 0)
	m = fillEffortForm(m, [3]string{"2026", "10", "05"}, "Implementar login", "3.5")
	m = press(press(m, tea.KeyShiftTab), tea.KeyShiftTab)
	assertFocus(t, m, "Fecha")

	for _, next := range []string{"Actividad", "Horas"} {
		var cmd tea.Cmd
		m, cmd = apply(m, tea.KeyMsg{Type: tea.KeyEnter})
		if cmd != nil {
			t.Fatalf("Enter before %s must only advance, but it emitted a command", next)
		}
		assertFocus(t, m, next)
	}
	if posts := effortPosts(rec); len(posts) != 0 {
		t.Fatalf("advancing sent %+v", posts)
	}
}
