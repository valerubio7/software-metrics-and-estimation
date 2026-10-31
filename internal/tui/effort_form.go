package tui

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
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
	seq                 int
	saving              bool
	err                 string
}

// openEffortForm starts a blank form for the story and task already chosen in
// the section and the given member. The date opens on today's UTC day. The
// sequence number survives the reset so an answer to a save of a previous form
// can never match the next one.
func (m Model) openEffortForm(member selectorItem) Model {
	m.effort = effortState{
		story:  m.section.selected,
		task:   m.section.detail,
		member: member,
		date:   dateFieldFromISO(m.today()),
		seq:    m.effort.seq,
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
	if m.effort.saving {
		b.WriteString("\n\n")
		b.WriteString(statusMutedStyle.Render("Registrando…"))
	}
	if m.effort.err != "" {
		b.WriteString("\n\n")
		b.WriteString(m.errorBlock(m.effort.err))
	}
	return b.String()
}

// updateEffortForm edits the focused field. Text fields consume every printable
// key, so shortcuts such as q, p, r and ? are plain characters here.
func (m Model) updateEffortForm(key tea.KeyMsg) (Model, tea.Cmd) {
	if m.effort.saving {
		// The POST is not idempotent: no key may send it again or leave the form
		// before the answer is known. Ctrl+C is resolved earlier, in Update.
		return m, nil
	}
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
			if key.Type == tea.KeyBackspace || key.Type == tea.KeyRunes {
				m.effort.err = ""
			}
			return m, nil
		}
	case effortFieldDescription:
		if value, ok := editText(m.effort.description, key); ok {
			m.effort.description, m.effort.err = value, ""
			return m, nil
		}
	case effortFieldHours:
		if value, ok := editText(m.effort.hours, key); ok {
			m.effort.hours, m.effort.err = value, ""
			return m, nil
		}
	}
	return m.navigateEffortForm(key)
}

// navigateEffortForm moves the focus between the fields, wrapping around.
// Enter advances and, on the last field, sends the form.
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
			return m, nil
		}
		return m.submitEffortForm()
	}
	return m, nil
}

// effortFormFooter lists only the keys that act inside the form: q, p, r and ?
// are text there.
func (m Model) effortFormFooter() string {
	if m.effort.saving {
		return m.keyHints("Ctrl+C", "salir")
	}
	return m.keyHints("Tab/↑/↓", "campo", "←/→", "fecha", "Enter", "siguiente/enviar", "Esc", "volver", "Ctrl+C", "salir")
}

// effortSaveResult is the asynchronous result of the effort POST, tagged with
// the sequence number of the submission that produced it.
type effortSaveResult struct {
	seq      int
	response APIResponse
	err      error
}

// submitEffortForm builds the request from the form and sends it.
func (m Model) submitEffortForm() (Model, tea.Cmd) {
	input, problem := m.validateEffortInput()
	if problem != "" {
		m.effort.err = problem
		return m, nil
	}
	m.effort.saving = true
	m.effort.seq++
	return m, m.registerEffortCmd(input)
}

// Local validation messages. Only the date and the hours are checked here; the
// range, the decimals, the activity and a future date are the API's call.
const (
	effortHoursNotNumberMessage   = "Las horas deben ser un número mayor que 0 (el separador decimal es el punto)."
	effortHoursNotPositiveMessage = "Las horas deben ser mayores que 0."

	// effortTransportMessage takes the transport error. The API is not
	// idempotent, so the user must check before sending the record again.
	effortTransportMessage = "No se pudo confirmar el registro: error de conexión (%v). Es posible que el esfuerzo se haya guardado; verificar las horas reales en Métricas antes de reintentar."
)

// validateEffortInput checks the date and the hours of the form, reporting the
// date first, and returns the request body or the message to show.
func (m Model) validateEffortInput() (EffortInput, string) {
	workDate, dateProblem := m.effort.date.validate()
	if dateProblem != "" {
		return EffortInput{}, "La fecha de trabajo " + dateProblem + "."
	}
	hours, hoursProblem := parseEffortHours(m.effort.hours)
	if hoursProblem != "" {
		return EffortInput{}, hoursProblem
	}
	return EffortInput{
		MemberID:    m.effort.member.ID,
		TaskID:      m.effort.task.ID,
		WorkDate:    workDate,
		Description: m.effort.description,
		Hours:       hours,
	}, ""
}

// parseEffortHours accepts a finite number written with a decimal point and
// greater than zero. It returns the message to show when the text is not one.
func parseEffortHours(text string) (float64, string) {
	hours, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || math.IsNaN(hours) || math.IsInf(hours, 0) {
		return 0, effortHoursNotNumberMessage
	}
	if hours <= 0 {
		return 0, effortHoursNotPositiveMessage
	}
	return hours, ""
}

func (m Model) registerEffortCmd(input EffortInput) tea.Cmd {
	client, projectID, storyID, seq := m.effortClient, m.activeProjectID, m.effort.story.ID, m.effort.seq
	return func() tea.Msg {
		response, err := client.Register(context.Background(), projectID, storyID, input)
		return effortSaveResult{seq: seq, response: response, err: err}
	}
}

// applyEffortSave shows the result screen once the API answers 201. Any other
// answer, or a transport error, keeps the form with its data and explains why.
func (m Model) applyEffortSave(result effortSaveResult) (Model, tea.Cmd) {
	if !m.effort.saving || result.seq != m.effort.seq {
		return m, nil // an answer nobody is waiting for
	}
	m.effort.saving = false
	switch {
	case result.err != nil:
		m.effort.err = fmt.Sprintf(effortTransportMessage, result.err)
	case result.response.StatusCode != http.StatusCreated:
		m.effort.err = describeBacklogFailure(result.response)
	default:
		m.Screen = "effort-result"
	}
	return m, nil
}

// viewEffortResult is a minimal marker until the result screen shows the record.
func (m Model) viewEffortResult() string {
	return m.okBlock("Esfuerzo registrado.")
}
