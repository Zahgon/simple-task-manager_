package tasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SilentPlaces/simple-task-manager/internal/adapters/database"
	internaldb "github.com/SilentPlaces/simple-task-manager/internal/database"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/entities"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/ports/repositories"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/usecase/dto"
)

type PostgresTaskRepository struct {
	db *sql.DB
}

func NewPostgresTaskRepository(db *sql.DB) repositories.TaskRepository {
	return &PostgresTaskRepository{db: db}
}

func (r *PostgresTaskRepository) executor(ctx context.Context) internaldb.Executor {
	return database.NewMetricsExecutor(internaldb.GetExecutor(ctx, r.db), "tasks")
}

func (r *PostgresTaskRepository) GetAll(ctx context.Context, filters *dto.TaskFilters) ([]entities.Task, error) {
	timeOutContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := "SELECT id, title, description, completed, due_date FROM tasks"
	var conditions []string
	var args []any

	if filters != nil {
		if filters.Completed != nil {
			args = append(args, *filters.Completed)
			conditions = append(conditions, fmt.Sprintf("completed = $%d", len(args)))
		}
		if filters.DueBefore != nil {
			args = append(args, *filters.DueBefore)
			conditions = append(conditions, fmt.Sprintf("due_date < $%d", len(args)))
		}
		if filters.DueAfter != nil {
			args = append(args, *filters.DueAfter)
			conditions = append(conditions, fmt.Sprintf("due_date > $%d", len(args)))
		}
		if len(conditions) > 0 {
			query += " WHERE " + strings.Join(conditions, " AND ")
		}
	}

	rows, err := r.executor(ctx).QueryContext(timeOutContext, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying tasks: %w", err)
	}
	defer rows.Close()

	var tasks []entities.Task
	for rows.Next() {
		var task entities.Task
		if err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Completed, &task.DueDate); err != nil {
			return nil, fmt.Errorf("scanning task row: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating task rows: %w", err)
	}

	if tasks == nil {
		tasks = []entities.Task{}
	}
	return tasks, nil
}

func (r *PostgresTaskRepository) GetByID(ctx context.Context, id string) (*entities.Task, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := "SELECT id, title, description, completed, due_date FROM tasks WHERE id = $1"
	row := r.executor(ctx).QueryRowContext(timeoutCtx, query, id)

	var task entities.Task
	if err := row.Scan(&task.ID, &task.Title, &task.Description, &task.Completed, &task.DueDate); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("task %s: %w", id, entities.ErrTaskNotFound)
		}
		return nil, fmt.Errorf("scanning task %s: %w", id, err)
	}
	return &task, nil
}

func (r *PostgresTaskRepository) Add(ctx context.Context, task entities.Task) (*entities.Task, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `INSERT INTO tasks (id, title, description, completed, due_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, title, description, completed, due_date`

	row := r.executor(ctx).QueryRowContext(timeoutCtx, query, task.ID, task.Title, task.Description, task.Completed, task.DueDate)

	var inserted entities.Task
	if err := row.Scan(&inserted.ID, &inserted.Title, &inserted.Description, &inserted.Completed, &inserted.DueDate); err != nil {
		return nil, fmt.Errorf("inserting task %s: %w", task.ID, err)
	}
	return &inserted, nil
}

func (r *PostgresTaskRepository) Update(ctx context.Context, task entities.Task) (*entities.Task, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `UPDATE tasks SET title = $1, description = $2, completed = $3, due_date = $4
		WHERE id = $5
		RETURNING id, title, description, completed, due_date`

	row := r.executor(ctx).QueryRowContext(timeoutCtx, query, task.Title, task.Description, task.Completed, task.DueDate, task.ID)

	var updated entities.Task
	if err := row.Scan(&updated.ID, &updated.Title, &updated.Description, &updated.Completed, &updated.DueDate); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("task %s: %w", task.ID, entities.ErrTaskNotFound)
		}
		return nil, fmt.Errorf("updating task %s: %w", task.ID, err)
	}
	return &updated, nil
}

func (r *PostgresTaskRepository) Delete(ctx context.Context, id string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := "DELETE FROM tasks WHERE id = $1"
	result, err := r.executor(ctx).ExecContext(timeoutCtx, query, id)
	if err != nil {
		return fmt.Errorf("deleting task %s: %w", id, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking delete result for task %s: %w", id, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("task %s: %w", id, entities.ErrTaskNotFound)
	}
	return nil
}
