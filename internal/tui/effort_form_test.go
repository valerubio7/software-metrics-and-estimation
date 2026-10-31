package tui

import (
	"testing"
	"time"
)

func TestTodayFallsBackToTimeNowForZeroClock(t *testing.T) {
	before := time.Now().UTC().Format("2006-01-02")
	got := Model{}.today()
	after := time.Now().UTC().Format("2006-01-02")
	if got != before && got != after {
		t.Fatalf("today() = %q, want the UTC date of time.Now() (%q)", got, before)
	}
}

func TestTodayUsesInjectedClockInUTC(t *testing.T) {
	m := Model{now: func() time.Time {
		return time.Date(2026, 10, 31, 23, 30, 0, 0, time.FixedZone("UTC-3", -3*3600))
	}}
	if got := m.today(); got != "2026-11-01" {
		t.Fatalf("today() = %q, want 2026-11-01 (UTC day of 23:30 at UTC-3)", got)
	}
}
