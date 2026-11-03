// Package pdf renders report values into a downloadable PDF document.
package pdf

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/go-pdf/fpdf"

	metricsdomain "github.com/valerubio7/software-metrics-and-estimation/internal/metrics/domain"
	"github.com/valerubio7/software-metrics-and-estimation/internal/report/domain"
)

const (
	pageMargin    = 15.0
	bottomMargin  = 18.0
	tableFontSize = 8.0
	tableLine     = 4.0
	tablePadding  = 1.2

	headerFillRed   = 232
	headerFillGreen = 237
	headerFillBlue  = 243
	accentRed       = 25
	accentGreen     = 55
	accentBlue      = 90
)

type translator func(string) string

type column struct {
	title string
	width float64
	align string
}

// Renderer produces a PDF from a report. It has no state and is safe to reuse.
type Renderer struct{}

// NewRenderer creates the default PDF renderer.
func NewRenderer() *Renderer { return &Renderer{} }

// Render draws every report section and returns the complete PDF bytes.
func (r *Renderer) Render(report domain.Report) ([]byte, error) {
	doc := fpdf.New("P", "mm", "A4", "")
	doc.SetMargins(pageMargin, pageMargin, pageMargin)
	doc.SetAutoPageBreak(true, bottomMargin)
	doc.SetCompression(false)

	title := reportTitle(report)
	doc.SetTitle(title, true)
	doc.SetAuthor("Software Metrics and Estimation", true)
	doc.SetCreator("Software Metrics and Estimation", true)
	doc.SetSubject("Reporte de proyecto o Sprint", true)
	doc.AliasNbPages("{nb}")
	tr := doc.UnicodeTranslatorFromDescriptor("")
	doc.SetFooterFunc(func() {
		doc.SetY(-15)
		doc.SetFont("Helvetica", "I", 8)
		doc.SetTextColor(100, 100, 100)
		doc.CellFormat(0, 10, tr(fmt.Sprintf("Página %d de {nb}", doc.PageNo())), "", 0, "C", false, 0, "")
		doc.SetTextColor(0, 0, 0)
	})

	doc.AddPage()

	writeHeader(doc, tr, report)
	writeMetrics(doc, tr, report)
	writeStories(doc, tr, report)
	writeTasks(doc, tr, report)
	writeEffort(doc, tr, report)
	writeDefects(doc, tr, report)

	var buffer bytes.Buffer
	if err := doc.Output(&buffer); err != nil {
		return nil, fmt.Errorf("render report PDF: %w", err)
	}
	return buffer.Bytes(), nil
}

func reportTitle(report domain.Report) string {
	if report.Sprint != nil {
		return "Reporte de Sprint"
	}
	return "Reporte de proyecto"
}

func writeHeader(doc *fpdf.Fpdf, tr translator, report domain.Report) {
	doc.SetTextColor(accentRed, accentGreen, accentBlue)
	doc.SetFont("Helvetica", "B", 18)
	doc.Cell(0, 10, tr(reportTitle(report)))
	doc.Ln(10)

	doc.SetTextColor(20, 20, 20)
	doc.SetFont("Helvetica", "B", 12)
	doc.Cell(0, 6, tr(report.Project.Name))
	doc.Ln(6)

	doc.SetFont("Helvetica", "", 9)
	doc.Cell(0, 5, tr("Proyecto: "+report.Project.ID))
	doc.Ln(5)
	doc.Cell(0, 5, tr("Inicio: "+formatDate(report.Project.StartDate)+"    Fin planificado: "+formatDate(report.Project.PlannedFinishDate)))
	doc.Ln(5)

	if report.Sprint != nil {
		doc.Cell(0, 5, tr("Sprint: "+report.Sprint.ID))
		doc.Ln(5)
		doc.MultiCell(0, 5, tr("Objetivo: "+report.Sprint.Goal), "", "L", false)
		doc.Cell(0, 5, tr("Estado: "+sprintState(report.Sprint.Closed)))
		doc.Ln(5)
	}

	doc.SetFont("Helvetica", "I", 8)
	doc.SetTextColor(90, 90, 90)
	doc.Cell(0, 5, tr("Generado: "+report.GeneratedAt.UTC().Format("2006-01-02 15:04 UTC")))
	doc.Ln(5)

	doc.SetDrawColor(accentRed, accentGreen, accentBlue)
	doc.SetLineWidth(0.5)
	width, _ := doc.GetPageSize()
	doc.Line(pageMargin, doc.GetY(), width-pageMargin, doc.GetY())
	doc.Ln(2)
	doc.SetTextColor(0, 0, 0)
}

