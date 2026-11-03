package tui

import (
	"fmt"
	"strings"
)

// keyGroup is the heading a shortcut is listed under in the help overlay.
type keyGroup int

const (
	keyNav keyGroup = iota
	keyAction
	keyGeneral
)

var keyGroupTitles = []string{"Navegación", "Acciones", "General"}

// footerLimit is the maximum number of shortcuts shown in the footer.
const footerLimit = 5

// binding is one shortcut of the active screen: desc is the short footer label
// and help the sentence shown in the help overlay (desc when empty).
type binding struct {
	key, desc, help string
	group           keyGroup
	primary         bool // candidate for the footer
}

func (b binding) helpText() string {
	if b.help != "" {
		return b.help
	}
	return b.desc
}

type keyList []binding

func (l keyList) has(key string) bool {
	for _, b := range l {
		if b.key == key {
			return true
		}
	}
	return false
}

func (l *keyList) add(group keyGroup, primary bool, key, desc, help string) {
	*l = append(*l, binding{key: key, desc: desc, help: help, group: group, primary: primary})
}

// helpAvailable reports whether "?" opens the help: inside a text field, a
// filter or the Planning Poker vote it is a plain character.
func (m Model) helpAvailable() bool {
	return !entersText(m.Screen) && !m.filtering() && !(m.Screen == "planning-poker" && m.pokerPhase == "vote")
}

// bindings returns, in footer order, exactly the shortcuts valid on the active
// screen and step. footer() and viewHelp() both read it, so they never disagree.
func (m Model) bindings() []binding {
	var l keyList
	if m.filtering() {
		l.add(keyNav, true, "escribí", "para filtrar", "Escribir para filtrar la lista")
		l.add(keyNav, true, "↑/↓", "mover", "Mover la selección")
		l.add(keyAction, true, "Enter", "dejar de filtrar", "Dejar de filtrar")
		l.add(keyNav, true, "Esc", "limpiar", "Borrar el filtro")
		l.add(keyGeneral, false, "Ctrl+C", "salir", "Salir desde cualquier pantalla")
		return l
	}
	m.screenBindings(&l)
	if m.tabRoot() {
		l.add(keyNav, false, "Tab/Shift+Tab", "pestaña", "Pasar a la pestaña siguiente o anterior")
		l.add(keyNav, false, fmt.Sprintf("1–%d", len(workspaceTabs)), "ir a pestaña", "Ir directo a la pestaña de ese número")
	}
	if m.helpAvailable() {
		l.add(keyGeneral, false, "?", "ayuda", "Mostrar u ocultar esta ayuda")
	}
	if quitsOnQ(m.Screen) && !m.defectSubmissionInFlight() && !l.has("q") {
		l.add(keyGeneral, false, "q", "salir", "Salir")
	}
	l.add(keyGeneral, m.onlyCtrlC(), "Ctrl+C", "salir", "Salir desde cualquier pantalla")
	return l
}

// onlyCtrlC marks the steps where Ctrl+C belongs in the footer: forms, where q
// is text, and submissions in flight, where nothing else acts.
func (m Model) onlyCtrlC() bool {
	return entersText(m.Screen) || m.defectSubmissionInFlight()
}

