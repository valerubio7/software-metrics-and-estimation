package tui

import (
	"fmt"
	"strings"
)

// masterDetail reports whether the active screen gets the two-pane layout.
// Tests and headless renders keep width and height at zero, so they always get
// the single full-width body they assert on.
func (m Model) masterDetail() bool {
	if m.width < 100 || m.height <= 0 {
		return false
	}
	switch m.Screen {
	case "backlog", "story-detail", "sprints", "defects", "defect-detail":
		return true
	}
	return false
}

// masterDetailView composes the two panes: the collection on the left and the
// highlighted item on the right. Both panes share one frame height, so moving
// the cursor never makes the layout jump.
func (m Model) masterDetailView() string {
	total := m.contentWidth()
	height := m.bodyInnerHeight()
	intro := m.sectionIntroView(total)
	if intro != "" {
		// The orientation block sits above both panes; keep two spare rows as
		// breathing room, and drop it when the terminal is too short to be useful.
		reserved := strings.Count(intro, "\n") + 2
		if height-reserved >= 6 {
			height -= reserved
		} else {
			intro = ""
		}
	}
	left := total * 2 / 5
	if left < 34 {
		left = 34
	}
	if left > 54 {
		left = 54
	}
	right := total - left - 1
	if right < 30 {
		right = 30
		left = total - right - 1
	}

	list := m
	list.paneWidth, list.paneHeight = left, height
	list.quietSelection = m.Screen == "story-detail" || m.Screen == "defect-detail"
	detail := m
	detail.paneWidth, detail.paneHeight = right, height
	detail.quietSelection = false

	var listBody, detailBody string
	switch m.Screen {
	case "backlog":
		listBody, detailBody = list.backlogPane(), detail.storyPreviewPane()
	case "story-detail":
		if m.storyDetail.origin == storyDetailFromSprint {
			listBody, detailBody = list.sprintStoryPane(), detail.viewStoryDetail()
		} else {
			listBody, detailBody = list.backlogPane(), detail.viewStoryDetail()
		}
	case "sprints":
		listBody, detailBody = list.sprintsPane(), detail.sprintPreviewPane()
	case "defects":
		listBody, detailBody = list.defectsPane(), detail.defectPreviewPane()
	case "defect-detail":
		listBody, detailBody = list.defectsPane(), detail.viewDefectDetail()
	}

	leftCard := m.cardAt(left, height, listBody)
	rightCard := m.cardAt(right, height, detailBody)
	body := joinPanes(leftCard, rightCard)
	if intro != "" {
		return intro + "\n\n" + body
	}
	return body
}

