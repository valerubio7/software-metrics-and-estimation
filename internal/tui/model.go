package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Model is the Bubble Tea model for the TUI.
type Model struct {
	APIURL  string
	Choices []string
	Cursor  int
	Screen  string
	APIErr  error

	projectCursor int
	projectID     string
	projectAction string
	formField     int
	form          ProjectInput
	message       string
	client        *ProjectClient
}

// NewModel creates the base navigation model.
func NewModel(apiURL string) Model {
	return Model{
		APIURL:  apiURL,
		Choices: []string{"Proyectos", "Backlog", "Sprints", "Salir"},
		Screen:  "menu",
		client:  NewProjectClient(apiURL),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if result, ok := msg.(ProjectResultMsg); ok {
		return m.applyProjectResult(result)
	}
	key, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return m, nil
	}
	if key.Type == tea.KeyCtrlC || (key.Type == tea.KeyRunes && len(key.Runes) == 1 && (key.Runes[0] == 'q' || key.Runes[0] == 'Q')) {
		return m, tea.Quit
	}
	switch m.Screen {
	case "menu":
		return m.updateMenu(key)
	case "projects":
		return m.updateProjects(key)
	case "create-project", "edit-project":
		return m.updateProjectForm(key)
	case "project-id":
		return m.updateProjectID(key)
	case "project-result":
		if key.Type == tea.KeyEsc || key.Type == tea.KeyEnter {
			m.Screen = "projects"
			m.message = ""
		}
		return m, nil
	default:
		if key.Type == tea.KeyEsc {
			m.Screen = "menu"
			m.Cursor = 0
		}
		return m, nil
	}
}

func (m Model) updateMenu(key tea.KeyMsg) (Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyUp:
		if m.Cursor > 0 {
			m.Cursor--
		}
	case tea.KeyDown:
		if m.Cursor < len(m.Choices)-1 {
			m.Cursor++
		}
	case tea.KeyEnter:
		switch m.Cursor {
		case 0:
			m.Screen, m.projectCursor = "projects", 0
		case 1:
			m.Screen = "backlog"
		case 2:
			m.Screen = "sprints"
		default:
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) updateProjects(key tea.KeyMsg) (Model, tea.Cmd) {
	options := []string{"Crear proyecto", "Modificar proyecto", "Consultar estado", "Volver"}
	switch key.Type {
	case tea.KeyUp:
		if m.projectCursor > 0 {
			m.projectCursor--
		}
	case tea.KeyDown:
		if m.projectCursor < len(options)-1 {
			m.projectCursor++
		}
	case tea.KeyEsc:
		m.Screen, m.Cursor = "menu", 0
	case tea.KeyEnter:
		switch m.projectCursor {
		case 0:
			m.Screen, m.form, m.formField = "create-project", ProjectInput{}, 0
		case 1:
			m.Screen, m.projectID, m.form, m.formField, m.projectAction = "project-id", "", ProjectInput{}, 0, "edit"
			m.message = "Ingresá el ID del proyecto para editar."
		case 2:
			m.Screen, m.projectID, m.projectAction = "project-id", "", "status"
			m.message = "Ingresá el ID del proyecto para consultar estado."
		default:
			m.Screen, m.Cursor = "menu", 0
		}
	}
	return m, nil
}

func (m Model) updateProjectID(key tea.KeyMsg) (Model, tea.Cmd) {
	if key.Type == tea.KeyEsc {
		m.Screen, m.message = "projects", ""
		return m, nil
	}
	if key.Type == tea.KeyBackspace && len(m.projectID) > 0 {
		m.projectID = m.projectID[:len(m.projectID)-1]
		return m, nil
	}
	if key.Type == tea.KeyRunes {
		m.projectID += string(key.Runes)
		return m, nil
	}
	if key.Type == tea.KeyEnter && strings.TrimSpace(m.projectID) != "" {
		id := strings.TrimSpace(m.projectID)
		if m.projectAction == "edit" {
			m.formField = 0
			m.Screen = "edit-project"
			return m, nil
		}
		return m, m.statusCmd(id)
	}
	return m, nil
}

func (m Model) updateProjectForm(key tea.KeyMsg) (Model, tea.Cmd) {
	fields := []*string{&m.form.Name, &m.form.StartDate, &m.form.PlannedFinishDate}
	if key.Type == tea.KeyEsc {
		m.Screen = "projects"
		return m, nil
	}
	if key.Type == tea.KeyBackspace {
		field := fields[m.formField]
		if len(*field) > 0 {
			*field = (*field)[:len(*field)-1]
		}
		return m, nil
	}
	if key.Type == tea.KeyRunes {
		*fields[m.formField] += string(key.Runes)
		return m, nil
	}
	if key.Type == tea.KeyTab || key.Type == tea.KeyDown {
		m.formField = (m.formField + 1) % len(fields)
		return m, nil
	}
	if key.Type == tea.KeyShiftTab || key.Type == tea.KeyUp {
		m.formField = (m.formField + len(fields) - 1) % len(fields)
		return m, nil
	}
	if key.Type == tea.KeyEnter {
		if m.formField < len(fields)-1 {
			m.formField++
			return m, nil
		}
		if m.Screen == "create-project" {
			return m, m.createCmd(m.form)
		}
		return m, m.updateCmd(m.projectID, m.form)
	}
	return m, nil
}

// ProjectResultMsg is the asynchronous result of a project API request.
type ProjectResultMsg struct {
	Response APIResponse
	Err      error
}

func (m Model) createCmd(input ProjectInput) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		response, err := client.Create(context.Background(), input)
		return ProjectResultMsg{Response: response, Err: err}
	}
}

