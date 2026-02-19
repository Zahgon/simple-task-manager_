package tasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/SilentPlaces/simple-task-manager/internal/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		fmt.Println("Skipping integration tests: TEST_DATABASE_URL not set")
		os.Exit(0)
	}

	var err error
	testDB, err = sql.Open("postgres", connStr)
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := testDB.PingContext(ctx); err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	testDB.Close()
	os.Exit(code)
}

func newTestRepo() *PostgresTaskRepository {
	return &PostgresTaskRepository{db: testDB}
}

func seedTask(t *testing.T, repo *PostgresTaskRepository, id string) entities.Task {
	t.Helper()
	dueDate := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	task := entities.Task{
		ID:          id,
		Title:       "Test Task",
		Description: "This is a test",
		Completed:   false,
		DueDate:     &dueDate,
	}
	inserted, err := repo.Add(context.Background(), task)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = testDB.Exec("DELETE FROM tasks WHERE id = $1", id)
	})

	return *inserted
}

func TestAdd(t *testing.T) {
	repo := newTestRepo()
	id := fmt.Sprintf("test-add-%d", time.Now().UnixNano())

	dueDate := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	task := entities.Task{
		ID:          id,
		Title:       "New Task",
		Description: "Integration test",
		Completed:   false,
		DueDate:     &dueDate,
	}

	t.Cleanup(func() {
		_, _ = testDB.Exec("DELETE FROM tasks WHERE id = $1", id)
	})

	inserted, err := repo.Add(context.Background(), task)
	require.NoError(t, err)
	assert.Equal(t, id, inserted.ID)
	assert.Equal(t, "New Task", inserted.Title)
	assert.Equal(t, false, inserted.Completed)
}

func TestGetByID(t *testing.T) {
	repo := newTestRepo()
	id := fmt.Sprintf("test-get-%d", time.Now().UnixNano())
	seeded := seedTask(t, repo, id)

	found, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, seeded.ID, found.ID)
	assert.Equal(t, seeded.Title, found.Title)
}

func TestGetByID_NotFound(t *testing.T) {
	repo := newTestRepo()

	_, err := repo.GetByID(context.Background(), "nonexistent-id")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entities.ErrTaskNotFound))
}

func TestUpdate(t *testing.T) {
	repo := newTestRepo()
	id := fmt.Sprintf("test-update-%d", time.Now().UnixNano())
	seeded := seedTask(t, repo, id)

	seeded.Title = "Updated Title"
	seeded.Completed = true

	updated, err := repo.Update(context.Background(), seeded)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", updated.Title)
	assert.Equal(t, true, updated.Completed)

	found, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", found.Title)
}

func TestUpdate_NotFound(t *testing.T) {
	repo := newTestRepo()

	task := entities.Task{ID: "nonexistent-id", Title: "x", Description: "x"}
	_, err := repo.Update(context.Background(), task)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entities.ErrTaskNotFound))
}

func TestDelete(t *testing.T) {
	repo := newTestRepo()
	id := fmt.Sprintf("test-delete-%d", time.Now().UnixNano())
	seedTask(t, repo, id)

	err := repo.Delete(context.Background(), id)
	assert.NoError(t, err)

	_, err = repo.GetByID(context.Background(), id)
	assert.True(t, errors.Is(err, entities.ErrTaskNotFound))
}

func TestDelete_NotFound(t *testing.T) {
	repo := newTestRepo()

	err := repo.Delete(context.Background(), "nonexistent-id")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entities.ErrTaskNotFound))
}

func TestGetAll(t *testing.T) {
	repo := newTestRepo()

	id1 := fmt.Sprintf("test-getall-1-%d", time.Now().UnixNano())
	id2 := fmt.Sprintf("test-getall-2-%d", time.Now().UnixNano())
	seedTask(t, repo, id1)
	seedTask(t, repo, id2)

	tasks, err := repo.GetAll(context.Background(), nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tasks), 2)
}