// joinPanes glues two frames of equal height with a one-column gutter.
func joinPanes(left, right string) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")
	out := make([]string, 0, len(leftLines))
	for i := range leftLines {
		line := leftLines[i]
		if i < len(rightLines) {
			line += " " + rightLines[i]
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// backlogPane is the left side of the Backlog tab: the searchable story list,
// windowed to the pane height.
func (m Model) backlogPane() string {
	var lines []string
	if m.backlogFiltering || m.backlogFilter != "" {
		lines = append(lines, searchLine(m.backlogFilter, m.backlogFiltering))
	}
	stories := m.visibleStories()
	switch {
	case m.backlogLoading:
		lines = append(lines, statusMutedStyle.Render("Cargando backlog…"))
		return strings.Join(lines, "\n")
	case m.backlogMessage != "":
		lines = append(lines, m.errorBlock(m.backlogMessage))
		return strings.Join(lines, "\n")
	}
	total := len(stories)
	if total == 0 {
		lines = append(lines, emptyState("◇", "El backlog no tiene historias.", "Presioná n para crear la primera."))
		return strings.Join(lines, "\n")
	}
	available := m.paneHeight - len(lines) - 1
	if available < 1 {
		available = 1
	}
	cursor := clampCursor(m.backlogCursor, total)
	start, end := windowBounds(cursor, total, available)
	lines = append(lines, sectionTitle("Historias", cursor, total, start, end))
	for i := start; i < end; i++ {
		lines = append(lines, m.row(i == cursor, storyCompactRow(stories[i])))
	}
	return strings.Join(lines, "\n")
}

// storyCompactRow is the narrow list row used by the master pane: the full row
// with badges and totals would spend its width on data the detail already shows.
// sprintStoryPane is the left side of a Sprint story detail: the stories of the
// open Sprint, windowed, so Esc returns to the same selection.
func (m Model) sprintStoryPane() string {
	var lines []string
	if m.sprintFiltering || m.sprintFilter != "" {
		lines = append(lines, searchLine(m.sprintFilter, m.sprintFiltering))
	}
	switch {
	case m.sprintStoriesLoading:
		lines = append(lines, statusMutedStyle.Render("Cargando historias…"))
		return strings.Join(lines, "\n")
	case m.sprintMessage != "":
		lines = append(lines, m.errorBlock(m.sprintMessage))
		return strings.Join(lines, "\n")
	}
	stories := m.visibleSprintStories()
	total := len(stories)
	if total == 0 {
		lines = append(lines, emptyState("◇", "El Sprint no tiene historias asignadas.", "Asigná historias con a desde Sprints."))
		return strings.Join(lines, "\n")
	}
	available := m.paneHeight - len(lines) - 1
	if available < 1 {
		available = 1
	}
	cursor := clampCursor(m.sprintCursor, total)
	start, end := windowBounds(cursor, total, available)
	lines = append(lines, sectionTitle("Historias", cursor, total, start, end))
	for i := start; i < end; i++ {
		lines = append(lines, m.row(i == cursor, storyCompactRow(stories[i])))
	}
	return strings.Join(lines, "\n")
}

func storyCompactRow(story Story) string {
	points := "—"
	if story.StoryPoints != nil {
		points = fmt.Sprintf("%d", *story.StoryPoints)
	}
	meta := normalizeStoryStatus(story.Status)
	if meta == "" {
		meta = "sin estado"
	}
	return story.Title + hintStyle.Render(fmt.Sprintf("   %s · %s pts", meta, points))
}

// storyPreviewPane is the right side of the Backlog tab: the highlighted story
// with its data, so the list never disappears while the user explores.
func (m Model) storyPreviewPane() string {
	stories := m.visibleStories()
	if len(stories) == 0 {
		return emptyState("◇", "Ninguna historia seleccionada.", "Movete con ↑/↓ para ver el detalle acá.")
	}
	story := stories[clampCursor(m.backlogCursor, len(stories))]
	points, hours := "—", "—"
	if story.StoryPoints != nil {
		points = fmt.Sprintf("%d", *story.StoryPoints)
	}
	if story.EstimatedHours != nil {
		hours = fmt.Sprintf("%g", *story.EstimatedHours)
	}
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render(story.Title))
	b.WriteString("\n")
	b.WriteString(appPaneMeta.Render("Historia " + shortID(story.ID) + " · " + strings.ToLower(story.Priority) + " · " + normalizeStoryStatus(story.Status)))
	b.WriteString("\n\n")
	for _, field := range []struct{ label, value string }{
		{"Descripción", story.Description},
		{"Story points", points},
		{"Horas estimadas", hours},
		{"Criterios de aceptación", strings.Join(story.AcceptanceCriteria, "; ")},
	} {
		b.WriteString(detailField(field.label, field.value))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// sprintsPane is the left side of the Sprints tab: the open Sprints, windowed.
func (m Model) sprintsPane() string {
	var lines []string
	if m.sprintFiltering || m.sprintFilter != "" {
		lines = append(lines, searchLine(m.sprintFilter, m.sprintFiltering))
	}
	if m.sprintLoading {
		lines = append(lines, statusMutedStyle.Render("Cargando Sprints…"))
		return strings.Join(lines, "\n")
	}
	if m.sprintMessage != "" {
		lines = append(lines, m.errorBlock(m.sprintMessage))
	}
	sprints := m.visibleOpenSprints()
	total := len(sprints)
	if total == 0 {
		lines = append(lines, emptyState("◇", "No hay Sprints abiertos.", "Presioná n para crear el primero."))
		return strings.Join(lines, "\n")
	}
	available := m.paneHeight - len(lines) - 1
	if available < 1 {
		available = 1
	}
	cursor := clampCursor(m.sprintCursor, total)
	start, end := windowBounds(cursor, total, available)
	lines = append(lines, sectionTitle("Abiertos", cursor, total, start, end))
	for i := start; i < end; i++ {
		lines = append(lines, m.row(i == cursor, sprintRow(sprints[i])))
	}
	return strings.Join(lines, "\n")
}

// sprintPreviewPane summarizes the highlighted Sprint and lists its actions.
func (m Model) sprintPreviewPane() string {
	sprints := m.visibleOpenSprints()
	if len(sprints) == 0 {
		return emptyState("◇", "Ningún Sprint seleccionado.", "Creá uno con n para empezar a planificar.")
	}
	sprint := sprints[clampCursor(m.sprintCursor, len(sprints))]
	goal := sprint.SprintGoal
	if goal == "" {
		goal = "(sin objetivo)"
	}
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render(goal))
	b.WriteString("\n")
	b.WriteString(appPaneMeta.Render("Sprint " + shortID(sprint.ID) + " · abierto"))
	b.WriteString("\n\n")
	b.WriteString(detailField("Proyecto", m.projectLabel()))
	b.WriteString("\n")
	b.WriteString(detailField("Estado", "abierto"))
	if m.report.message != "" || m.report.loading {
		b.WriteString("\n\n")
		b.WriteString(m.reportView(sprintReportScope, m.activeProjectID))
	}
	return strings.TrimRight(b.String(), "\n")
}

// defectsPane is the left side of the Defectos tab.
func (m Model) defectsPane() string {
	var lines []string
	if m.defectFiltering || m.defectFilter != "" {
		lines = append(lines, searchLine(m.defectFilter, m.defectFiltering))
	}
	if m.defectLoading {
		lines = append(lines, statusMutedStyle.Render("Cargando defectos…"))
		return strings.Join(lines, "\n")
	}
	if m.defectError != "" {
		lines = append(lines, m.errorBlock(m.defectError))
		return strings.Join(lines, "\n")
	}
	visible := m.visibleDefects()
	total := len(visible)
	if total == 0 {
		lines = append(lines, emptyState("◇", "No hay defectos registrados.", "Se registran desde el detalle de una historia."))
		return strings.Join(lines, "\n")
	}
	available := m.paneHeight - len(lines) - 1
	if available < 1 {
		available = 1
	}
	cursor := clampCursor(m.defectCursor, total)
	start, end := windowBounds(cursor, total, available)
	lines = append(lines, sectionTitle("Defectos", cursor, total, start, end))
	for i := start; i < end; i++ {
		defect := visible[i]
		lines = append(lines, m.splitRow(i == cursor, defect.Description, defectCompactMeta(defect)))
	}
	return strings.Join(lines, "\n")
}

// defectCompactMeta is the quiet metadata of the narrow defect list.
func defectCompactMeta(defect Defect) string {
	status := strings.ToLower(strings.TrimSpace(defect.Status))
	if status == "" {
		status = "sin estado"
	}
	severity := strings.ToLower(strings.TrimSpace(defect.Severity))
	if severity == "" {
		return status
	}
	return severity + " · " + status
}

// defectPreviewPane is the right side of the Defectos tab.
func (m Model) defectPreviewPane() string {
	visible := m.visibleDefects()
	if len(visible) == 0 {
		return emptyState("◇", "Ningún defecto seleccionado.", "Los defectos aparecen acá al registrarse.")
	}
	defect := visible[clampCursor(m.defectCursor, len(visible))]
	resolution := stringValue(defect.ResolutionSprintID)
	if resolution == "" {
		resolution = "—"
	}
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render(defect.Description))
	b.WriteString("\n")
	b.WriteString(appPaneMeta.Render("Defecto " + shortID(defect.ID) + " · " + strings.ToLower(defect.Status)))
	b.WriteString("\n\n")
	for _, field := range []struct{ label, value string }{
		{"Severidad", defect.Severity},
		{"Estado", defect.Status},
		{"Historia relacionada", defect.StoryID},
		{"Sprint de detección", defect.SprintID},
		{"Sprint de resolución", resolution},
	} {
		b.WriteString(detailField(field.label, field.value))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
