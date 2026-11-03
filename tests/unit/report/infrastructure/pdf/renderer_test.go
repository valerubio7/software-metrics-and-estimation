package pdf_test

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"testing"
	"time"

	metricsdomain "github.com/valerubio7/software-metrics-and-estimation/internal/metrics/domain"
	reportdomain "github.com/valerubio7/software-metrics-and-estimation/internal/report/domain"
	reportpdf "github.com/valerubio7/software-metrics-and-estimation/internal/report/infrastructure/pdf"
)

func TestRenderProjectReportIncludesEverySection(t *testing.T) {
	completedAt := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	report := reportdomain.Build(reportdomain.Snapshot{
		Project: reportdomain.Project{
			ID:                "11111111-1111-4111-8111-111111111111",
			Name:              "Proyecto Atlas",
			StartDate:         time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
			PlannedFinishDate: time.Date(2026, 12, 20, 0, 0, 0, 0, time.UTC),
		},
		Stories: []reportdomain.Story{{
			ID: "story-a", Title: "Historia A", Priority: "alta", Status: "completada",
			StoryPoints: intPointer(5), EstimatedHours: floatPointer(10), Completed: true, CompletedAt: &completedAt,
			Tasks: []reportdomain.Task{{ID: "task-a", Title: "Tarea A1", SprintID: "sprint-a", EstimatedHours: floatPointer(3)}},
		}},
		EffortRecords: []reportdomain.EffortRecord{{
			ID: "effort-a", StoryID: "story-a", StoryTitle: "Historia A", TaskID: stringPointer("task-a"),
			TaskTitle: stringPointer("Tarea A1"), TaskSprintID: stringPointer("sprint-a"),
			MemberName: "Ana Pérez", WorkDate: completedAt, Description: "Implementación", Hours: 4,
		}},
		Defects: []reportdomain.Defect{{
			ID: "defect-a", StoryID: "story-a", StoryTitle: "Historia A", Description: "Fallo de login",
			Severity: "alta", Status: "resuelto", DetectionSprintID: "sprint-a", DetectionSprintGoal: "Entrega A",
		}},
		ClosedSprints: []metricsdomain.ClosedSprintInput{{CompletedStoryPoints: []*int{intPointer(5)}}},
	}, time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC))

	document := render(t, report)

	assertPDFDocument(t, document)
	for _, want := range []string{
		"Reporte de proyecto",
		"Proyecto Atlas",
		"Proyecto: 11111111-1111-4111-8111-111111111111",
		"Inicio: 2026-01-05    Fin planificado: 2026-12-20",
		"Generado: 2026-11-03 12:00 UTC",
		"Puntos planificados",
		"Puntos completados",
		"Velocidad",
		"Horas estimadas",
		"Horas reales",
		"Desviaci\xf3n",
		"Historias completadas",
		"Defectos detectados",
		"Defectos resueltos",
		"Historias planificadas y completadas",
		"Total: 1; completadas: 1",
		"Tareas y estimaciones",
		"Esfuerzo registrado",
		"Ana P",
		"Implementaci",
		"Defectos relacionados",
		"Fallo de login",
		"Entrega A",
	} {
		if !bytes.Contains(document, []byte(want)) {
			t.Errorf("PDF does not contain %q", want)
		}
	}
	for _, accent := range []string{"M\xe9tricas del \xe1mbito", "Descripci\xf3n", "S\xed", "P\xe1gina 1 de 1"} {
		if !bytes.Contains(document, []byte(accent)) {
			t.Errorf("PDF does not contain the Latin-1 encoded text %q", accent)
		}
	}
}

