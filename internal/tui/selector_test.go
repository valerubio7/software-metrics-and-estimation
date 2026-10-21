package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func selectorRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func selectorItems() []selectorItem {
	return []selectorItem{
		{ID: "s1", Label: "Login", Meta: "alta · pendiente"},
		{ID: "s2", Label: "Pago", Meta: "media · en progreso"},
		{ID: "s3", Label: "Reporte", Meta: "baja · completada"},
	}
}

func TestSelectorFiltersLabelMetaAndIDWithoutLosingTheCollection(t *testing.T) {
	selector := selector{}
	selector.setItems(selectorItems())

	for _, tc := range []struct {
		filter string
		want   string
	}{
		{"login", "Login"},
		{"ALTA", "Login"},
		{"s2", "Pago"},
		{"reporte", "Reporte"},
	} {
		selector.filter = tc.filter
		visible := selector.visible()
		if len(visible) != 1 || visible[0].Label != tc.want {
			t.Errorf("filter %q = %#v, want only %q", tc.filter, visible, tc.want)
		}
	}
	selector.filter = ""
	if len(selector.visible()) != 3 {
		t.Errorf("clearing the filter lost items: %#v", selector.visible())
	}
}

func TestSelectorSlashAndPrintableCharactersStartTheFilter(t *testing.T) {
	selector := selector{}
	selector.setItems(selectorItems())

	if action := selector.update(selectorRune('/')); action != selectorIdle || !selector.filtering {
		t.Fatalf("slash = action %v filtering %t, want idle/true", action, selector.filtering)
	}
	if action := selector.update(selectorRune('p')); action != selectorIdle || selector.filter != "p" {
		t.Fatalf("typing = action %v filter %q, want idle/p", action, selector.filter)
	}

	selector.filtering, selector.filter = false, ""
	if action := selector.update(selectorRune('r')); action != selectorIdle || !selector.filtering || selector.filter != "r" {
		t.Fatalf("printable start = action %v filtering %t filter %q, want idle/true/r", action, selector.filtering, selector.filter)
	}
}

func TestSelectorEscapeClearsTheFilterBeforeGoingBack(t *testing.T) {
	selector := selector{}
	selector.setItems(selectorItems())
	selector.filter = "login"

	if action := selector.update(tea.KeyMsg{Type: tea.KeyEsc}); action != selectorIdle || selector.filter != "" {
		t.Fatalf("first Esc = action %v filter %q, want idle/empty", action, selector.filter)
	}
	if action := selector.update(tea.KeyMsg{Type: tea.KeyEsc}); action != selectorBack {
		t.Fatalf("second Esc = action %v, want back", action)
	}
}

func TestSelectorMovesTheCursorWithinVisibleItems(t *testing.T) {
	selector := selector{}
	selector.setItems(selectorItems())

	selector.update(tea.KeyMsg{Type: tea.KeyDown})
	if selector.cursor != 1 {
		t.Fatalf("cursor after down = %d, want 1", selector.cursor)
	}
	selector.update(tea.KeyMsg{Type: tea.KeyDown})
	selector.update(tea.KeyMsg{Type: tea.KeyDown})
	if selector.cursor != 2 {
		t.Fatalf("cursor at the end = %d, want 2", selector.cursor)
	}
	selector.update(tea.KeyMsg{Type: tea.KeyUp})
	if selector.cursor != 1 {
		t.Fatalf("cursor after up = %d, want 1", selector.cursor)
	}

	selector.filter = "login"
	selector.cursor = 0
	selector.update(tea.KeyMsg{Type: tea.KeyDown})
	if selector.cursor != 0 {
		t.Fatalf("cursor moved beyond a one-item filter: %d", selector.cursor)
	}
}

func TestSelectorChoosesOnlyWhenThereIsARow(t *testing.T) {
	selector := selector{}
	if action := selector.update(tea.KeyMsg{Type: tea.KeyEnter}); action != selectorIdle {
		t.Fatalf("Enter on empty = %v, want idle", action)
	}

	selector.setItems(selectorItems())
	selector.update(tea.KeyMsg{Type: tea.KeyDown})
	item, ok := selector.selected()
	if !ok || item.ID != "s2" {
		t.Fatalf("selected = %#v/%t, want s2", item, ok)
	}
	if action := selector.update(tea.KeyMsg{Type: tea.KeyEnter}); action != selectorChosen {
		t.Fatalf("Enter on a row = %v, want chosen", action)
	}
}

func TestSelectorRefreshAndLifecycle(t *testing.T) {
	selector := selector{}
	selector.setItems(selectorItems())
	if action := selector.update(selectorRune('r')); action != selectorRefresh {
		t.Fatalf("r = %v, want refresh", action)
	}

	selector.startLoading()
	if !selector.loading || selector.err != "" {
		t.Fatalf("loading = %t err %q, want true/empty", selector.loading, selector.err)
	}
	selector.fail("API no disponible")
	if selector.loading || selector.err != "API no disponible" {
		t.Fatalf("failed = %t/%q, want false/API no disponible", selector.loading, selector.err)
	}
}

func TestSelectorSetItemsPreservesTheFilterAndClampsTheCursor(t *testing.T) {
	selector := selector{}
	selector.setItems(selectorItems())
	selector.filter = "reporte"
	selector.cursor = 8

	selector.setItems([]selectorItem{{ID: "s3", Label: "Reporte", Meta: "baja"}})
	if selector.filter != "reporte" {
		t.Fatalf("filter = %q, want it preserved across refresh", selector.filter)
	}
	if selector.cursor != 0 {
		t.Fatalf("cursor = %d, want it clamped to the only visible row", selector.cursor)
	}
}
