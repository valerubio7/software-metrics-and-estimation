package tui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const errHoursNotNumber = "Horas estimadas debe ser un número"

// storyField is one labelled text input of the story form.
type storyField struct {
	label string
	value *string
}

// storyForm holds the text being edited to create (storyID == "") or replace a story.
type storyForm struct {
	storyID                                               string
	title, description, priority, status, criteria, hours string
	field                                                 int
	err                                                   string
}

// newUpdateForm prefills a form with the current values of story.
func newUpdateForm(story Story) storyForm {
	form := storyForm{
		storyID:     story.ID,
		title:       story.Title,
		description: story.Description,
		priority:    story.Priority,
		status:      story.Status,
		criteria:    strings.Join(story.AcceptanceCriteria, "; "),
	}
	if story.EstimatedHours != nil {
		form.hours = strconv.FormatFloat(*story.EstimatedHours, 'f', -1, 64)
	}
	return form
}

// fields returns the editable fields in display order. The pointers refer to f.
func (f *storyForm) fields() []storyField {
	if f.storyID == "" {
		return []storyField{
			{"Título", &f.title},
			{"Descripción", &f.description},
			{"Prioridad", &f.priority},
			{"Criterios de aceptación", &f.criteria},
		}
	}
	return []storyField{
		{"Título", &f.title},
		{"Descripción", &f.description},
		{"Prioridad", &f.priority},
		{"Estado", &f.status},
		{"Criterios de aceptación", &f.criteria},
		{"Horas estimadas", &f.hours},
	}
}

func (f storyForm) createInput() StoryCreateInput {
	return StoryCreateInput{
		Title:              f.title,
		Description:        f.description,
		Priority:           f.priority,
		AcceptanceCriteria: splitCriteria(f.criteria),
	}
}

func (f storyForm) updateInput() (StoryUpdateInput, error) {
	hours, err := parseEstimatedHours(f.hours)
	if err != nil {
		return StoryUpdateInput{}, err
	}
	return StoryUpdateInput{
		Title:              f.title,
		Description:        f.description,
		Priority:           f.priority,
		Status:             f.status,
		AcceptanceCriteria: splitCriteria(f.criteria),
		EstimatedHours:     hours,
	}, nil
}

// splitCriteria splits the semicolon-separated line into trimmed, non-empty items.
// The result is never nil so that it serializes as [] instead of null.
func splitCriteria(s string) []string {
	items := []string{}
	for _, item := range strings.Split(s, ";") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}

// parseEstimatedHours turns the field text into the request value: empty means null.
// Only non-numeric and non-finite values are rejected; ranges are the API's concern.
func parseEstimatedHours(s string) (*float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	hours, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(hours) || math.IsInf(hours, 0) {
		return nil, errors.New(errHoursNotNumber)
	}
	return &hours, nil
}

func backlogActions(hasProject bool) []string {
	if hasProject {
		return []string{"Crear historia", "Cambiar proyecto", "Refrescar", "Volver"}
	}
	return []string{"Elegir proyecto", "Volver"}
}

// openBacklog enters the backlog screen, loading the session project when there is
// one and offering the picker otherwise.
func (m Model) openBacklog() (Model, tea.Cmd) {
	m.backlogCursor = 0
	m.backlogFilter, m.backlogFiltering = "", false
	if m.activeProjectID == "" {
		return m.openProjectPicker("backlog")
	}
	m.Screen = "backlog"
	return m.startBacklogLoad()
}

// visibleStories is the backlog narrowed by the search box, preserving API order.
func (m Model) visibleStories() []Story {
	filter := strings.ToLower(strings.TrimSpace(m.backlogFilter))
	if filter == "" {
		return m.backlogStories
	}
	matches := make([]Story, 0, len(m.backlogStories))
	for _, story := range m.backlogStories {
		if strings.Contains(strings.ToLower(story.Title), filter) {
			matches = append(matches, story)
		}
	}
	return matches
}

func (m Model) startBacklogLoad() (Model, tea.Cmd) {
	m.backlogStories, m.backlogLoading, m.backlogCursor = nil, true, 0
	return m, m.listStoriesCmd(m.activeProjectID)
}

