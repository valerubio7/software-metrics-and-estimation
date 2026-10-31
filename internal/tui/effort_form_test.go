package tui

import (
	"reflect"
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

// Opening a section must drop whatever the previous Esfuerzo flow left in
// m.effort, but keep seq: an answer still in flight for the old flow can then
// never match a request of the new one.
func TestOpenSectionResetsEffortFlowKeepingSeq(t *testing.T) {
	m := NewModel("http://localhost")
	m.activeProjectID = "p1"
	m.effort = effortState{
		story:         selectorItem{ID: "s1", Label: "Login"},
		description:   "Implementar login",
		hours:         "3.5",
		seq:           7,
		record:        &EffortRecord{ID: "e1"},
		rawBody:       "no-json",
		metricsStatus: effortMetricsLoading,
		metricsErr:    "boom",
	}

	m, _ = m.openSection(sectionEffort)

	if want := (effortState{seq: 7}); !reflect.DeepEqual(m.effort, want) {
		t.Fatalf("effort = %+v, want only seq kept: %+v", m.effort, want)
	}
}
