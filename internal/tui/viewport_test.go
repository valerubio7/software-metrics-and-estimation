package tui

import "testing"

func TestWindowStartKeepsTheCursorVisible(t *testing.T) {
	cases := []struct {
		name                  string
		cursor, total, height int
		want                  int
	}{
		{"zero height shows everything", 40, 100, 0, 0},
		{"collection fits", 4, 5, 10, 0},
		{"first rows", 0, 100, 10, 0},
		{"cursor centered", 50, 100, 10, 45},
		{"last rows clamped", 99, 100, 10, 90},
		{"negative cursor", -3, 100, 10, 0},
		{"cursor past the end", 200, 100, 10, 90},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := windowStart(tc.cursor, tc.total, tc.height); got != tc.want {
				t.Errorf("windowStart(%d, %d, %d) = %d, want %d", tc.cursor, tc.total, tc.height, got, tc.want)
			}
		})
	}
}

func TestWindowBoundsNeverExceedsTheCollection(t *testing.T) {
	start, end := windowBounds(99, 7, 3)
	if start != 4 || end != 7 {
		t.Errorf("windowBounds(99, 7, 3) = %d, %d, want 4, 7", start, end)
	}
	start, end = windowBounds(0, 0, 5)
	if start != 0 || end != 0 {
		t.Errorf("empty collection bounds = %d, %d, want 0, 0", start, end)
	}
}

func TestListCounterClampsTheCursor(t *testing.T) {
	for _, tc := range []struct {
		cursor, total int
		want          string
	}{
		{0, 0, ""},
		{0, 4, "1/4"},
		{3, 4, "4/4"},
		{-1, 4, "1/4"},
		{9, 4, "4/4"},
	} {
		if got := listCounter(tc.cursor, tc.total); got != tc.want {
			t.Errorf("listCounter(%d, %d) = %q, want %q", tc.cursor, tc.total, got, tc.want)
		}
	}
}

func TestScrollHintReportsHiddenRows(t *testing.T) {
	if got := scrollHint(0, 10, 10); got != "" {
		t.Errorf("scrollHint with no hidden rows = %q, want empty", got)
	}
	if got := scrollHint(3, 10, 20); got != "▲ 3  ▼ 10" {
		t.Errorf("scrollHint = %q, want the rows above and below", got)
	}
}
