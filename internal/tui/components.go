package tui

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var ansiPattern = regexp.MustCompile("\x1b\\[[0-9;]*m")

// stripANSI removes SGR sequences so a styled string can be repainted in one
// pass without its inner resets cutting the background short.
func stripANSI(s string) string { return ansiPattern.ReplaceAllString(s, "") }

// row renders a selectable list row that spans the content width, so the
// selection reads as a full-width bar instead of a text highlight.
func (m Model) row(selected bool, text string) string {
	width := m.innerWidth()
	marker := "  "
	if selected {
		if m.quietSelection {
			marker = "› "
		} else {
			marker = "▸ "
		}
		text = stripANSI(text)
	}
	line := marker + text
	if pad := width - lipgloss.Width(line); pad > 0 {
		line += strings.Repeat(" ", pad)
	}
	if selected {
		if m.quietSelection {
			return quietBarStyle.Render(line)
		}
		return selectedBarStyle.Render(line)
	}
	return rowStyle.Render(line)
}

// splitRow renders a row with the trailing metadata aligned to the right edge,
// which keeps long lists scannable without wrapping the description. When the
// terminal is too narrow for a comfortable gap, the row falls back to a single
// left-aligned line.
func (m Model) splitRow(selected bool, left, right string) string {
	width := m.innerWidth()
	if right == "" {
		return m.row(selected, left)
	}
	available := width - 2 // marker
	rightWidth := lipgloss.Width(right)
	if available-rightWidth < 12 {
		return m.row(selected, left+"  "+right)
	}
	if lipgloss.Width(left) > available-rightWidth-2 {
		left = ansi.Truncate(left, available-rightWidth-2, "…")
	}
	gap := available - lipgloss.Width(left) - rightWidth
	if gap < 2 {
		gap = 2
	}
	return m.row(selected, left+strings.Repeat(" ", gap)+right)
}

// twoLineRow renders a selectable block of two lines: a title with its badges
// and a quieter line with the references. It is used where a single line would
// wrap and break the rhythm of the list.
func (m Model) twoLineRow(selected bool, title, meta string) string {
	width := m.innerWidth()
	if selected {
		title, meta = stripANSI(title), stripANSI(meta)
	}
	title = ansi.Truncate(title, width-2, "…")
	meta = ansi.Truncate(meta, width-4, "…")
	if selected {
		first := padToWidth("▸ "+title, width)
		second := padToWidth("  "+meta, width)
		return selectedBarStyle.Render(first) + "\n" + selectedBarStyle.Render(second)
	}
	first := padToWidth("  "+title, width)
	second := padToWidth("    "+meta, width)
	return rowStyle.Render(first) + "\n" + hintStyle.Render(second)
}

// searchLine renders the search box shared by every navigable list: the magnifier,
// the label, the typed filter and a caret while the box owns the keyboard.
func searchLine(filter string, filtering bool) string {
	cursor := ""
	if filtering {
		cursor = "_"
	}
	return cursorStyle.Render("⌕ ") + hintStyle.Render("Buscar: ") + filter + cursor
}

// tone is a semantic color role for badges and inline cues.
type tone int

const (
	toneNeutral tone = iota
	toneAccent
	toneOk
	toneWarn
	toneDanger
	toneInfo
)

// statusCue renders a colored dot plus the value verbatim. Lists use it
// instead of a badge so the API text (for example "high") is shown exactly as
// it comes.
func statusCue(text string, t tone) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	return lipgloss.NewStyle().Foreground(toneColor(t)).Bold(true).Render("● " + text)
}

func toneColor(t tone) lipgloss.AdaptiveColor {
	switch t {
	case toneAccent:
		return accent
	case toneOk:
		return ok
	case toneWarn:
		return warn
	case toneDanger:
		return danger
	case toneInfo:
		return info
	default:
		return muted
	}
}

// badge renders an uppercase label on a semantic background. The text itself
// always carries the meaning, so color is never the only cue.
func badge(text string, t tone) string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(onAccent).
		Background(toneColor(t)).
		Padding(0, 1)
	return style.Render(strings.ToUpper(text))
}

// projectStatusBadge maps the API project status to a labeled badge.
func projectStatusBadge(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "planned":
		return badge("planificado", toneInfo)
	case "active":
		return badge("en curso", toneOk)
	case "overdue":
		return badge("atrasado", toneDanger)
	default:
		if strings.TrimSpace(status) == "" {
			return ""
		}
		return badge(status, toneNeutral)
	}
}

// storyStatusBadge maps a story status to a labeled badge.
func storyStatusBadge(status string) string {
	switch normalizeStoryStatus(status) {
	case "pendiente":
		return badge("pendiente", toneWarn)
	case "en progreso":
		return badge("en progreso", toneInfo)
	case "completada":
		return badge("completada", toneOk)
	default:
		if strings.TrimSpace(status) == "" {
			return ""
		}
		return badge(status, toneNeutral)
	}
}

func normalizeStoryStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(strings.ReplaceAll(status, "_", " "))) {
	case "en curso", "en progreso", "en_progreso":
		return "en progreso"
	case "hecha", "completada", "done":
		return "completada"
	case "pendiente", "pending", "todo":
		return "pendiente"
	default:
		return strings.ToLower(strings.TrimSpace(status))
	}
}

// priorityBadge maps a free-text priority to a labeled badge, or an inline label
// when the value is outside the known vocabulary (the TUI never validates it).
func priorityBadge(priority string) string {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "alta", "high":
		return badge("alta", toneDanger)
	case "media", "medium":
		return badge("media", toneWarn)
	case "baja", "low":
		return badge("baja", toneInfo)
	default:
		if strings.TrimSpace(priority) == "" {
			return ""
		}
		return badge(priority, toneNeutral)
	}
}

// severityBadge maps the known defect severities to the semantic scale.
func severityBadge(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "alta", "high", "critical", "critica", "crítica":
		return badge(severity, toneDanger)
	case "media", "medium", "moderate":
		return badge(severity, toneWarn)
	case "baja", "low", "minor":
		return badge(severity, toneInfo)
	default:
		if strings.TrimSpace(severity) == "" {
			return ""
		}
		return badge(severity, toneNeutral)
	}
}

// defectStatusBadge maps the defect lifecycle to a labeled badge.
func defectStatusBadge(status string) string {
	switch strings.ToLower(strings.TrimSpace(strings.ReplaceAll(status, "_", " "))) {
	case "detectado", "open", "abierto":
		return badge(status, toneWarn)
	case "en correccion", "en_correccion", "in progress":
		return badge(status, toneInfo)
	case "resuelto", "resolved", "closed", "cerrado":
		return badge(status, toneOk)
	default:
		if strings.TrimSpace(status) == "" {
			return ""
		}
		return badge(status, toneNeutral)
	}
}

// breadcrumb renders the trail from Proyectos, highlighting the current node.
func breadcrumb(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	var b strings.Builder
	for i, part := range parts {
		if i > 0 {
			b.WriteString(crumbSepStyle.Render(" › "))
		}
		if i == len(parts)-1 {
			b.WriteString(crumbCurrentStyle.Render(part))
		} else {
			b.WriteString(crumbStyle.Render(part))
		}
	}
	return b.String()
}

// keyHints renders the footer as "key label" pairs, wrapping whole pairs at the
// content width so narrow terminals keep every shortcut visible instead of
// letting Bubble Tea clip the overflowing end of the line.
func (m Model) keyHints(pairs ...string) string {
	width := m.contentWidth()
	separator := hintStyle.Render("  ·  ")
	var lines []string
	var current strings.Builder
	currentWidth := 0
	for i := 0; i+1 < len(pairs); i += 2 {
		part := kbdStyle.Render(pairs[i]) + " " + hintStyle.Render(pairs[i+1])
		partWidth := lipgloss.Width(part)
		if currentWidth > 0 && currentWidth+lipgloss.Width(separator)+partWidth > width {
			lines = append(lines, current.String())
			current.Reset()
			currentWidth = 0
		}
		if currentWidth > 0 {
			current.WriteString(separator)
			currentWidth += lipgloss.Width(separator)
		}
		current.WriteString(part)
		currentWidth += partWidth
	}
	if currentWidth > 0 {
		lines = append(lines, current.String())
	}
	return strings.Join(lines, "\n")
}

// screenHeader renders the optional eyebrow, the title and the subtitle that
// open a screen. The subtitle explains context; shortcuts stay in the footer.
func screenHeader(eyebrow, title, subtitle string) string {
	var b strings.Builder
	if eyebrow != "" {
		b.WriteString(sectionLabel(eyebrow))
		b.WriteString("\n")
	}
	b.WriteString(screenTitleStyle.Render(title))
	if subtitle != "" {
		b.WriteString("\n")
		b.WriteString(hintStyle.Render(subtitle))
	}
	return b.String()
}

// sectionLabel renders an uppercase section heading.
func sectionLabel(text string) string {
	return sectionLabelStyle.Render(strings.ToUpper(text))
}

// panel wraps body in a titled, rounded container.
func panel(title, body string) string {
	content := body
	if title != "" {
		content = panelTitleStyle.Render(title) + "\n" + body
	}
	return panelStyle.Render(content)
}

