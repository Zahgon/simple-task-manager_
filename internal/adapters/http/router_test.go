package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SilentPlaces/simple-task-manager/internal/adapters/http/handlers"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/entities"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/ports/logger"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/ports/repositories"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/usecase/dto"
	usecasetasks "github.com/SilentPlaces/simple-task-manager/internal/domain/usecase/tasks"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const seededTaskID = "router-test-task"

type TaskRepositoryStub struct {
	Data map[string]entities.Task
}

func NewTaskRepositoryStub() repositories.TaskRepository {
	due := time.Date(2027, 1, 15, 10, 0, 0, 0, time.UTC)
	return &TaskRepositoryStub{
		Data: map[string]entities.Task{
			seededTaskID: {
				ID:          seededTaskID,
				Title:       "Alpha task",
				Description: "First seeded task",
				Completed:   false,
				DueDate:     &due,
			},
		},
	}
}

func (s *TaskRepositoryStub) GetAll(ctx context.Context, filters *dto.TaskFilters) ([]entities.Task, error) {
	result := []entities.Task{}
	for _, item := range s.Data {
		result = append(result, item)
	}
	return result, nil
}

func (s *TaskRepositoryStub) GetByID(ctx context.Context, id string) (*entities.Task, error) {
	item, exists := s.Data[id]
	if !exists {
		return nil, fmt.Errorf("task %s: %w", id, entities.ErrTaskNotFound)
	}
	return &item, nil
}

func (s *TaskRepositoryStub) Add(ctx context.Context, task entities.Task) (*entities.Task, error) {
	s.Data[task.ID] = task
	return &task, nil
}

func (s *TaskRepositoryStub) Update(ctx context.Context, task entities.Task) (*entities.Task, error) {
	if _, exists := s.Data[task.ID]; !exists {
		return nil, fmt.Errorf("task %s: %w", task.ID, entities.ErrTaskNotFound)
	}
	s.Data[task.ID] = task
	return &task, nil
}

func (s *TaskRepositoryStub) Delete(ctx context.Context, id string) error {
	if _, exists := s.Data[id]; !exists {
		return fmt.Errorf("task %s: %w", id, entities.ErrTaskNotFound)
	}
	delete(s.Data, id)
	return nil
}

type LoggerStub struct{}

func (LoggerStub) Info(msg string, keysAndValues ...any)  {}
func (LoggerStub) Error(msg string, keysAndValues ...any) {}
func (LoggerStub) Warn(msg string, keysAndValues ...any)  {}
func (LoggerStub) Debug(msg string, keysAndValues ...any) {}
func (LoggerStub) Fatal(msg string, keysAndValues ...any) {}
func (l LoggerStub) With(keysAndValues ...any) logger.Logger {
	return l
}

func newTestRouter() http.Handler {
	repo := NewTaskRepositoryStub()
	useCase := usecasetasks.NewTaskUseCase(repo)
	handler := handlers.NewTaskHandler(useCase, LoggerStub{})
	return NewRouter(handler, LoggerStub{})
}

