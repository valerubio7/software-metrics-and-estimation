package tui

import (
	"context"
	"encoding/json"
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

// effortMetricsStatus is the state of the project metrics query shown on the
// result screen.
type effortMetricsStatus int

const (
	effortMetricsIdle effortMetricsStatus = iota
	effortMetricsLoading
	effortMetricsReady
	effortMetricsUnavailable
)

// effortState groups the data of the effort flow. A task with an empty ID means
// the effort is registered over the whole story. seq numbers every request of the
// flow (the save and each metrics query) so only the latest answer is applied.
type effortState struct {
	story, task, member selectorItem
	date                dateField
	description, hours  string
	field               effortField
	segment             int
	seq                 int
	saving              bool
	err                 string

	record        *EffortRecord // the 201 body, nil when it could not be read
	rawBody       string        // the 201 body as received when record is nil
	metricsStatus effortMetricsStatus
	metrics       metricsPayload
	metricsErr    string // why the metrics are unavailable
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

	// effortMetricsUnreadableMessage is the reason shown when the metrics answer
	// is not a JSON object.
	effortMetricsUnreadableMessage = "No se pudo interpretar la respuesta de la API."

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
		var record EffortRecord
		if err := json.Unmarshal(result.response.Body, &record); err != nil {
			m.effort.rawBody = string(result.response.Body)
		} else {
			m.effort.record = &record
		}
		return m.startEffortMetrics()
	}
	return m, nil
}

// effortMetricsResult is the asynchronous result of a project metrics query,
// tagged with the sequence number of the query that produced it.
type effortMetricsResult struct {
	seq      int
	response APIResponse
	err      error
}

// startEffortMetrics marks a new metrics query as the one in flight and sends it.
func (m Model) startEffortMetrics() (Model, tea.Cmd) {
	m.effort.metricsStatus = effortMetricsLoading
	m.effort.seq++
	client, projectID, seq := m.metricsClient, m.activeProjectID, m.effort.seq
	return m, func() tea.Msg {
		response, err := client.Project(context.Background(), projectID)
		return effortMetricsResult{seq: seq, response: response, err: err}
	}
}

// applyEffortMetrics stores the answer of the latest metrics query. It does not
// look at the screen: the answer may arrive while the project picker, opened with
// p from the result, is covering it. Leaving the result clears metricsStatus, so
// an answer for a screen the user has left is still dropped.
func (m Model) applyEffortMetrics(result effortMetricsResult) (Model, tea.Cmd) {
	if m.effort.metricsStatus != effortMetricsLoading || result.seq != m.effort.seq {
		return m, nil // an answer nobody is waiting for
	}
	// A failed query never touches the registration: the effort is already saved.
	metrics, readable := parseMetricsPayload(result.response.Body)
	switch failure := sectionFailure(result.err, result.response); {
	case failure != "":
		m.effort.metricsStatus, m.effort.metricsErr = effortMetricsUnavailable, failure
	case !readable:
		m.effort.metricsStatus, m.effort.metricsErr = effortMetricsUnavailable, effortMetricsUnreadableMessage
	default:
		m.effort.metrics, m.effort.metricsStatus = metrics, effortMetricsReady
	}
	return m, nil
}

// updateEffortResult handles the keys of the result screen. Enter is not bound:
// leaving is Esc, so a stray Enter after the confirmation cannot skip the result.
func (m Model) updateEffortResult(key tea.KeyMsg) (Model, tea.Cmd) {
	switch {
	case key.Type == tea.KeyEsc:
		return m.leaveEffortResult(), nil
	case key.Type == tea.KeyRunes && string(key.Runes) == "r" && m.effort.metricsStatus != effortMetricsLoading:
		return m.startEffortMetrics()
	}
	return m, nil
}

// leaveEffortResult returns to the task selector of the same story, restored from
// the snapshot taken when the task was chosen, to register another effort. The
// flow data is dropped; seq survives so answers still in flight stay stale.
func (m Model) leaveEffortResult() Model {
	m.Screen = "section"
	m.section.step = sectionStepTasks
	m.section.selector = m.section.tasks
	m.effort = effortState{seq: m.effort.seq}
	return m
}

// effortResultFooter lists the keys that act on the result screen.
func (m Model) effortResultFooter() string {
	return m.keyHints("r", "refrescar métricas", "Esc", "volver", "p", "proyecto", "?", "ayuda", "q", "salir")
}

// effortResultHelpRows lists the same keys for the help overlay.
func effortResultHelpRows() [][2]string {
	return [][2]string{
		{"r", "Refrescar las métricas del proyecto"},
		{"Esc", "Volver a las tareas de la historia"},
		{"p", "Cambiar el proyecto activo"},
		{"?", "Mostrar u ocultar esta ayuda"},
		{"q", "Salir"},
	}
}

// viewEffortResult shows the record the API returned, the user's selection and
// the project metrics.
func (m Model) viewEffortResult() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Resultado"))
	b.WriteString("\n\n")
	b.WriteString(m.okBlock("Esfuerzo registrado."))
	b.WriteString("\n\n")
	if record := m.effort.record; record != nil {
		b.WriteString("Fecha: " + record.WorkDate + "\n")
		b.WriteString("Horas: " + formatMetric(&record.Hours, " h") + "\n")
		b.WriteString("Actividad: " + record.Description + "\n")
	} else {
		b.WriteString("Respuesta de la API: " + m.effort.rawBody + "\n")
	}
	b.WriteString("Historia: " + m.effort.story.Label + "\n")
	b.WriteString("Tarea: " + m.effort.task.Label + "\n")
	b.WriteString("Integrante: " + m.effort.member.Label + "\n\n")
	b.WriteString(panelTitleStyle.Render("Métricas del proyecto"))
	b.WriteString("\n")
	b.WriteString(m.viewEffortMetrics())
	return b.String()
}

// viewEffortMetrics renders the metrics block. Hours and deviation are printed
// exactly as the API returned them (nil reads "sin datos"); the TUI computes
// nothing. A failed query is a notice, never an error block, because the effort
// was registered.
func (m Model) viewEffortMetrics() string {
	switch m.effort.metricsStatus {
	case effortMetricsLoading:
		return statusMutedStyle.Render("Cargando métricas…")
	case effortMetricsUnavailable:
		return statusWarnStyle.Render("El esfuerzo quedó registrado; las métricas del proyecto no están disponibles.") +
			"\n" + m.effort.metricsErr + "\n" + hintStyle.Render("r para reintentar.")
	}
	return "Horas reales: " + formatMetric(m.effort.metrics.ActualHours, " h") + "\n" +
		"Desviación: " + formatMetric(m.effort.metrics.Deviation, "")
}