func writeMetrics(doc *fpdf.Fpdf, tr translator, report domain.Report) {
	writeSection(doc, tr, "Métricas del ámbito")
	coverage := report.Metrics.Coverage
	rows := [][]string{
		{"Puntos planificados", formatFloat(report.Metrics.StoryPointsPlanned), missingObservations(coverage.StoryPointsPlanned.MissingObservations)},
		{"Puntos completados", formatFloat(report.Metrics.StoryPointsCompleted), missingObservations(coverage.StoryPointsCompleted.MissingObservations)},
		{"Velocidad", formatFloat(report.Metrics.Velocity), missingObservations(coverage.Velocity.MissingObservations)},
		{"Horas estimadas", formatFloat(report.Metrics.EstimatedHours), missingObservations(coverage.EstimatedHours.MissingObservations)},
		{"Horas reales", formatFloat(report.Metrics.ActualHours), actualHoursCoverage(coverage.ActualHours)},
		{"Desviación (horas)", formatFloat(report.Metrics.Deviation), "—"},
		{"Desviación (%)", formatPercent(report.Metrics.DeviationPercentage), "—"},
		{"Historias completadas (%)", formatPercent(report.Metrics.CompletedStoriesPercentage), "—"},
		{"Defectos detectados", strconv.Itoa(report.Metrics.DefectsDetected), "—"},
		{"Defectos resueltos", strconv.Itoa(report.Metrics.DefectsResolved), "—"},
	}
	writeTable(doc, tr, []column{
		{title: "Métrica", width: 65, align: "L"},
		{title: "Valor", width: 50, align: "R"},
		{title: "Cobertura", width: 65, align: "L"},
	}, rows)
}

func writeStories(doc *fpdf.Fpdf, tr translator, report domain.Report) {
	writeSection(doc, tr, "Historias planificadas y completadas")
	if len(report.Stories) == 0 {
		writeEmpty(doc, tr, "Sin historias en el ámbito.")
		return
	}
	completed := 0
	rows := make([][]string, 0, len(report.Stories))
	for _, story := range report.Stories {
		completion := "No"
		if story.Completed {
			completed++
			completion = "Sí"
			if story.CompletedAt != nil {
				completion += " (" + formatDate(*story.CompletedAt) + ")"
			}
		}
		rows = append(rows, []string{
			story.Title,
			story.Priority,
			story.Status,
			formatInt(story.StoryPoints),
			formatFloat(story.EstimatedHours),
			completion,
		})
	}
	doc.SetFont("Helvetica", "", 9)
	doc.Cell(0, 5, tr(fmt.Sprintf("Total: %d; completadas: %d", len(report.Stories), completed)))
	doc.Ln(7)
	writeTable(doc, tr, []column{
		{title: "Historia", width: 58, align: "L"},
		{title: "Prioridad", width: 18, align: "L"},
		{title: "Estado", width: 26, align: "L"},
		{title: "Puntos", width: 16, align: "R"},
		{title: "Horas est.", width: 28, align: "R"},
		{title: "Completada", width: 34, align: "L"},
	}, rows)
}