func (m Model) updateBacklog(key tea.KeyMsg) (Model, tea.Cmd) {
	actions := backlogActions(m.activeProjectID != "")
	stories := m.visibleStories()

	if m.backlogFiltering {
		switch key.Type {
		case tea.KeyEsc:
			m.backlogFiltering, m.backlogFilter, m.backlogCursor = false, "", 0
			return m, nil
		case tea.KeyEnter:
			m.backlogFiltering = false
			return m, nil
		}
		if value, ok := editText(m.backlogFilter, key); ok {
			m.backlogFilter = value
			m.backlogCursor = 0
		}
		return m, nil
	}

	last := len(actions) + len(stories) - 1
	switch key.Type {
	case tea.KeyUp:
		if m.backlogCursor > 0 {
			m.backlogCursor--
		}
	case tea.KeyDown:
		if m.backlogCursor < last {
			m.backlogCursor++
		}
	case tea.KeyEsc:
		m.Screen, m.Cursor = "menu", 0
	case tea.KeyRunes:
		switch string(key.Runes) {
		case "r":
			return m.startBacklogLoad()
		case "/":
			m.backlogFiltering = true
			return m, nil
		}
	case tea.KeyEnter:
		if m.backlogCursor >= len(actions) {
			index := m.backlogCursor - len(actions)
			if index < len(stories) {
				m.Screen, m.backlogForm = "backlog-form", newUpdateForm(stories[index])
			}
			return m, nil
		}
		switch actions[m.backlogCursor] {
		case "Crear historia":
			m.Screen, m.backlogForm = "backlog-form", storyForm{}
		case "Elegir proyecto", "Cambiar proyecto":
			return m.openProjectPicker("backlog")
		case "Refrescar":
			return m.startBacklogLoad()
		default:
			m.Screen, m.Cursor = "menu", 0
		}
	}
	return m, nil
}

func (m Model) updateBacklogForm(key tea.KeyMsg) (Model, tea.Cmd) {
	form := m.backlogForm
	fields := form.fields()
	switch key.Type {
	case tea.KeyEsc:
		m.Screen = "backlog"
		return m, nil
	case tea.KeyTab, tea.KeyDown:
		form.field = (form.field + 1) % len(fields)
	case tea.KeyShiftTab, tea.KeyUp:
		form.field = (form.field + len(fields) - 1) % len(fields)
	case tea.KeyEnter:
		if form.field < len(fields)-1 {
			form.field++
			break
		}
		return m.submitStoryForm(form)
	default:
		if value, ok := editText(*fields[form.field].value, key); ok {
			*fields[form.field].value = value
			form.err = ""
		}
	}
	m.backlogForm = form
	return m, nil
}

// submitStoryForm sends the form to the API, or reports a local numeric error without a request.
func (m Model) submitStoryForm(form storyForm) (Model, tea.Cmd) {
	m.backlogForm = form
	if form.storyID == "" {
		return m, m.createStoryCmd(m.activeProjectID, form.createInput())
	}
	input, err := form.updateInput()
	if err != nil {
		m.backlogForm.err = err.Error()
		return m, nil
	}
	return m, m.updateStoryCmd(m.activeProjectID, form.storyID, input)
}

func (m Model) updateBacklogResult(key tea.KeyMsg) (Model, tea.Cmd) {
	if key.Type != tea.KeyEsc && key.Type != tea.KeyEnter {
		return m, nil
	}
	switch m.backlogReturn {
	case "backlog-form":
		m.Screen = "backlog-form"
	case "project-picker":
		return m.openProjectPicker("backlog")
	default:
		m.Screen = "backlog"
		return m.startBacklogLoad()
	}
	return m, nil
}

func backlogCmd(op BacklogOp, call func(context.Context) (APIResponse, error)) tea.Cmd {
	return func() tea.Msg {
		response, err := call(context.Background())
		return BacklogResultMsg{Op: op, Response: response, Err: err}
	}
}

func (m Model) listStoriesCmd(projectID string) tea.Cmd {
	client := m.backlogClient
	return backlogCmd(BacklogOpList, func(ctx context.Context) (APIResponse, error) {
		return client.List(ctx, projectID)
	})
}

func (m Model) createStoryCmd(projectID string, input StoryCreateInput) tea.Cmd {
	client := m.backlogClient
	return backlogCmd(BacklogOpCreate, func(ctx context.Context) (APIResponse, error) {
		return client.Create(ctx, projectID, input)
	})
}

