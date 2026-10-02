package domain

import "time"

// ProjectStatus is the date-derived planning state of a project.
type ProjectStatus string

const (
	ProjectStatusPlanned ProjectStatus = "planned"
	ProjectStatusActive  ProjectStatus = "active"
	ProjectStatusOverdue ProjectStatus = "overdue"
)

// DeriveStatus classifies a project against a reference calendar date.
// Only year, month, and day participate; time and location are deliberately ignored.
func (p Project) DeriveStatus(reference time.Time) ProjectStatus {
	date := calendarDate(reference)
	start := calendarDate(p.StartDate)
	finish := calendarDate(p.PlannedFinishDate)
	if date < start {
		return ProjectStatusPlanned
	}
	if date > finish {
		return ProjectStatusOverdue
	}
	return ProjectStatusActive
}

type dateKey int

func calendarDate(value time.Time) dateKey {
	year, month, day := value.Date()
	return dateKey(year*10000 + int(month)*100 + day)
}
