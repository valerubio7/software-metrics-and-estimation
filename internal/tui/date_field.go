package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// The three segments of a YYYY-MM-DD date, in display order.
const (
	segYear = iota
	segMonth
	segDay
	numDateSegments
)

var (
	dateSegmentWidths = [numDateSegments]int{4, 2, 2}
	dateSegmentHints  = [numDateSegments]string{"AAAA", "MM", "DD"}
	dateSegmentNames  = [numDateSegments]string{"año", "mes", "día"}
)

// projectLabelWidth aligns the value column across the project form labels; it
// is the widest label, so every row starts its value at the same column.
const projectLabelWidth = len("Fecha fin planificada")

// dateField keeps the digits the user typed for each segment of a calendar
// date. Segments stay unpadded while editing; the value is zero-padded and
// validated only on submit.
type dateField struct {
	Year  string
	Month string
	Day   string
}

func (d dateField) segment(i int) string {
	switch i {
	case segYear:
		return d.Year
	case segMonth:
		return d.Month
	default:
		return d.Day
	}
}

func (d dateField) withSegment(i int, value string) dateField {
	switch i {
	case segYear:
		d.Year = value
	case segMonth:
		d.Month = value
	default:
		d.Day = value
	}
	return d
}

// value returns the zero-padded YYYY-MM-DD string and whether all three
// segments have been filled in.
func (d dateField) value() (string, bool) {
	if d.Year == "" || d.Month == "" || d.Day == "" {
		return "", false
	}
	return fmt.Sprintf("%s-%s-%s", zeroPad(d.Year, 4), zeroPad(d.Month, 2), zeroPad(d.Day, 2)), true
}

// validate returns the normalized date and an empty error, or a human message
// describing what is missing or wrong.
func (d dateField) validate() (string, string) {
	raw, complete := d.value()
	if !complete {
		return "", "está incompleta"
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return "", "no es una fecha válida"
	}
	return parsed.Format("2006-01-02"), ""
}

// typeDigit appends a digit to the focused segment and returns the field and
// the segment to focus next. Full segments auto-advance so the user can type
// "20261016" straight through.
func (d dateField) typeDigit(seg int, digit string) (dateField, int) {
	if len(d.segment(seg)) >= dateSegmentWidths[seg] {
		if seg == numDateSegments-1 {
			return d, seg
		}
		seg++
	}
	current := d.segment(seg) + digit
	d = d.withSegment(seg, current)
	if len(current) >= dateSegmentWidths[seg] && seg < numDateSegments-1 {
		seg++
	}
	return d, seg
}

// backspace removes the last digit of the focused segment, stepping back to the
// previous segment when the current one is already empty.
func (d dateField) backspace(seg int) (dateField, int) {
	if current := d.segment(seg); current != "" {
		return d.withSegment(seg, current[:len(current)-1]), seg
	}
	if seg == segYear {
		return d, seg
	}
	seg--
	if current := d.segment(seg); current != "" {
		d = d.withSegment(seg, current[:len(current)-1])
	}
	return d, seg
}

// projectTextRow renders a labelled text field of the project form, padded so
// the value column lines up with the date rows. The focused row is a full-width
// selection bar; the name carries a trailing cursor.
func (m Model) projectTextRow(field int, label, value string) string {
	cell := padRight(label+":", projectLabelWidth)
	text := cell + " " + value
	if m.formField == field {
		return selectedBarStyle.Render(padToWidth("▸ "+text+"_", m.innerWidth()))
	}
	return rowStyle.Render("  " + text)
}

// projectDateRow renders the segmented year/month/day editor for a date field.
// The focused row paints the whole width as a bar and inverts the active
// segment, so the three sub-fields are always visible and no format has to be
// remembered.
func (m Model) projectDateRow(field int, label string, d dateField) string {
	cell := padRight(label+":", projectLabelWidth)
	focused := m.formField == field
	var segments []string
	for i := 0; i < numDateSegments; i++ {
		raw := d.segment(i)
		empty := raw == ""
		text := zeroPad(raw, dateSegmentWidths[i])
		if empty {
			text = dateSegmentHints[i]
		}
		text = padRight(text, dateSegmentWidths[i])
		segments = append(segments, renderDateSegment(text, empty, focused && m.formSegment == i, focused))
	}
	if !focused {
		return "  " + cell + " " + strings.Join(segments, " ")
	}
	line := selectedBarStyle.Render("▸ "+cell+" ") + strings.Join(segments, selectedBarStyle.Render(" "))
	if pad := m.innerWidth() - lipgloss.Width(line); pad > 0 {
		line += selectedBarStyle.Render(strings.Repeat(" ", pad))
	}
	return line
}

func renderDateSegment(text string, empty, active, focused bool) string {
	switch {
	case focused && active:
		return dateSegmentActiveStyle.Render(" " + text + " ")
	case focused:
		return selectedBarStyle.Render(" " + text + " ")
	case empty:
		return dateSegmentEmptyStyle.Render(" " + text + " ")
	default:
		return dateSegmentIdleStyle.Render(" " + text + " ")
	}
}

// zeroPad left-pads a numeric segment with zeros up to width.
func zeroPad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat("0", width-len(s)) + s
}