func (m Model) screenBindings(l *keyList) {
	quit := func() { l.add(keyGeneral, true, "q", "salir", "Salir") }
	switch m.Screen {
	case "project-picker":
		l.add(keyNav, true, "↑/↓", "mover", "Mover la selección")
		l.add(keyAction, true, "Enter", "abrir", "Abrir el proyecto resaltado")
		l.add(keyAction, true, "n", "crear", "Crear un proyecto")
		l.add(keyAction, true, "e", "editar", "Editar el proyecto resaltado")
		l.add(keyNav, true, "/", "buscar", "Buscar o filtrar proyectos")
	case "create-project", "edit-project":
		if !m.projectSaving {
			m.formBindings(l, "Tab/↑/↓", "continuar", "Continuar al siguiente campo o guardar")
		}
	case "backlog":
		l.add(keyNav, true, "↑/↓", "mover", "Mover la selección")
		l.add(keyAction, true, "Enter", "editar", "Editar la historia resaltada")
		l.add(keyAction, true, "n", "crear", "Crear una historia")
		l.add(keyNav, true, "/", "buscar", "Buscar o filtrar historias")
		l.add(keyNav, true, "Esc", "proyectos", "Volver a Proyectos")
		l.add(keyAction, false, "r", "refrescar", "Refrescar el backlog")
	case "backlog-form":
		if !m.backlogForm.saving {
			m.formBindings(l, "Tab/↑/↓", "siguiente/enviar", "Pasar al siguiente campo o enviar")
		}
	case "sprints":
		l.add(keyNav, true, "↑/↓", "mover", "Mover la selección")
		l.add(keyAction, true, "Enter", "historias", "Ver las historias del Sprint resaltado")
		l.add(keyAction, true, "n", "nuevo", "Crear un Sprint")
		l.add(keyNav, true, "/", "buscar", "Buscar o filtrar Sprints")
		l.add(keyNav, true, "Esc", "proyectos", "Volver a Proyectos")
		l.add(keyAction, false, "d", "PDF", "Descargar el reporte PDF del Sprint resaltado")
		l.add(keyAction, false, "a", "asignar", "Asignar historias del backlog")
		l.add(keyAction, false, "x", "cerrar", "Cerrar el Sprint resaltado (pide confirmación)")
		l.add(keyNav, false, "h", "cerrados", "Ver los Sprints cerrados")
		l.add(keyAction, false, "r", "refrescar", "Refrescar el listado")
	case "sprint-goal":
		l.add(keyAction, true, "Enter", "enviar", "Crear el Sprint con el objetivo escrito")
		l.add(keyNav, true, "Esc", "volver", "Volver sin crear el Sprint")
	case "sprint-confirm":
		l.add(keyAction, true, "s", "confirmar", "Confirmar el cierre del Sprint")
		l.add(keyNav, true, "n/Esc", "cancelar", "Cancelar sin cerrar el Sprint")
		quit()
	case "sprint-assigned-list":
		l.add(keyNav, true, "↑/↓", "mover", "Mover la selección")
		l.add(keyAction, true, "Enter", "tareas", "Ver o agregar las tareas de la historia resaltada")
		l.add(keyAction, true, "c", "completar", "Marcar la historia resaltada como completada")
		l.add(keyNav, true, "/", "buscar", "Buscar o filtrar historias")
		l.add(keyNav, true, "Esc", "volver", "Volver a los Sprints")
		l.add(keyAction, false, "r", "refrescar", "Refrescar el listado")
	case "sprint-story-select":
		l.add(keyNav, true, "↑/↓", "mover", "Mover la selección")
		l.add(keyAction, true, "Space", "marcar", "Marcar o desmarcar la historia resaltada")
		l.add(keyAction, true, "Enter", "asignar", "Asignar las historias marcadas al Sprint")
		l.add(keyNav, true, "/", "buscar", "Buscar o filtrar historias")
		l.add(keyNav, true, "Esc", "volver", "Volver a los Sprints")
	case "sprint-task-form":
		m.formBindings(l, "Tab", "continuar/enviar", "Pasar al siguiente campo o enviar la tarea")
	case "sprint-history":
		l.add(keyNav, true, "↑/↓", "mover", "Mover la selección")
		l.add(keyAction, true, "Enter", "desplegar", "Desplegar u ocultar el Sprint cerrado")
		l.add(keyAction, true, "r", "refrescar", "Refrescar los Sprints cerrados")
		l.add(keyNav, true, "/", "buscar", "Buscar o filtrar Sprints")
		l.add(keyNav, true, "Esc", "volver", "Volver a los Sprints abiertos")
	case "section":
		m.sectionBindings(l)
	case "dashboard":
		l.add(keyAction, true, "r", "refrescar", "Refrescar el estado y las métricas del proyecto")
		l.add(keyAction, true, "s", "ámbito", "Elegir proyecto o Sprint para las métricas")
		l.add(keyAction, true, "d", "PDF", "Descargar el reporte PDF del ámbito mostrado")
		l.add(keyNav, true, "Esc", "proyectos", "Volver a Proyectos")
		quit()
	case "dashboard-sprint-select":
		m.selectorBindings(l, &m.dashboard.scopes, "elegir ámbito", "Mostrar las métricas del ámbito resaltado", "Volver al Dashboard sin cambiar el ámbito")
	case "planning-poker":
		m.pokerBindings(l)
	case "effort-form":
		if !m.effort.saving {
			l.add(keyNav, true, "Tab/↑/↓", "campo", "Cambiar de campo")
			l.add(keyAction, true, "←/→", "fecha", "Mover la fecha un día")
			l.add(keyAction, true, "Enter", "siguiente/enviar", "Pasar al siguiente campo o registrar el esfuerzo")
			l.add(keyNav, true, "Esc", "volver", "Volver sin registrar")
		}
	case "defects":
		visible := len(m.visibleDefects()) > 0
		l.add(keyNav, true, "↑/↓", "mover", "Mover la selección")
		if visible {
			l.add(keyAction, true, "Enter", "detalle", "Ver el detalle del defecto")
			l.add(keyAction, true, "u", "actualizar estado", "Actualizar el estado del defecto resaltado")
		}
		l.add(keyAction, true, "n", "nuevo defecto", "Crear un defecto")
		l.add(keyNav, true, "Esc", "proyectos", "Volver a Proyectos")
		l.add(keyNav, true, "/", "buscar", "Buscar o filtrar defectos")
		l.add(keyAction, false, "r", "refrescar", "Refrescar el listado")
	case "defect-detail":
		l.add(keyAction, true, "u", "cambiar estado", "Cambiar el estado del defecto")
		l.add(keyNav, true, "Esc", "volver al listado", "Volver al listado")
		quit()
	case "defect-story-select":
		m.selectorBindings(l, &m.defectStorySelector, "elegir historia", "Elegir la opción resaltada", "Volver al borrador del defecto")
	case "defect-sprint-select":
		m.selectorBindings(l, &m.defectSprintSelector, "elegir Sprint", "Elegir la opción resaltada", "Volver al borrador del defecto")
	case "defect-resolution-select":
		if m.defectStatusOptionsLoading || m.defectStatusLoading {
			l.add(keyNav, true, "Esc", "cancelar", "Cancelar sin cambios")
			return
		}
		m.selectorBindings(l, &m.defectResolutionSelector, "elegir Sprint", "Elegir la opción resaltada", "Cancelar sin cambios")
	case "defect-status-select":
		if !m.defectStatusLoading {
			l.add(keyNav, true, "↑/↓", "elegir estado", "Elegir un estado")
			l.add(keyAction, true, "Enter", "confirmar estado", "Confirmar el estado elegido")
			l.add(keyNav, true, "Esc", "cancelar", "Cancelar sin cambios")
			quit()
		}
	case "defect-create-form":
		l.add(keyNav, true, "Tab/↑/↓", "campo", "Cambiar de campo")
		l.add(keyAction, true, "←/→", "opción", "Cambiar severidad o estado")
		l.add(keyAction, true, "Enter", "continuar", "Continuar al selector o confirmar")
		l.add(keyNav, true, "Esc", "cancelar", "Cancelar el borrador")
	case "defect-create-confirm":
		if !m.defectCreateLoading {
			l.add(keyAction, true, "s", "confirmar registro", "Confirmar y registrar el defecto")
			l.add(keyAction, true, "n", "cancelar", "Cancelar el registro")
			l.add(keyNav, true, "Esc", "volver", "Volver al borrador")
			quit()
		}
	default:
		l.add(keyNav, true, "Esc", "volver", "Volver")
		if quitsOnQ(m.Screen) {
			quit()
		}
	}
}