func writeTasks(doc *fpdf.Fpdf, tr translator, report domain.Report) {
	writeSection(doc, tr, "Tareas y estimaciones")
	rows := make([][]string, 0)
	for _, story := range report.Stories {
		for _, task := range story.Tasks {
			rows = append(rows, []string{story.Title, task.Title, formatFloat(task.EstimatedHours)})
		}
	}
	if len(rows) == 0 {
		writeEmpty(doc, tr, "Sin tareas descompuestas en el ámbito.")
		return
	}
	writeTable(doc, tr, []column{
		{title: "Historia", width: 65, align: "L"},
		{title: "Tarea", width: 80, align: "L"},
		{title: "Horas estimadas", width: 35, align: "R"},
	}, rows)
}

func writeEffort(doc *fpdf.Fpdf, tr translator, report domain.Report) {
	writeSection(doc, tr, "Esfuerzo registrado")
	if len(report.EffortRecords) == 0 {
		writeEmpty(doc, tr, "Sin registros de esfuerzo atribuibles al ámbito.")
	} else {
		rows := make([][]string, 0, len(report.EffortRecords))
		for _, record := range report.EffortRecords {
			task := "—"
			if record.TaskTitle != nil {
				task = *record.TaskTitle
			}
			rows = append(rows, []string{
				formatDate(record.WorkDate),
				record.MemberName,
				record.StoryTitle,
				task,
				formatHours(record.Hours),
				record.Description,
			})
		}
		writeTable(doc, tr, []column{
			{title: "Fecha", width: 20, align: "L"},
			{title: "Integrante", width: 35, align: "L"},
			{title: "Historia", width: 42, align: "L"},
			{title: "Tarea", width: 30, align: "L"},
			{title: "Horas", width: 14, align: "R"},
			{title: "Descripción", width: 39, align: "L"},
		}, rows)
	}
	if report.Sprint != nil && report.Metrics.Coverage.ActualHours.UnattributedStoryLevelRecords > 0 {
		writeNote(doc, tr, fmt.Sprintf(
			"Registros directos a historia no atribuibles al Sprint: %d. Se informan en la cobertura de métricas.",
			report.Metrics.Coverage.ActualHours.UnattributedStoryLevelRecords,
		))
	}
}

func writeDefects(doc *fpdf.Fpdf, tr translator, report domain.Report) {
	writeSection(doc, tr, "Defectos relacionados")
	if len(report.Defects) == 0 {
		writeEmpty(doc, tr, "Sin defectos relacionados con el ámbito.")
		return
	}
	rows := make([][]string, 0, len(report.Defects))
	for _, defect := range report.Defects {
		resolution := "—"
		if defect.ResolutionSprintGoal != nil {
			resolution = *defect.ResolutionSprintGoal
		}
		rows = append(rows, []string{
			defect.Description,
			defect.Severity,
			defect.Status,
			defect.StoryTitle,
			defect.DetectionSprintGoal,
			resolution,
		})
	}
	writeTable(doc, tr, []column{
		{title: "Descripción", width: 48, align: "L"},
		{title: "Severidad", width: 16, align: "L"},
		{title: "Estado", width: 22, align: "L"},
		{title: "Historia", width: 34, align: "L"},
		{title: "Sprint detección", width: 30, align: "L"},
		{title: "Sprint resolución", width: 30, align: "L"},
	}, rows)
}

func writeSection(doc *fpdf.Fpdf, tr translator, title string) {
	ensureSpace(doc, 18)
	doc.Ln(3)
	doc.SetTextColor(accentRed, accentGreen, accentBlue)
	doc.SetFont("Helvetica", "B", 12)
	doc.Cell(0, 7, tr(title))
	doc.Ln(7)
	doc.SetTextColor(0, 0, 0)
	doc.SetFont("Helvetica", "", 9)
}

func writeEmpty(doc *fpdf.Fpdf, tr translator, text string) {
	doc.SetFont("Helvetica", "I", 9)
	doc.MultiCell(0, 5, tr(text), "", "L", false)
	doc.Ln(2)
	doc.SetFont("Helvetica", "", 9)
}