func (m Model) updateStoryCmd(projectID, storyID string, input StoryUpdateInput) tea.Cmd {
	client := m.backlogClient
	return backlogCmd(BacklogOpUpdate, func(ctx context.Context) (APIResponse, error) {
		return client.Update(ctx, projectID, storyID, input)
	})
}

// applyBacklogResult shows the outcome of a backlog request. A list result only
// applies while the backlog screen is waiting for it; create and update results
// always apply because the mutation already happened on the server.
func (m Model) applyBacklogResult(result BacklogResultMsg) (Model, tea.Cmd) {
	if result.Op == BacklogOpList {
		if m.Screen != "backlog" || !m.backlogLoading {
			return m, nil
		}
		m.backlogLoading = false
		if message, failed := backlogFailure(result); failed {
			m.Screen, m.backlogMessage, m.backlogReturn = "backlog-result", message, "project-picker"
			return m, nil
		}
		var listing struct {
			Stories []Story `json:"stories"`
		}
		if err := json.Unmarshal(result.Response.Body, &listing); err != nil {
			m.Screen, m.backlogReturn = "backlog-result", "project-picker"
			m.backlogMessage = "No se pudo interpretar la respuesta de la API: " + err.Error()
			return m, nil
		}
		m.backlogStories = listing.Stories
		return m, nil
	}

	m.Screen = "backlog-result"
	if message, failed := backlogFailure(result); failed {
		m.backlogMessage, m.backlogReturn = message, "backlog-form"
		return m, nil
	}
	verb := "creada"
	if result.Op == BacklogOpUpdate {
		verb = "modificada"
	}
	m.backlogMessage = fmt.Sprintf("Historia %s correctamente (%d)\n\n%s", verb, result.Response.StatusCode, describeStory(result.Response.Body))
	m.backlogReturn = "backlog"
	return m, nil
}

// backlogFailure describes a transport error or a non-2xx response.
func backlogFailure(result BacklogResultMsg) (string, bool) {
	if result.Err != nil {
		return "Error de conexión: " + result.Err.Error(), true
	}
	if status := result.Response.StatusCode; status < 200 || status >= 300 {
		return describeBacklogFailure(result.Response), true
	}
	return "", false
}

// describeBacklogFailure formats a non-2xx API response for display.
func describeBacklogFailure(response APIResponse) string {
	var payload struct {
		Error   string            `json:"error"`
		Message string            `json:"message"`
		Fields  map[string]string `json:"fields"`
	}
	raw := strings.TrimSpace(string(response.Body))
	if json.Unmarshal(response.Body, &payload) != nil || (payload.Error == "" && payload.Message == "") {
		if raw == "" {
			raw = "sin detalle"
		}
		return fmt.Sprintf("Error HTTP %d: %s", response.StatusCode, raw)
	}
	detail := payload.Message
	if payload.Error != "" {
		detail = payload.Error + ": " + payload.Message
	}
	switch response.StatusCode {
	case 422:
		var b strings.Builder
		fmt.Fprintf(&b, "La API rechazó los datos (422): %s", payload.Message)
		names := make([]string, 0, len(payload.Fields))
		for name := range payload.Fields {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fmt.Fprintf(&b, "\n- %s: %s", name, payload.Fields[name])
		}
		return b.String()
	case 404:
		return "El recurso solicitado no existe (404). Detalle de la API: " + detail
	default:
		return fmt.Sprintf("Error HTTP %d: %s", response.StatusCode, detail)
	}
}

// describeStory shows the story returned by the API, or its raw body when it is not one.
func describeStory(body []byte) string {
	var story Story
	if json.Unmarshal(body, &story) == nil && story.Title != "" {
		return storyRow(story)
	}
	return strings.TrimSpace(string(body))
}

func storyRow(story Story) string {
	points := "—"
	if story.StoryPoints != nil {
		points = strconv.Itoa(*story.StoryPoints)
	}
	hours := "—"
	if story.EstimatedHours != nil {
		hours = strconv.FormatFloat(*story.EstimatedHours, 'f', -1, 64)
	}
	title := story.Title
	parts := []string{title}
	if story.Priority != "" {
		parts = append(parts, priorityBadge(story.Priority))
	}
	if story.Status != "" {
		parts = append(parts, storyStatusBadge(story.Status))
	}
	return strings.Join(parts, "  ") + hintStyle.Render(fmt.Sprintf("   puntos %s · horas %s", points, hours))
}

