package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func dateKeyRunes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestDateFieldApplyKeyHandlesEditingKeys(t *testing.T) {
	tests := []struct {
		name    string
		field   dateField
		seg     int
		key     tea.KeyMsg
		want    dateField
		wantSeg int
	}{
		{"digits auto-advance across segments", dateField{}, segYear, dateKeyRunes("20261005"), dateField{"2026", "10", "05"}, segDay},
		{"digits stop at the full day segment", dateField{"2026", "10", "05"}, segDay, dateKeyRunes("9"), dateField{"2026", "10", "05"}, segDay},
		{"dash advances from a filled segment", dateField{Year: "2026"}, segYear, dateKeyRunes("-"), dateField{Year: "2026"}, segMonth},
		{"slash advances from a filled segment", dateField{"2026", "1", ""}, segMonth, dateKeyRunes("/"), dateField{"2026", "1", ""}, segDay},
		{"separator does not advance from an empty segment", dateField{Year: "2026"}, segMonth, dateKeyRunes("-"), dateField{Year: "2026"}, segMonth},
		{"separator does not move past the last segment", dateField{"2026", "10", "5"}, segDay, dateKeyRunes("-"), dateField{"2026", "10", "5"}, segDay},
		{"left moves back one segment", dateField{"2026", "10", "5"}, segMonth, tea.KeyMsg{Type: tea.KeyLeft}, dateField{"2026", "10", "5"}, segYear},
		{"left stops at the year", dateField{"2026", "10", "5"}, segYear, tea.KeyMsg{Type: tea.KeyLeft}, dateField{"2026", "10", "5"}, segYear},
		{"right moves forward one segment", dateField{"2026", "10", "5"}, segMonth, tea.KeyMsg{Type: tea.KeyRight}, dateField{"2026", "10", "5"}, segDay},
		{"right stops at the day", dateField{"2026", "10", "5"}, segDay, tea.KeyMsg{Type: tea.KeyRight}, dateField{"2026", "10", "5"}, segDay},
		{"backspace erases the last digit", dateField{"2026", "10", "5"}, segDay, tea.KeyMsg{Type: tea.KeyBackspace}, dateField{"2026", "10", ""}, segDay},
		{"backspace on an empty segment steps back and erases", dateField{"2026", "1", ""}, segDay, tea.KeyMsg{Type: tea.KeyBackspace}, dateField{"2026", "", ""}, segMonth},
		{"backspace on an empty year changes nothing", dateField{}, segYear, tea.KeyMsg{Type: tea.KeyBackspace}, dateField{}, segYear},
		{"letters are consumed without changing the value", dateField{Year: "20"}, segYear, dateKeyRunes("abc"), dateField{Year: "20"}, segYear},
		{"letters mixed with digits keep only the digits", dateField{}, segYear, dateKeyRunes("2a0"), dateField{Year: "20"}, segYear},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, seg, handled := tt.field.applyKey(tt.seg, tt.key)
			if !handled {
				t.Fatalf("applyKey handled = false, want true")
			}
			if got != tt.want || seg != tt.wantSeg {
				t.Errorf("applyKey = %+v seg %d, want %+v seg %d", got, seg, tt.want, tt.wantSeg)
			}
		})
	}
}

func TestDateFieldApplyKeyReportsUnhandledKeys(t *testing.T) {
	field := dateField{"2026", "10", "5"}
	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"tab", tea.KeyMsg{Type: tea.KeyTab}},
		{"shift+tab", tea.KeyMsg{Type: tea.KeyShiftTab}},
		{"up", tea.KeyMsg{Type: tea.KeyUp}},
		{"down", tea.KeyMsg{Type: tea.KeyDown}},
		{"enter", tea.KeyMsg{Type: tea.KeyEnter}},
		{"esc", tea.KeyMsg{Type: tea.KeyEsc}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, seg, handled := field.applyKey(segMonth, tt.key)
			if handled {
				t.Errorf("applyKey handled = true, want false")
			}
			if got != field || seg != segMonth {
				t.Errorf("applyKey changed state: %+v seg %d, want %+v seg %d", got, seg, field, segMonth)
			}
		})
	}
}

// The expected strings were captured from projectDateRow and projectTextRow
// before they delegated to the generic helpers, so the refactor is checked
// against the original rendering and not against itself.
func TestDateRowMatchesProjectDateRow(t *testing.T) {
	m := NewModel("")
	filled := dateField{"2026", "10", "5"}
	tests := []struct {
		name      string
		d         dateField
		focused   bool
		want      string
		wantWidth int
	}{
		{"focused with value", filled, true, "▸ Fecha inicio:          2026   10   05", 76},
		{"focused empty", dateField{}, true, "▸ Fecha inicio:          AAAA   MM   DD", 76},
		{"unfocused with value", filled, false, "  Fecha inicio:          2026   10   05 ", 0},
		{"unfocused empty", dateField{}, false, "  Fecha inicio:          AAAA   MM   DD ", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := 2
			if tt.focused {
				field = 1
			}
			m.formField, m.formSegment = field, segMonth
			want := m.projectDateRow(1, "Fecha inicio", tt.d)

			got := m.dateRow("Fecha inicio", tt.d, tt.focused, segMonth, projectLabelWidth)
			if got != want {
				t.Errorf("dateRow = %q, want projectDateRow output %q", got, want)
			}
			if tt.wantWidth > 0 {
				if lipgloss.Width(got) != tt.wantWidth {
					t.Errorf("focused row width = %d, want %d", lipgloss.Width(got), tt.wantWidth)
				}
				got = trimTrailingSpaces(got)
			}
			if got != tt.want {
				t.Errorf("dateRow = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAlignedTextRowMatchesProjectTextRow(t *testing.T) {
	m := NewModel("")
	tests := []struct {
		name      string
		value     string
		focused   bool
		want      string
		wantWidth int
	}{
		{"focused with value", "Alpha", true, "▸ Nombre:               Alpha_", 76},
		{"focused empty", "", true, "▸ Nombre:               _", 76},
		{"unfocused with value", "Alpha", false, "  Nombre:               Alpha", 0},
		{"unfocused empty", "", false, "  Nombre:               ", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := 1
			if tt.focused {
				field = 0
			}
			m.formField = field
			want := m.projectTextRow(0, "Nombre", tt.value)

			got := m.alignedTextRow("Nombre", tt.value, tt.focused, projectLabelWidth)
			if got != want {
				t.Errorf("alignedTextRow = %q, want projectTextRow output %q", got, want)
			}
			if tt.wantWidth > 0 {
				if lipgloss.Width(got) != tt.wantWidth {
					t.Errorf("focused row width = %d, want %d", lipgloss.Width(got), tt.wantWidth)
				}
				got = trimTrailingSpaces(got)
			}
			if got != tt.want {
				t.Errorf("alignedTextRow = %q, want %q", got, tt.want)
			}
		})
	}
}

func trimTrailingSpaces(s string) string {
	end := len(s)
	for end > 0 && s[end-1] == ' ' {
		end--
	}
	return s[:end]
}