func do(t *testing.T, method, target string, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	if body == "" {
		req.Body = http.NoBody
		req.ContentLength = 0
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	newTestRouter().ServeHTTP(rec, req)
	return rec
}

// An unmatched route answers with a bare "404 page not found", a "text/plain"
// content type carrying no charset, and no nosniff header.
func TestNotFoundMatchesInheritedDefault(t *testing.T) {
	for _, target := range []string{"/", "/unknown", "/TASKS", "//tasks", "/tasks/" + seededTaskID + "/extra"} {
		rec := do(t, http.MethodGet, target, "", nil)
		assert.Equal(t, http.StatusNotFound, rec.Code, target)
		assert.Equal(t, "text/plain", rec.Header().Get("Content-Type"), target)
		assert.Equal(t, "404 page not found", rec.Body.String(), target)
		assert.Empty(t, rec.Header().Get("X-Content-Type-Options"), target)
	}
}

// The inherited engine left method-not-allowed handling off, so a wrong method
// is a 404 with no Allow header rather than the router's native 405.
func TestMethodMismatchIsNotFoundNot405(t *testing.T) {
	cases := []struct {
		method string
		target string
	}{
		{http.MethodPatch, "/tasks/" + seededTaskID},
		{http.MethodPost, "/tasks/" + seededTaskID},
		{http.MethodPut, "/tasks"},
		{http.MethodDelete, "/tasks"},
		{http.MethodHead, "/tasks"},
		{http.MethodOptions, "/tasks"},
		{http.MethodPost, "/metrics"},
	}
	for _, c := range cases {
		rec := do(t, c.method, c.target, "", nil)
		label := c.method + " " + c.target
		assert.Equal(t, http.StatusNotFound, rec.Code, label)
		assert.Empty(t, rec.Header().Get("Allow"), label)
		assert.Equal(t, "text/plain", rec.Header().Get("Content-Type"), label)
	}
}

func TestTrailingSlashRedirects(t *testing.T) {
	cases := []struct {
		method   string
		target   string
		code     int
		location string
	}{
		{http.MethodGet, "/tasks/", http.StatusMovedPermanently, "/tasks"},
		{http.MethodGet, "/tasks/" + seededTaskID + "/", http.StatusMovedPermanently, "/tasks/" + seededTaskID},
		{http.MethodGet, "/metrics/", http.StatusMovedPermanently, "/metrics"},
		{http.MethodDelete, "/tasks/" + seededTaskID + "/", http.StatusTemporaryRedirect, "/tasks/" + seededTaskID},
		{http.MethodPost, "/tasks/", http.StatusTemporaryRedirect, "/tasks"},
	}
	for _, c := range cases {
		rec := do(t, c.method, c.target, "", nil)
		label := c.method + " " + c.target
		assert.Equal(t, c.code, rec.Code, label)
		assert.Equal(t, c.location, rec.Header().Get("Location"), label)
		if c.method == http.MethodGet {
			assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"), label)
			assert.Equal(t,
				`<a href="`+c.location+`">Moved Permanently</a>.`+"\n\n",
				rec.Body.String(), label)
		} else {
			assert.Empty(t, rec.Body.String(), label)
		}
	}
}

// A trailing slash only redirects when the sibling path is itself routable for
// the same method; everything else stays a 404.
func TestTrailingSlashDoesNotRedirectUnroutablePaths(t *testing.T) {
	for _, target := range []string{"/unknown/", "/"} {
		rec := do(t, http.MethodGet, target, "", nil)
		assert.Equal(t, http.StatusNotFound, rec.Code, target)
		assert.Empty(t, rec.Header().Get("Location"), target)
	}
}

// Routing happens on the decoded path, so an encoded slash is indistinguishable
// from a real one: "/tasks%2Fid" reaches GetTask and "/tasks/a%2Fb" is a
// three-segment path that matches nothing.
func TestRoutingUsesDecodedPath(t *testing.T) {
	rec := do(t, http.MethodGet, "/tasks%2F"+seededTaskID, "", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))

	rec = do(t, http.MethodGet, "/tasks/a%2Fb", "", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "text/plain", rec.Header().Get("Content-Type"))

	rec = do(t, http.MethodGet, "/tasks/se%65ded", "", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Equal(t, `{"error":"task seeded: task not found"}`, rec.Body.String())
}

func TestGetTaskOK(t *testing.T) {
	rec := do(t, http.MethodGet, "/tasks/"+seededTaskID, "", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Equal(t,
		`{"id":"router-test-task","title":"Alpha task","description":"First seeded task","completed":false,"due_date":"2027-01-15T10:00:00Z"}`,
		rec.Body.String())
}

// json.Marshal is used rather than an encoder, so no newline is appended.
func TestJSONBodiesCarryNoTrailingNewline(t *testing.T) {
	for _, target := range []string{"/tasks", "/tasks/" + seededTaskID, "/tasks/missing"} {
		rec := do(t, http.MethodGet, target, "", nil)
		assert.False(t, strings.HasSuffix(rec.Body.String(), "\n"), target)
		assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"), target)
	}
}

func TestGetTasksOK(t *testing.T) {
	rec := do(t, http.MethodGet, "/tasks", "", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t,
		`[{"id":"router-test-task","title":"Alpha task","description":"First seeded task","completed":false,"due_date":"2027-01-15T10:00:00Z"}]`,
		rec.Body.String())
}

func TestGetTasksInvalidDateFilters(t *testing.T) {
	rec := do(t, http.MethodGet, "/tasks?dueAfter=notatime", "", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, `{"error":"invalid dueAfter: expected RFC3339 format"}`, rec.Body.String())

	rec = do(t, http.MethodGet, "/tasks?dueBefore=notatime", "", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, `{"error":"invalid dueBefore: expected RFC3339 format"}`, rec.Body.String())
}

// An unparseable "completed" value is silently read as false rather than
// rejected, and unknown parameters are ignored outright.
func TestGetTasksTolerantQueryParsing(t *testing.T) {
	for _, target := range []string{"/tasks?completed=garbage", "/tasks?completed=", "/tasks?unknownParam=1"} {
		rec := do(t, http.MethodGet, target, "", nil)
		assert.Equal(t, http.StatusOK, rec.Code, target)
	}
}

func TestGetTaskNotFound(t *testing.T) {
	rec := do(t, http.MethodGet, "/tasks/does-not-exist", "", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Equal(t, `{"error":"task does-not-exist: task not found"}`, rec.Body.String())
}

func TestAddTaskCreated(t *testing.T) {
	rec := do(t, http.MethodPost, "/tasks",
		`{"title":"Created task","description":"Created by the router test"}`, nil)
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), `"title":"Created task"`)
	assert.NotContains(t, rec.Body.String(), `"due_date"`)
}

// The request content type is never consulted; the body is always JSON decoded.
func TestAddTaskIgnoresContentType(t *testing.T) {
	body := `{"title":"Created task","description":"Created by the router test"}`
	for _, ct := range []string{"text/plain", ""} {
		headers := map[string]string{}
		if ct != "" {
			headers["Content-Type"] = ct
		}
		rec := do(t, http.MethodPost, "/tasks", body, headers)
		assert.Equal(t, http.StatusCreated, rec.Code, ct)
	}

	rec := do(t, http.MethodPost, "/tasks", "title=Created+task",
		map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, `{"error":"invalid character 'i' in literal true (expecting 'r')"}`, rec.Body.String())
}

// The binding tags are surfaced verbatim through validator.ValidationErrors.
func TestAddTaskValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "missing title",
			body: `{"description":"Created by the router test"}`,
			want: `{"error":"Key: 'CreateTaskRequest.Title' Error:Field validation for 'Title' failed on the 'required' tag"}`,
		},
		{
			name: "title too short",
			body: `{"title":"ab","description":"Created by the router test"}`,
			want: `{"error":"Key: 'CreateTaskRequest.Title' Error:Field validation for 'Title' failed on the 'min' tag"}`,
		},
		{
			name: "missing description",
			body: `{"title":"Created task"}`,
			want: `{"error":"Key: 'CreateTaskRequest.Description' Error:Field validation for 'Description' failed on the 'required' tag"}`,
		},
		{
			name: "title too long",
			body: `{"title":"` + strings.Repeat("a", 256) + `","description":"Created by the router test"}`,
			want: `{"error":"Key: 'CreateTaskRequest.Title' Error:Field validation for 'Title' failed on the 'max' tag"}`,
		},
	}
	for _, c := range cases {
		rec := do(t, http.MethodPost, "/tasks", c.body, nil)
		assert.Equal(t, http.StatusBadRequest, rec.Code, c.name)
		assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"), c.name)
		assert.Equal(t, c.want, rec.Body.String(), c.name)
	}
}

