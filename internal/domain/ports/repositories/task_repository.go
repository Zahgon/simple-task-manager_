package repositories

import (
	"context"

	"github.com/SilentPlaces/simple-task-manager/internal/domain/entities"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/usecase/dto"
)

type TaskRepository interface {
	GetAll(ctx context.Context, filters *dto.TaskFilters) ([]entities.Task, error)
	GetByID(ctx context.Context, id string) (*entities.Task, error)
	Add(ctx context.Context, task entities.Task) (*entities.Task, error)
	Update(ctx context.Context, task entities.Task) (*entities.Task, error)
	Delete(ctx context.Context, id string) error
}
