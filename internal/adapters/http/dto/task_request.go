package dto

import (
	"time"

	"github.com/SilentPlaces/simple-task-manager/internal/domain/entities"
)

type CreateTaskRequest struct {
	Title       string     `json:"title" binding:"required,min=3,max=255"`
	Description string     `json:"description" binding:"required,min=3,max=1024"`
	DueDate     *time.Time `json:"due_date"`
}

func (r CreateTaskRequest) ToEntity() entities.Task {
	return entities.Task{
		Title:       r.Title,
		Description: r.Description,
		DueDate:     r.DueDate,
		Completed:   false,
	}
}

type UpdateTaskRequest struct {
	Title       string     `json:"title" binding:"required,min=3,max=255"`
	Description string     `json:"description" binding:"required,min=3,max=1024"`
	DueDate     *time.Time `json:"due_date"`
	Completed   bool       `json:"completed"`
}

func (r UpdateTaskRequest) ToEntity(id string) entities.Task {
	return entities.Task{
		ID:          id,
		Title:       r.Title,
		Description: r.Description,
		DueDate:     r.DueDate,
		Completed:   r.Completed,
	}
}