func writeNote(doc *fpdf.Fpdf, tr translator, text string) {
	ensureSpace(doc, 12)
	doc.SetFont("Helvetica", "I", 8)
	doc.SetTextColor(90, 90, 90)
	doc.MultiCell(0, 4, tr(text), "", "L", false)
	doc.SetTextColor(0, 0, 0)
	doc.SetFont("Helvetica", "", 9)
	doc.Ln(2)
}

func writeTable(doc *fpdf.Fpdf, tr translator, columns []column, rows [][]string) {
	doc.SetFont("Helvetica", "B", tableFontSize)
	headerHeight := tableRowHeight(doc, tr, columns, columnTitles(columns))
	ensureSpace(doc, headerHeight)
	drawTableRow(doc, tr, columns, columnTitles(columns), headerHeight, true)

	doc.SetFont("Helvetica", "", tableFontSize)
	for _, row := range rows {
		height := tableRowHeight(doc, tr, columns, row)
		ensureSpace(doc, height)
		drawTableRow(doc, tr, columns, row, height, false)
	}
	doc.SetFont("Helvetica", "", 9)
}

func columnTitles(columns []column) []string {
	titles := make([]string, len(columns))
	for index, column := range columns {
		titles[index] = column.title
	}
	return titles
}

func tableRowHeight(doc *fpdf.Fpdf, tr translator, columns []column, cells []string) float64 {
	lines := 1
	for index, cell := range cells {
		if index >= len(columns) {
			continue
		}
		count := len(doc.SplitLines([]byte(tr(cell)), columns[index].width-2*tablePadding))
		if count > lines {
			lines = count
		}
	}
	return float64(lines)*tableLine + 2*tablePadding
}

func drawTableRow(doc *fpdf.Fpdf, tr translator, columns []column, cells []string, height float64, header bool) {
	y := doc.GetY()
	x := pageMargin
	if header {
		doc.SetFillColor(headerFillRed, headerFillGreen, headerFillBlue)
		doc.SetDrawColor(180, 185, 190)
		doc.SetTextColor(20, 20, 20)
	} else {
		doc.SetDrawColor(205, 210, 215)
		doc.SetTextColor(20, 20, 20)
	}
	for index, column := range columns {
		cell := ""
		if index < len(cells) {
			cell = cells[index]
		}
		fill := "D"
		if header {
			fill = "DF"
		}
		doc.Rect(x, y, column.width, height, fill)
		doc.SetXY(x+tablePadding, y+tablePadding)
		doc.MultiCell(column.width-2*tablePadding, tableLine, tr(cell), "", column.align, false)
		x += column.width
	}
	doc.SetY(y + height)
	doc.SetX(pageMargin)
}

func ensureSpace(doc *fpdf.Fpdf, needed float64) {
	_, pageHeight := doc.GetPageSize()
	if doc.GetY()+needed > pageHeight-bottomMargin {
		doc.AddPage()
	}
}

func formatDate(value time.Time) string {
	if value.IsZero() {
		return "—"
	}
	return value.Format("2006-01-02")
}

func sprintState(closed bool) string {
	if closed {
		return "cerrado"
	}
	return "abierto"
}

func formatInt(value *int) string {
	if value == nil {
		return "No disponible"
	}
	return strconv.Itoa(*value)
}

func formatFloat(value *float64) string {
	if value == nil {
		return "No disponible"
	}
	return strconv.FormatFloat(*value, 'f', -1, 64)
}

func formatPercent(value *float64) string {
	if value == nil {
		return "No disponible"
	}
	return strconv.FormatFloat(*value, 'f', -1, 64) + " %"
}

func formatHours(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func missingObservations(count int) string {
	if count == 0 {
		return "Completa"
	}
	return fmt.Sprintf("Faltan %d observaciones", count)
}

func actualHoursCoverage(coverage metricsdomain.ActualHoursCoverage) string {
	if coverage.StoriesWithoutRecords == 0 && coverage.UnattributedStoryLevelRecords == 0 {
		return "Completa"
	}
	return fmt.Sprintf("Historias sin registros: %d; registros directos no atribuibles: %d",
		coverage.StoriesWithoutRecords, coverage.UnattributedStoryLevelRecords)
}
