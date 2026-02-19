package dto

import (
	"time"

	"github.com/SilentPlaces/simple-task-manager/internal/domain/entities"
)

type TaskResponse struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Completed   bool    `json:"completed"`
	DueDate     *string `json:"due_date,omitempty"`
}

// Maps domain entity to response
func TaskResponseFromEntity(task entities.Task) TaskResponse {
	var dueDate *string
	if task.DueDate != nil {
		formatted := task.DueDate.Format(time.RFC3339)
		dueDate = &formatted
	}
	return TaskResponse{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Completed:   task.Completed,
		DueDate:     dueDate,
	}
}
