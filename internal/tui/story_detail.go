package tui

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// storyDetailState references the story by ID; the story itself is always read
// from m.backlogStories so the detail never shows stale data.
type storyDetailState struct {
	storyID string
}

// openStoryDetail opens the detail of the story highlighted in the backlog.
// stories is the list as the user sees it (filtered), so the cursor indexes it
// directly. An empty list or an out-of-range cursor does nothing. Opening the
// detail never issues a request: the story is already in the loaded backlog.
func (m Model) openStoryDetail(stories []Story) (Model, tea.Cmd) {
	if m.backlogCursor < 0 || m.backlogCursor >= len(stories) {
		return m, nil
	}
	m.storyDetail = storyDetailState{storyID: stories[m.backlogCursor].ID}
	m.Screen = "story-detail"
	return m, nil
}

// detailStory rereads the story of the detail from the loaded backlog by ID, so
// a reload or an estimate is reflected the next time the detail is drawn.
func (m Model) detailStory() (Story, bool) {
	for _, story := range m.backlogStories {
		if story.ID == m.storyDetail.storyID {
			return story, true
		}
	}
	return Story{}, false
}

// storyActionsAvailable reports whether the detail may offer actions: the backlog
// is not reloading, the reload did not fail and the story is still in it.
func (m Model) storyActionsAvailable() bool {
	if m.backlogLoading || m.backlogMessage != "" {
		return false
	}
	_, found := m.detailStory()
	return found
}

// updateStoryDetail handles the keys of the story detail. Esc goes back to the
// backlog without reloading it, so its filter and cursor are kept.
func (m Model) updateStoryDetail(key tea.KeyMsg) (Model, tea.Cmd) {
	if key.Type == tea.KeyEsc {
		m.Screen = "backlog"
	}
	return m, nil
}

// storyDetailBindings lists the keys of the detail. Esc is the only primary one,
// so the footer ends in "? más"; the global keys (?, q, Ctrl+C) are added by
// bindings().
func (m Model) storyDetailBindings(l *keyList) {
	l.add(keyNav, true, "Esc", "backlog", "Volver a la lista del backlog")
}

func (m Model) viewStoryDetail() string {
	var b strings.Builder
	b.WriteString(screenTitleStyle.Render("Detalle de la historia"))
	b.WriteString("\n\n")
	// The four states are evaluated in this order: loading, error, missing, ready.
	switch {
	case m.backlogLoading:
		b.WriteString(statusMutedStyle.Render("Cargando backlog…") + "\n")
		return b.String()
	case m.backlogMessage != "":
		b.WriteString(m.errorBlock(m.backlogMessage) + "\n")
		return b.String()
	}
	story, found := m.detailStory()
	if !found {
		b.WriteString(emptyState("◇", "La historia ya no está en el backlog.", "Presione Esc para volver a la lista.") + "\n")
		return b.String()
	}
	points, hours := "", ""
	if story.StoryPoints != nil {
		points = strconv.Itoa(*story.StoryPoints)
	}
	if story.EstimatedHours != nil {
		hours = strconv.FormatFloat(*story.EstimatedHours, 'f', -1, 64)
	}
	for _, field := range []struct{ label, value string }{
		{"Título", story.Title},
		{"Descripción", story.Description},
		{"Prioridad", story.Priority},
		{"Estado", story.Status},
		{"Story points", points},
		{"Criterios de aceptación", strings.Join(story.AcceptanceCriteria, "; ")},
		{"Horas estimadas", hours},
	} {
		b.WriteString(detailField(field.label, field.value) + "\n")
	}
	return b.String()
}

// detailField renders one "label: value" line, with a dash for an empty value.
func detailField(label, value string) string {
	if strings.TrimSpace(value) == "" {
		value = "—"
	}
	return hintStyle.Render(label+": ") + value
}
