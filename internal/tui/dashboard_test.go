package tui

import (
	"strings"
	"testing"
)

func TestDashboardBarClampsFractions(t *testing.T) {
	for _, tc := range []struct {
		name     string
		fraction float64
		want     string
	}{
		{"empty for zero", 0, "░░░░░░░░░░"},
		{"half", 0.5, "█████░░░░░"},
		{"full", 1, "██████████"},
		{"overrun clamps to full", 1.5, "██████████"},
		{"negative clamps to empty", -0.5, "░░░░░░░░░░"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := dashboardBar(tc.fraction, 10); got != tc.want {
				t.Fatalf("dashboardBar(%v, 10) = %q, want %q", tc.fraction, got, tc.want)
			}
		})
	}
}

func TestDashboardBarWidthFitsThePanel(t *testing.T) {
	for _, inner := range []int{10, 40, 76} {
		width := dashboardBarWidth(inner)
		if width > inner {
			t.Errorf("inner %d: bar width %d exceeds the panel", inner, width)
		}
		if got := dashboardBar(1, width); len([]rune(got)) != width {
			t.Errorf("inner %d: bar renders %d cells, want %d", inner, len([]rune(got)), width)
		}
	}
	if got := dashboardBarWidth(100); got != 28 {
		t.Errorf("wide panel bar = %d, want the 28-cell cap", got)
	}
	if got := strings.Count(dashboardBar(0.619, 28), "█"); got != 17 {
		t.Errorf("61.9%% bar has %d filled cells, want 17", got)
	}
}
