package tui

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
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
		marker = "▸ "
		text = stripANSI(text)
	}
	line := marker + text
	if pad := width - lipgloss.Width(line); pad > 0 {
		line += strings.Repeat(" ", pad)
	}
	if selected {
		return selectedBarStyle.Render(line)
	}
	return rowStyle.Render(line)
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

// breadcrumb renders a Home-rooted trail, highlighting the current node.
func breadcrumb(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(crumbStyle.Render("Inicio"))
	for i, part := range parts {
		b.WriteString(crumbSepStyle.Render(" › "))
		if i == len(parts)-1 {
			b.WriteString(crumbCurrentStyle.Render(part))
		} else {
			b.WriteString(crumbStyle.Render(part))
		}
	}
	return b.String()
}

// keyHints renders an ordered list of "key label" pairs for the footer.
func keyHints(pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, kbdStyle.Render(pairs[i])+" "+hintStyle.Render(pairs[i+1]))
	}
	return strings.Join(parts, hintStyle.Render("  ·  "))
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

// shortID abbreviates a UUID-like identifier for inline display.
func shortID(id string) string {
	if len(id) <= 8 {
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