// Backlog screens reuse the menu styles. The row styles drop the menu padding so
// the plain text of every row stays exactly what it was before styling.
var (
	backlogSelectedStyle = selectedStyle.UnsetPadding()
	backlogItemStyle     = itemStyle.UnsetPadding()
)

// choiceRow renders a selectable row like the menu does: a cursor and the
// highlighted text when selected, an indented item otherwise.
func choiceRow(selected bool, text string) string {
	if selected {
		return cursorStyle.Render("▸") + " " + backlogSelectedStyle.Render(text)
	}
	return "  " + backlogItemStyle.Render(text)
}

// projectLabel is the human-readable name of the session project, falling back to
// its identifier when the picker did not supply a name.
func (m Model) projectLabel() string {
	if m.activeProjectName != "" {
		return m.activeProjectName
	}
	if m.activeProjectID != "" {
		return shortID(m.activeProjectID)
	}
	return "—"
}

func (m Model) viewBacklog() string {
	var b strings.Builder
	hasProject := m.activeProjectID != ""
	if hasProject {
		b.WriteString(screenTitleStyle.Render("Backlog del proyecto " + m.projectLabel()))
	} else {
		b.WriteString(screenTitleStyle.Render("Backlog"))
		b.WriteString("\n")
		b.WriteString(hintStyle.Render("Todavía no elegiste un proyecto."))
	}
	b.WriteString("\n\n")

	actions := backlogActions(hasProject)
	for i, action := range actions {
		b.WriteString(choiceRow(i == m.backlogCursor, action) + "\n")
	}
	b.WriteString("\n")

	if m.backlogFiltering || m.backlogFilter != "" {
		cursor := ""
		if m.backlogFiltering {
			cursor = "_"
		}
		b.WriteString(cursorStyle.Render("⌕ ") + hintStyle.Render("Buscar: ") + m.backlogFilter + cursor + "\n\n")
	}

	switch {
	case m.backlogLoading:
		b.WriteString(statusMutedStyle.Render("Cargando backlog…") + "\n")
		return b.String()
	case hasProject && len(m.backlogStories) == 0:
		b.WriteString(emptyState("◇", "El backlog no tiene historias.", "Elegí «Crear historia» para agregar la primera.") + "\n")
		return b.String()
	}

	stories := m.visibleStories()
	b.WriteString(sectionLabel("Historias") + "  " + hintStyle.Render(fmt.Sprintf("(%d)", len(stories))) + "\n")
	if len(stories) == 0 {
		b.WriteString(emptyState("⌕", "Sin coincidencias.", "Probá con otro título o borrá la búsqueda.") + "\n")
		return b.String()
	}
	for i, story := range stories {
		b.WriteString(choiceRow(len(actions)+i == m.backlogCursor, storyRow(story)) + "\n")
	}
	return b.String()
}

func (m Model) viewBacklogForm() string {
	form := m.backlogForm
	var b strings.Builder
	if form.storyID == "" {
		b.WriteString(screenTitleStyle.Render("Crear historia"))
	} else {
		b.WriteString(screenTitleStyle.Render("Modificar historia " + shortID(form.storyID)))
	}
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Proyecto: " + m.projectLabel()))
	b.WriteString("\n\n")
	for i, field := range form.fields() {
		b.WriteString(formRow(i == form.field, field.label, *field.value) + "\n")
	}
	if form.err != "" {
		b.WriteString("\n" + errorBannerStyle.Render(form.err) + "\n")
	}
	b.WriteString("\n" + hintStyle.Render("Separá los criterios de aceptación con punto y coma."))
	return b.String()
}

// viewBacklogResult shows failures in the error banner and successes in the ok
// style. Only a successful mutation returns to the backlog list afterwards.
func (m Model) viewBacklogResult() string {
	var body string
	if m.backlogReturn != "backlog" {
		body = errorBannerStyle.Render(m.backlogMessage)
	} else {
		body = resultBody(m.backlogMessage, true)
	}
	return screenTitleStyle.Render("Resultado") + "\n\n" + body
}