func (m Model) updateCmd(id string, input ProjectInput) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		response, err := client.Update(context.Background(), id, input)
		return ProjectResultMsg{Response: response, Err: err}
	}
}

func (m Model) statusCmd(id string) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		response, err := client.Status(context.Background(), id)
		return ProjectResultMsg{Response: response, Err: err}
	}
}

func (m Model) applyProjectResult(result ProjectResultMsg) (Model, tea.Cmd) {
	m.Screen = "project-result"
	if result.Err != nil {
		m.message = "Error de conexión: " + result.Err.Error()
		return m, nil
	}
	if result.Response.StatusCode < 200 || result.Response.StatusCode >= 300 {
		detail := strings.TrimSpace(string(result.Response.Body))
		if result.Response.StatusCode == 404 {
			m.message = "El proyecto no existe (404): " + detail
		} else {
			m.message = fmt.Sprintf("Error HTTP %d: %s", result.Response.StatusCode, detail)
		}
		return m, nil
	}
	var status struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if json.Unmarshal(result.Response.Body, &status) == nil && status.Status != "" {
		m.message = fmt.Sprintf("Proyecto %s: %s (%s)", status.ID, status.Name, status.Status)
	} else {
		m.message = strings.TrimSpace(string(result.Response.Body))
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	var b strings.Builder
	b.WriteString(header())
	b.WriteString("\n\n")
	if m.APIErr != nil {
		b.WriteString(errorBannerStyle.Render("API no disponible: " + m.APIErr.Error()))
		b.WriteString("\n")
		b.WriteString(errorHintStyle.Render("Revisá API_URL o levantá la API; la navegación sigue disponible."))
		b.WriteString("\n\n")
	} else {
		b.WriteString(statusOkStyle.Render("● API conectada · " + m.APIURL))
		b.WriteString("\n\n")
	}
	switch m.Screen {
	case "menu":
		for i, choice := range m.Choices {
			if m.Cursor == i {
				b.WriteString(cursorStyle.Render("▸") + " " + selectedStyle.Render(choice))
			} else {
				b.WriteString("  " + itemStyle.Render(choice))
			}
			b.WriteString("\n")
		}
	case "projects":
		b.WriteString("Gestión de proyectos\n\n")
		for i, option := range []string{"Crear proyecto", "Modificar proyecto", "Consultar estado", "Volver"} {
			prefix := "  "
			if i == m.projectCursor {
				prefix = "▸ "
			}
			b.WriteString(prefix + option + "\n")
		}
	case "create-project", "edit-project":
		title := "Crear proyecto"
		if m.Screen == "edit-project" {
			title = "Modificar proyecto " + m.projectID
		}
		b.WriteString(title + "\n\n")
		fields := []struct{ label, value string }{{"Nombre", m.form.Name}, {"Fecha inicio (YYYY-MM-DD)", m.form.StartDate}, {"Fecha fin planificada (YYYY-MM-DD)", m.form.PlannedFinishDate}}
		for i, field := range fields {
			prefix := "  "
			if i == m.formField {
				prefix = "▸ "
			}
			b.WriteString(prefix + field.label + ": " + field.value + "\n")
		}
		b.WriteString("\nTab/↑/↓ para cambiar campo; Enter en el último para enviar.\n")
	case "project-id":
		b.WriteString(m.message + "\n\nID: " + m.projectID + "_\nEnter para continuar, Esc para volver.\n")
	case "project-result":
		b.WriteString("Resultado\n\n" + m.message + "\n\nEnter/Esc para volver.\n")
	default:
		b.WriteString(screenTitleStyle.Render(m.Choices[m.Cursor]))
		b.WriteString("  ")
		b.WriteString(badgeStyle.Render("Disponible en " + placeholderRelease(m.Cursor)))
		b.WriteString("\n\nPulsa esc para volver, q para salir.\n")
	}
	return b.String()
}

func header() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Software Metrics & Estimation"))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Sprint 2 · La Interfaz · TUI base"))
	b.WriteString("\n")
	b.WriteString(ruleStyle.Render("────────────────────────────────"))
	return b.String()
}

func placeholderRelease(cursor int) string {
	switch cursor {
	case 0:
		return "TUI-02"
	case 1:
		return "TUI-03"
	default:
		return "TUI-04/TUI-05"
	}
}
