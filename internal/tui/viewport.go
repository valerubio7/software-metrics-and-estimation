package tui

import (
	"fmt"
	"strings"
)

// windowStart returns the first row of a window of height that keeps cursor
// visible. A zero height means "show everything" and keeps headless renders
// (tests, pipes) complete.
func windowStart(cursor, total, height int) int {
	if height <= 0 || total <= height {
		return 0
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= total {
		cursor = total - 1
	}
	start := cursor - height/2
	if start < 0 {
		start = 0
	}
	if start+height > total {
		start = total - height
	}
	return start
}

// windowBounds returns the half-open range [start,end) visible for cursor.
func windowBounds(cursor, total, height int) (int, int) {
	if height <= 0 || total <= height {
		return 0, total
	}
	start := windowStart(cursor, total, height)
	end := start + height
	if end > total {
		end = total
	}
	return start, end
}

// clampCursor keeps a cursor inside a collection of length n.
func clampCursor(cursor, n int) int {
	if n <= 0 {
		return 0
	}
	if cursor < 0 {
		return 0
	}
	if cursor >= n {
		return n - 1
	}
	return cursor
}

// listCounter renders the position of the cursor inside a collection, e.g.
// "3/40". It is the always-visible cue that a list scrolls.
func listCounter(cursor, total int) string {
	if total == 0 {
		return ""
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= total {
		cursor = total - 1
	}
	return fmt.Sprintf("%d/%d", cursor+1, total)
}

// scrollHint reports how many rows are hidden above and below the window.
func scrollHint(start, end, total int) string {
	var parts []string
	if start > 0 {
		parts = append(parts, fmt.Sprintf("▲ %d", start))
	}
	if end < total {
		parts = append(parts, fmt.Sprintf("▼ %d", total-end))
	}
	return strings.Join(parts, "  ")
}

// sectionTitle renders a pane heading with an optional counter and scroll cue.
func sectionTitle(label string, cursor, total, start, end int) string {
	line := sectionLabel(label)
	if total > 0 {
		line += "  " + appPaneMeta.Render(listCounter(cursor, total))
	}
	if cue := scrollHint(start, end, total); cue != "" {
		line += "  " + appHintDim.Render(cue)
	}
	return line
}