// formBindings lists the keys of a text form; q and ? are text there.
func (m Model) formBindings(l *keyList, fieldKey, enter, enterHelp string) {
	l.add(keyNav, true, fieldKey, "campo", "Cambiar de campo")
	l.add(keyAction, true, "Enter", enter, enterHelp)
	l.add(keyNav, true, "Esc", "volver", "Volver sin guardar")
}

// selectorBindings lists the keys of a list selector, offering Enter only when
// it has a visible row and refresh only once it is not loading.
func (m Model) selectorBindings(l *keyList, s *selector, choose, chooseHelp, escHelp string) {
	switch {
	case s.loading:
		l.add(keyNav, true, "Esc", "volver", escHelp)
	case s.err != "":
		l.add(keyAction, true, "r", "reintentar", "Reintentar la carga")
		l.add(keyNav, true, "Esc", "volver", escHelp)
	default:
		l.add(keyNav, true, "↑/↓", "mover", "Mover la selección")
		if len(s.visible()) > 0 {
			l.add(keyAction, true, "Enter", choose, chooseHelp)
		}
		l.add(keyAction, true, "r", "refrescar", "Refrescar las opciones")
		l.add(keyNav, true, "/", "buscar", "Buscar o filtrar")
		l.add(keyNav, true, "Esc", "volver", "Limpiar filtro o "+strings.ToLower(escHelp[:1])+escHelp[1:])
	}
}

