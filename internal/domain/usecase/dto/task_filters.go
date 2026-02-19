package dto

import "time"

type TaskFilters struct {
	Completed *bool
	DueBefore *time.Time
	DueAfter  *time.Time
}