// sectionIntro returns the plain-language orientation of a root tab. It never
// repeats shortcuts: the footer is the single place that lists them.
func (m Model) sectionIntro() (what, can string) {
	switch m.Screen {
	case "backlog":
		return "la lista priorizada de todo lo que el proyecto necesita, todavía sin fechas.",
			"crear historias nuevas, editarlas y abrir una historia para ver el detalle."
	case "sprints":
		return "un período de trabajo con un objetivo que agrupa las historias a completar.",
			"abrir un Sprint, asignarle historias, cerrarlo y consultar el historial."
	case "defects":
		return "un problema detectado en una historia. Acá ves todos los del proyecto, incluso los de Sprints cerrados.",
			"abrir un defecto para ver el detalle y cambiar el estado. Para reportar uno, abrí una historia."
	case "dashboard":
		return "el avance del proyecto medido en historias, horas y defectos.",
			"cambiar el ámbito, refrescar los datos y descargar el reporte."
	}
	return "", ""
}

// sectionIntroView renders the orientation block wrapped to width, with the
// label in its own column so the wrapped text stays aligned. It is empty for
// screens that are not section roots.
func (m Model) sectionIntroView(width int) string {
	what, can := m.sectionIntro()
	if what == "" {
		return ""
	}
	if width <= 0 {
		width = 80
	}
	return introLine("Qué es", what, width) + "\n" + introLine("Qué podés hacer", can, width)
}

func introLine(label, text string, width int) string {
	const gap = 2
	labelWidth := lipgloss.Width(label + ":")
	bodyWidth := width - labelWidth - gap
	if bodyWidth < 12 {
		bodyWidth = 12
	}
	lines := wrapPlain(text, bodyWidth)
	var b strings.Builder
	b.WriteString(sectionLabelStyle.Render(label + ":"))
	b.WriteString(strings.Repeat(" ", gap))
	b.WriteString(lines[0])
	for _, line := range lines[1:] {
		b.WriteString("\n")
		b.WriteString(strings.Repeat(" ", labelWidth+gap))
		b.WriteString(line)
	}
	return b.String()
}

// emptyState renders a consistent placeholder for lists with no items.
func emptyState(icon, title, hint string) string {
	var b strings.Builder
	b.WriteString(emptyTitleStyle.Render(icon + "  " + title))
	if hint != "" {
		b.WriteString("\n")
		b.WriteString(emptyHintStyle.Render("   " + hint))
	}
	return b.String()
}

// truncate shortens s to at most width runes, adding an ellipsis when clipped.
func truncate(s string, width int) string {
	if width <= 0 || utf8.RuneCountInString(s) <= width {
		return s
	}
	runes := []rune(s)
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}

// shortID abbreviates a UUID-like identifier for inline display while leaving
// short human-readable identifiers, such as test fixtures, untouched.
func shortID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:8]
}

// formRow renders a labelled input row that fills the content width. The
// focused row carries the selection bar and a trailing cursor; the plain text
// keeps the "Label: value" shape.
func (m Model) formRow(selected bool, label, value string) string {
	text := label + ": " + value
	marker := "  "
	if selected {
		marker = "▸ "
		text += "_"
	}
	line := marker + text
	if pad := m.innerWidth() - lipgloss.Width(line); pad > 0 {
		line += strings.Repeat(" ", pad)
	}
	if selected {
		return selectedBarStyle.Render(line)
	}
	return rowStyle.Render(line)
}

// wrapPlain word-wraps unstyled text to width, preserving explicit breaks.
func wrapPlain(s string, width int) []string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return []string{s}
	}
	var lines []string
	current := ""
	for _, word := range strings.Fields(s) {
		candidate := word
		if current != "" {
			candidate = current + " " + word
		}
		if lipgloss.Width(candidate) > width && current != "" {
			lines = append(lines, current)
			current = word
		} else {
			current = candidate
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

// errorBlock renders a failed operation as wrapped, danger-colored lines with a
// leading marker. It has no border of its own: the surrounding panel provides it.
func (m Model) errorBlock(message string) string {
	width := m.innerWidth() - 2
	head := lipgloss.NewStyle().Foreground(danger).Bold(true)
	body := lipgloss.NewStyle().Foreground(danger)
	var b strings.Builder
	for i, raw := range strings.Split(message, "\n") {
		for j, line := range wrapPlain(raw, width) {
			if i == 0 && j == 0 {
				b.WriteString(head.Render("✗ "+line) + "\n")
				continue
			}
			b.WriteString(body.Render("  "+line) + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// okBlock renders a successful message head in the ok style, wrapped to width.
func (m Model) okBlock(message string) string {
	head, rest, found := strings.Cut(message, "\n\n")
	width := m.innerWidth() - 2
	wrapped := wrapPlain(head, width)
	for i, line := range wrapped {
		wrapped[i] = statusOkStyle.Render(line)
	}
	out := strings.Join(wrapped, "\n")
	if found {
		out += "\n\n" + rest
	}
	return out
}

// resultBlock renders a result message: ok style for successes, the error block
// for failures.
func (m Model) resultBlock(message string, success bool) string {
	if success {
		return m.okBlock(message)
	}
	return m.errorBlock(message)
}