// encoding/json decode failures are reported with the decoder's own wording.
func TestAddTaskDecodeErrors(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "malformed json",
			body: `{"title":"Created task"`,
			want: `{"error":"unexpected EOF"}`,
		},
		{
			name: "empty body",
			body: "",
			want: `{"error":"EOF"}`,
		},
		{
			name: "wrong field type",
			body: `{"title":123,"description":"Created by the router test"}`,
			want: `{"error":"json: cannot unmarshal number into Go struct field CreateTaskRequest.title of type string"}`,
		},
		{
			name: "array body",
			body: `[]`,
			want: `{"error":"json: cannot unmarshal array into Go value of type dto.CreateTaskRequest"}`,
		},
		{
			name: "bad due date",
			body: `{"title":"Created task","description":"Created by the router test","due_date":"not-a-date"}`,
			want: `{"error":"parsing time \"not-a-date\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"not-a-date\" as \"2006\""}`,
		},
	}
	for _, c := range cases {
		rec := do(t, http.MethodPost, "/tasks", c.body, nil)
		assert.Equal(t, http.StatusBadRequest, rec.Code, c.name)
		assert.Equal(t, c.want, rec.Body.String(), c.name)
	}
}

func TestUpdateTask(t *testing.T) {
	rec := do(t, http.MethodPut, "/tasks/"+seededTaskID,
		`{"title":"Updated task","description":"Updated by the router test","completed":true}`, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t,
		`{"id":"router-test-task","title":"Updated task","description":"Updated by the router test","completed":true}`,
		rec.Body.String())

	rec = do(t, http.MethodPut, "/tasks/does-not-exist",
		`{"title":"Updated task","description":"Updated by the router test"}`, nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, `{"error":"task does-not-exist: task not found"}`, rec.Body.String())

	rec = do(t, http.MethodPut, "/tasks/"+seededTaskID, `{"title":"ab","description":"x"}`, nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// 204 carries neither a content type nor a body.
func TestDeleteTaskNoContent(t *testing.T) {
	rec := do(t, http.MethodDelete, "/tasks/"+seededTaskID, "", nil)
	require.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Header().Get("Content-Type"))
	assert.Empty(t, rec.Body.String())
}

func TestDeleteTaskNotFound(t *testing.T) {
	rec := do(t, http.MethodDelete, "/tasks/does-not-exist", "", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, `{"error":"task does-not-exist: task not found"}`, rec.Body.String())
}

func TestMetricsEndpointIsExposed(t *testing.T) {
	rec := do(t, http.MethodGet, "/metrics", "", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/plain")
	assert.Contains(t, rec.Body.String(), "http_requests_total")
}

// prometheus.yml and the shipped dashboards key off the source framework's
// pattern syntax, so a parameterised route must still be labelled "/tasks/:id".
func TestMetricsPathLabelUsesSourceFrameworkPatterns(t *testing.T) {
	do(t, http.MethodGet, "/tasks/"+seededTaskID, "", nil)
	do(t, http.MethodGet, "/tasks", "", nil)

	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)

	seen := map[string]bool{}
	for _, family := range families {
		if family.GetName() != "http_requests_total" {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "path" {
					seen[label.GetValue()] = true
				}
			}
		}
	}

	assert.True(t, seen["/tasks/:id"], "expected a /tasks/:id path label, saw %v", seen)
	assert.True(t, seen["/tasks"], "expected a /tasks path label, saw %v", seen)
	assert.False(t, seen["/tasks/{id}"], "target framework pattern syntax leaked into the metrics labels")
}
