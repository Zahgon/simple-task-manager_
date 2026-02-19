package entities

import "time"

type Task struct {
	ID          string
	Title       string
	Description string
	Completed   bool
	DueDate     *time.Time
}
