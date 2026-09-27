package domain

import (
	"cmp"
	"slices"
)

// Backlog is the Product Backlog of a project: its stories in planning order.
type Backlog struct {
	ProjectID string
	Stories   []Story
}

// NewBacklog orders a copy of stories by priority precedence. The sort is stable:
// stories with the same priority keep the order in which they were received, so the
// creation order supplied by the caller acts as the tie-breaker.
func NewBacklog(projectID string, stories []Story) Backlog {
	ordered := slices.Clone(stories)
	slices.SortStableFunc(ordered, func(a, b Story) int {
		return cmp.Compare(priorityRank(a.Priority), priorityRank(b.Priority))
	})
	return Backlog{ProjectID: projectID, Stories: ordered}
}

// priorityRank returns the position of priority in AllowedPriorities. An unknown
// priority ranks after every known one.
func priorityRank(priority string) int {
	allowed := AllowedPriorities()
	if rank := slices.Index(allowed, priority); rank >= 0 {
		return rank
	}
	return len(allowed)
}