func (m Model) sectionBindings(l *keyList) {
	if m.section.step == sectionStepNotice {
		l.add(keyNav, true, "Enter/Esc", "volver", "Volver a la lista de la sección")
		return
	}
	l.add(keyNav, true, "↑/↓", "mover", "Mover la selección")
	if m.section.kind == sectionPlanningPoker && m.section.step == sectionStepMembers {
		l.add(keyAction, true, "Space", "seleccionar", "Marcar o desmarcar el participante resaltado")
		l.add(keyAction, true, "Enter", "confirmar participantes", "Confirmar los participantes marcados")
		l.add(keyNav, true, "/", "buscar", "Buscar o filtrar en la lista")
		l.add(keyNav, true, "Esc", "volver", "Limpiar el filtro y volver un nivel")
		return
	}
	action, help := "elegir", "Elegir / confirmar la fila resaltada"
	switch m.section.step {
	case sectionStepPrimary:
		if m.section.kind.primaryUsesStories() {
			action, help = "elegir historia", "Elegir la historia resaltada"
		} else {
			action, help = "elegir Sprint", "Elegir el Sprint resaltado"
		}
	case sectionStepMembers:
		action, help = "elegir integrante", "Elegir el integrante resaltado"
	case sectionStepTasks:
		action, help = "elegir tarea", "Elegir la tarea resaltada"
	case sectionStepSprints:
		action, help = "elegir Sprint", "Elegir el Sprint resaltado"
	}
	l.add(keyAction, true, "Enter", action, help)
	l.add(keyNav, true, "/", "buscar", "Buscar o filtrar en la lista")
	l.add(keyAction, true, "r", "refrescar", "Refrescar el listado actual")
	l.add(keyNav, true, "Esc", "volver", "Limpiar el filtro y volver un nivel")
}

func (m Model) pokerBindings(l *keyList) {
	switch m.pokerPhase {
	case "vote":
		l.add(keyAction, true, "0–100/?", "votar", "Registrar el voto privado")
		l.add(keyNav, true, "↑/↓", "cambiar participante", "Cambiar de participante")
		l.add(keyAction, true, "Enter", "siguiente", "Confirmar voto y pasar al siguiente participante")
	case "reveal":
		l.add(keyAction, true, "Enter/r", "revelar ronda", "Revelar los votos de la ronda")
	case "revealed":
		l.add(keyAction, true, "Enter", "elegir estimación", "Elegir la estimación final")
		l.add(keyAction, true, "n", "nueva ronda", "Empezar una nueva ronda de votos")
	case "estimate":
		l.add(keyNav, true, "↑/↓", "cambiar estimación", "Cambiar la estimación")
		l.add(keyAction, true, "Enter", "continuar", "Continuar a la confirmación")
	case "confirm":
		l.add(keyAction, true, "y/Enter", "confirmar envío", "Guardar los Story Points de la historia")
	}
	l.add(keyNav, true, "Esc", "volver", "Volver")
	l.add(keyGeneral, true, "q", "salir", "Salir")
}

func (m Model) footer() string {
	if m.showHelp {
		return m.keyHints("Esc", "cerrar ayuda", "?", "cerrar", "Ctrl+C", "salir")
	}
	var pairs []string
	shown, hidden, help := 0, false, false
	for _, b := range m.bindings() {
		switch {
		case b.key == "?":
			help = true
		case b.primary && shown < footerLimit:
			pairs = append(pairs, b.key, b.desc)
			shown++
		default:
			hidden = true
		}
	}
	if hidden && help {
		pairs = append(pairs, "?", "más")
	}
	return m.keyHints(pairs...)
}

// viewHelp lists every shortcut of the active screen grouped by keyGroup,
// omitting empty groups.
func (m Model) viewHelp() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Atajos de teclado"))
	all := m.bindings()
	for group, title := range keyGroupTitles {
		var rows strings.Builder
		for _, k := range all {
			if k.group == keyGroup(group) {
				rows.WriteString("  " + helpKeyStyle.Render(padRight(k.key, 18)) + hintStyle.Render(k.helpText()) + "\n")
			}
		}
		if rows.Len() > 0 {
			b.WriteString("\n\n" + panelTitleStyle.Render(title) + "\n" + strings.TrimRight(rows.String(), "\n"))
		}
	}
	return b.String() + "\n"
}
