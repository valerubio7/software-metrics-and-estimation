package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// selectorItem is one row of a navigable collection.
type selectorItem struct {
	ID    string
	Label string
	Meta  string
}

// selectorAction is the outcome of a key press on a selector.
type selectorAction int

const (
	selectorIdle selectorAction = iota
	selectorChosen
	selectorRefresh
	selectorBack
)

// selector is the shared navigable collection used by the Sprint 3 sections.
// It owns filtering, the cursor and the loading/error state; screens only
// supply the items and the action attached to Enter.
type selector struct {
	items     []selectorItem
	filter    string
	filtering bool
	cursor    int
	loading   bool
	err       string
}

// setItems replaces the collection, preserving the filter and clamping the
// cursor so a refresh never leaves it out of range.
func (s *selector) setItems(items []selectorItem) {
	s.items = items
	s.loading = false
	s.err = ""
	if s.cursor < 0 {
		s.cursor = 0
	}
	if last := len(s.visible()) - 1; s.cursor > last {
		s.cursor = last
	}
	if s.cursor < 0 {
		s.cursor = 0
	}
}

func (s *selector) startLoading() {
	s.loading = true
	s.err = ""
}

func (s *selector) fail(message string) {
	s.loading = false
	s.err = message
}

// visible returns the rows matching the filter by label, metadata or ID.
func (s *selector) visible() []selectorItem {
	filter := strings.ToLower(strings.TrimSpace(s.filter))
	if filter == "" {
		return s.items
	}
	matches := make([]selectorItem, 0, len(s.items))
	for _, item := range s.items {
		haystack := strings.ToLower(item.Label + " " + item.Meta + " " + item.ID)
		if strings.Contains(haystack, filter) {
			matches = append(matches, item)
		}
	}
	return matches
}

func (s *selector) selected() (selectorItem, bool) {
	list := s.visible()
	if s.cursor < 0 || s.cursor >= len(list) {
		return selectorItem{}, false
	}
	return list[s.cursor], true
}

// update applies the common navigator keys and reports the action the screen
// must perform. Filtering owns every printable key until Enter or Esc.
func (s *selector) update(key tea.KeyMsg) selectorAction {
	list := s.visible()

	if s.filtering {
		switch key.Type {
		case tea.KeyEsc:
			s.filtering, s.filter, s.cursor = false, "", 0
			return selectorIdle
		case tea.KeyEnter:
			s.filtering = false
			return selectorIdle
		case tea.KeyUp:
			if s.cursor > 0 {
				s.cursor--
			}
			return selectorIdle
		case tea.KeyDown:
			if s.cursor < len(list)-1 {
				s.cursor++
			}
			return selectorIdle
		}
		if value, ok := editText(s.filter, key); ok {
			s.filter, s.cursor = value, 0
		}
		return selectorIdle
	}

	switch key.Type {
	case tea.KeyEsc:
		if s.filter != "" {
			s.filter, s.cursor = "", 0
			return selectorIdle
		}
		return selectorBack
	case tea.KeyUp:
		if s.cursor > 0 {
			s.cursor--
		}
	case tea.KeyDown:
		if s.cursor < len(list)-1 {
			s.cursor++
		}
	case tea.KeyEnter:
		if _, ok := s.selected(); ok {
			return selectorChosen
		}
	case tea.KeyRunes:
		switch string(key.Runes) {
		case "/":
			s.filtering = true
		case "r":
			return selectorRefresh
		default:
			s.filtering = true
			if value, ok := editText(s.filter, key); ok {
				s.filter, s.cursor = value, 0
			}
		}
	}
	return selectorIdle
}
