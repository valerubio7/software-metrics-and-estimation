package tui

import (
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
type selector struct {
	items     []selectorItem
	filter    string
	filtering bool
	cursor    int
	loading   bool
	err       string
}

func (s *selector) setItems(items []selectorItem) { s.items = items }

func (s *selector) startLoading() {}

func (s *selector) fail(message string) {}

func (s *selector) visible() []selectorItem { return s.items }

func (s *selector) selected() (selectorItem, bool) { return selectorItem{}, false }

func (s *selector) update(key tea.KeyMsg) selectorAction { return selectorIdle }
