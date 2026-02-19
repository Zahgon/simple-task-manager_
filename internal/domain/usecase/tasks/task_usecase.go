package tasks

import (
	"context"

	"github.com/google/uuid"

	"github.com/SilentPlaces/simple-task-manager/internal/domain/entities"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/ports/repositories"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/usecase/dto"
)

type TaskUseCase struct {
	repository repositories.TaskRepository
}

func NewTaskUseCase(repository repositories.TaskRepository) *TaskUseCase {
	return &TaskUseCase{repository: repository}
}

func (uc *TaskUseCase) GetTasks(ctx context.Context, filters *dto.TaskFilters) ([]entities.Task, error) {
	return uc.repository.GetAll(ctx, filters)
}

func (uc *TaskUseCase) GetTask(ctx context.Context, id string) (*entities.Task, error) {
	return uc.repository.GetByID(ctx, id)
}

func (uc *TaskUseCase) AddTask(ctx context.Context, task entities.Task) (*entities.Task, error) {
	task.ID = uuid.New().String()
	return uc.repository.Add(ctx, task)
}

func (uc *TaskUseCase) UpdateTask(ctx context.Context, task entities.Task) (*entities.Task, error) {
	return uc.repository.Update(ctx, task)
}

func (uc *TaskUseCase) DeleteTask(ctx context.Context, id string) error {
	return uc.repository.Delete(ctx, id)
}
