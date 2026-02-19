package tasks

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/SilentPlaces/simple-task-manager/internal/domain/entities"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/ports/repositories"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/usecase/dto"
	"github.com/stretchr/testify/assert"
)

type TaskRepositoryMock struct {
	Data map[string]entities.Task
}

func NewTaskRepositoryMock() repositories.TaskRepository {
	return &TaskRepositoryMock{
		Data: make(map[string]entities.Task),
	}
}

func (m *TaskRepositoryMock) GetAll(ctx context.Context, filters *dto.TaskFilters) ([]entities.Task, error) {
	result := []entities.Task{}

	for _, item := range m.Data {
		result = append(result, item)
	}

	return result, nil
}

func (m *TaskRepositoryMock) GetByID(ctx context.Context, id string) (*entities.Task, error) {
	item, exists := m.Data[id]
	if !exists {
		return nil, entities.ErrTaskNotFound
	}
	return &item, nil
}

func (m *TaskRepositoryMock) Add(ctx context.Context, task entities.Task) (*entities.Task, error) {
	m.Data[task.ID] = task
	return &task, nil
}

func (m *TaskRepositoryMock) Update(ctx context.Context, task entities.Task) (*entities.Task, error) {
	_, exists := m.Data[task.ID]
	if !exists {
		return nil, entities.ErrTaskNotFound
	}
	m.Data[task.ID] = task
	return &task, nil
}

func (m *TaskRepositoryMock) Delete(ctx context.Context, id string) error {
	_, exists := m.Data[id]
	if !exists {
		return entities.ErrTaskNotFound
	}
	delete(m.Data, id)
	return nil
}

func createTestTask(id string) entities.Task {
	dueDate := time.Now().Add(24 * time.Hour)
	return entities.Task{
		ID:          id,
		Title:       "Test Task",
		Description: "This is a test",
		Completed:   false,
		DueDate:     &dueDate,
	}
}

func createTaskList() map[string]entities.Task {
	result := make(map[string]entities.Task)

	for index := range 10 {
		id := fmt.Sprintf("id-%d", index+1)
		result[id] = createTestTask(id)
	}
	return result
}

func setupUseCaseWithTasks(t *testing.T) (*TaskUseCase, []string) {
	t.Helper()
	uc := NewTaskUseCase(NewTaskRepositoryMock())
	var ids []string
	for range 10 {
		added, err := uc.AddTask(context.Background(), createTestTask(""))
		assert.NoError(t, err)
		ids = append(ids, added.ID)
	}
	return uc, ids
}

func TestTaskUseCase_AddTask(t *testing.T) {
	uc := NewTaskUseCase(NewTaskRepositoryMock())

	task := createTestTask("")
	newTask, err := uc.AddTask(context.Background(), task)
	assert.NoError(t, err)
	assert.NotEmpty(t, newTask.ID)
	assert.Equal(t, "Test Task", newTask.Title)

	taskFromRepo, err := uc.GetTask(context.Background(), newTask.ID)
	assert.NoError(t, err)
	assert.Equal(t, newTask.ID, taskFromRepo.ID)
}

func TestTaskUseCase_GetTask(t *testing.T) {
	uc, ids := setupUseCaseWithTasks(t)

	task, err := uc.GetTask(context.Background(), ids[0])
	assert.NoError(t, err)
	assert.Equal(t, ids[0], task.ID)
}

func TestTaskUseCase_GetTask_NotFound(t *testing.T) {
	uc := NewTaskUseCase(NewTaskRepositoryMock())

	_, err := uc.GetTask(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, entities.ErrTaskNotFound)
}

func TestTaskUseCase_GetTasks(t *testing.T) {
	uc, _ := setupUseCaseWithTasks(t)

	allTasks, err := uc.GetTasks(context.Background(), nil)
	assert.NoError(t, err)
	assert.Len(t, allTasks, 10)
}

func TestTaskUseCase_UpdateTask(t *testing.T) {
	uc := NewTaskUseCase(NewTaskRepositoryMock())

	added, err := uc.AddTask(context.Background(), createTestTask(""))
	assert.NoError(t, err)

	added.Title = "Updated Title"
	updatedTask, err := uc.UpdateTask(context.Background(), *added)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Title", updatedTask.Title)
}

func TestTaskUseCase_UpdateTask_NotFound(t *testing.T) {
	uc := NewTaskUseCase(NewTaskRepositoryMock())

	task := createTestTask("nonexistent")
	_, err := uc.UpdateTask(context.Background(), task)
	assert.ErrorIs(t, err, entities.ErrTaskNotFound)
}

func TestTaskUseCase_DeleteTask(t *testing.T) {
	uc, ids := setupUseCaseWithTasks(t)

	err := uc.DeleteTask(context.Background(), ids[0])
	assert.NoError(t, err)

	_, err = uc.GetTask(context.Background(), ids[0])
	assert.ErrorIs(t, err, entities.ErrTaskNotFound)
}

func TestTaskUseCase_DeleteTask_NotFound(t *testing.T) {
	uc := NewTaskUseCase(NewTaskRepositoryMock())

	err := uc.DeleteTask(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, entities.ErrTaskNotFound)
}