func TestRenderSprintReportShowsGoalStateAndAttributionNote(t *testing.T) {
	report := reportdomain.Build(reportdomain.Snapshot{
		Project: reportdomain.Project{ID: "project", Name: "Proyecto"},
		Sprint:  &reportdomain.Sprint{ID: "sprint-a", Goal: "Entrega A", Closed: true},
		Stories: []reportdomain.Story{{
			ID: "story-a", Title: "Historia A", StoryPoints: intPointer(3), Completed: true,
		}},
		EffortRecords: []reportdomain.EffortRecord{
			{ID: "effort-direct", StoryID: "story-a", StoryTitle: "Historia A", Hours: 2},
			{ID: "effort-task", StoryID: "story-a", StoryTitle: "Historia A", TaskID: stringPointer("task-a"),
				TaskTitle: stringPointer("Tarea A"), TaskSprintID: stringPointer("sprint-a"), Hours: 4},
		},
	}, time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC))

	document := render(t, report)

	assertPDFDocument(t, document)
	for _, want := range []string{
		"Reporte de Sprint",
		"Sprint: sprint-a",
		"Objetivo: Entrega A",
		"Estado: cerrado",
		"Registros directos a historia no atribuibles al Sprint: 1",
	} {
		if !bytes.Contains(document, []byte(want)) {
			t.Errorf("PDF does not contain %q", want)
		}
	}
}

func TestRenderEmptyScopesUseExplicitEmptyMessages(t *testing.T) {
	document := render(t, reportdomain.Build(reportdomain.Snapshot{
		Project: reportdomain.Project{ID: "project", Name: "Proyecto"},
	}, time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)))

	assertPDFDocument(t, document)
	for _, want := range []string{
		"No disponible",
		"Sin historias en el",
		"Sin tareas descompuestas en el",
		"Sin registros de esfuerzo atribuibles al",
		"Sin defectos relacionados con el",
	} {
		if !bytes.Contains(document, []byte(want)) {
			t.Errorf("PDF does not contain %q", want)
		}
	}
}

func TestRenderLongReportPaginates(t *testing.T) {
	stories := make([]reportdomain.Story, 0, 120)
	for index := 0; index < 120; index++ {
		stories = append(stories, reportdomain.Story{
			ID:          fmt.Sprintf("story-%03d", index),
			Title:       fmt.Sprintf("Historia %03d con un título suficientemente largo para ocupar varias líneas de la tabla", index),
			Priority:    "media",
			Status:      "pendiente",
			StoryPoints: intPointer(index % 8),
		})
	}
	document := render(t, reportdomain.Build(reportdomain.Snapshot{
		Project: reportdomain.Project{ID: "project", Name: "Proyecto"},
		Stories: stories,
	}, time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)))

	assertPDFDocument(t, document)
	if pages := pageCount(t, document); pages < 2 {
		t.Errorf("PDF page count = %d, want at least 2 for a long report", pages)
	}
	if !bytes.Contains(document, []byte("P\xe1gina 2 de ")) {
		t.Error("PDF does not paginate the footer")
	}
}

func render(t *testing.T, report reportdomain.Report) []byte {
	t.Helper()
	document, err := reportpdf.NewRenderer().Render(report)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	return document
}

func assertPDFDocument(t *testing.T, document []byte) {
	t.Helper()
	if !bytes.HasPrefix(document, []byte("%PDF-1.")) {
		t.Errorf("document does not start with a PDF signature: %q", document[:min(len(document), 8)])
	}
	if !bytes.HasSuffix(bytes.TrimSpace(document), []byte("%%EOF")) {
		t.Errorf("document does not end with the PDF trailer")
	}
	if len(document) < 1000 {
		t.Errorf("document size = %d bytes, want a complete PDF", len(document))
	}
	if !bytes.Contains(document, []byte("/Type /Pages")) {
		t.Error("document does not contain a pages tree")
	}
}

func pageCount(t *testing.T, document []byte) int {
	t.Helper()
	matches := regexp.MustCompile(`/Type /Pages[^>]*/Count (\d+)`).FindSubmatch(document)
	if matches == nil {
		matches = regexp.MustCompile(`/Count (\d+)`).FindSubmatch(document)
	}
	if matches == nil {
		t.Fatal("PDF does not expose a page count")
	}
	count, err := strconv.Atoi(string(matches[1]))
	if err != nil {
		t.Fatalf("parse page count %q: %v", matches[1], err)
	}
	return count
}

func intPointer(value int) *int           { return &value }
func floatPointer(value float64) *float64 { return &value }
func stringPointer(value string) *string  { return &value }
