package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// effortField identifies a row of the effort form.
type effortField int

const (
	effortFieldDate effortField = iota
	effortFieldDescription
	effortFieldHours
	effortFormFields
)

// effortLabelWidth aligns the value column of the effort form rows: the widest
// label ("Actividad") plus its colon.
const effortLabelWidth = len("Actividad:")

// effortState groups the data of the effort form. A task with an empty ID means
// the effort is registered over the whole story.
type effortState struct {
	story, task, member selectorItem
	date                dateField
	description, hours  string
	field               effortField
	segment             int
}

// openEffortForm starts a blank form for the story and task already chosen in
// the section and the given member. The date opens on today's UTC day.
func (m Model) openEffortForm(member selectorItem) Model {
	m.effort = effortState{
		story:  m.section.selected,
		task:   m.section.detail,
		member: member,
		date:   dateFieldFromISO(m.today()),
	}
	m.Screen = "effort-form"
	return m
}

func (m Model) viewEffortForm() string {
	task := "Tarea: " + m.effort.task.Label
	if m.effort.task.ID == "" {
		task = "Tarea: toda la historia"
	}
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Registrar esfuerzo"))
	b.WriteString("\n")
	for _, line := range []string{"Historia: " + m.effort.story.Label, task, "Integrante: " + m.effort.member.Label} {
		b.WriteString(hintStyle.Render(line))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.dateRow("Fecha", m.effort.date, m.effort.field == effortFieldDate, m.effort.segment, effortLabelWidth) + "\n")
	b.WriteString(m.alignedTextRow("Actividad", m.effort.description, m.effort.field == effortFieldDescription, effortLabelWidth) + "\n")
	b.WriteString(m.alignedTextRow("Horas", m.effort.hours, m.effort.field == effortFieldHours, effortLabelWidth))
	return b.String()
}

// updateEffortForm edits the focused field. Text fields consume every printable
// key, so shortcuts such as q, p, r and ? are plain characters here.
func (m Model) updateEffortForm(key tea.KeyMsg) (Model, tea.Cmd) {
	if key.Type == tea.KeyEsc {
		// The member selector is still intact in the section, so leaving restores
		// it without a request; what was typed is dropped on the next opening.
		m.Screen = "section"
		return m, nil
	}
	switch m.effort.field {
	case effortFieldDate:
		if date, segment, handled := m.effort.date.applyKey(m.effort.segment, key); handled {
			m.effort.date, m.effort.segment = date, segment
			return m, nil
		}
	case effortFieldDescription:
		if value, ok := editText(m.effort.description, key); ok {
			m.effort.description = value
			return m, nil
		}
	case effortFieldHours:
		if value, ok := editText(m.effort.hours, key); ok {
			m.effort.hours = value
			return m, nil
		}
	}
	return m.navigateEffortForm(key)
}

// navigateEffortForm moves the focus between the fields, wrapping around.
// Enter advances; sending the form on the last field arrives with the submit.
func (m Model) navigateEffortForm(key tea.KeyMsg) (Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyTab, tea.KeyDown:
		m.effort.field = (m.effort.field + 1) % effortFormFields
		m.effort.segment = 0
	case tea.KeyShiftTab, tea.KeyUp:
		m.effort.field = (m.effort.field + effortFormFields - 1) % effortFormFields
		m.effort.segment = 0
	case tea.KeyEnter:
		if m.effort.field < effortFormFields-1 {
			m.effort.field++
			m.effort.segment = 0
		}
	}
	return m, nil
}

// effortFormFooter lists only the keys that act inside the form: q, p, r and ?
// are text there.
func (m Model) effortFormFooter() string {
	return m.keyHints("Tab/↑/↓", "campo", "←/→", "fecha", "Enter", "siguiente/enviar", "Esc", "volver", "Ctrl+C", "salir")
}
