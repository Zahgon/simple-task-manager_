package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	httpDTO "github.com/SilentPlaces/simple-task-manager/internal/adapters/http/dto"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/entities"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/ports/logger"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/usecase/dto"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/usecase/tasks"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type TaskHandler struct {
	uc  *tasks.TaskUseCase
	log logger.Logger
}

func NewTaskHandler(useCase *tasks.TaskUseCase, log logger.Logger) *TaskHandler {
	return &TaskHandler{uc: useCase, log: log}
}

type errorResponse struct {
	Error string `json:"error"`
}

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New()
	v.SetTagName("binding")
	return v
}

func bindJSON(r *http.Request, obj any) error {
	if r == nil || r.Body == nil {
		return errors.New("invalid request")
	}
	if err := json.NewDecoder(r.Body).Decode(obj); err != nil {
		return err
	}
	return validate.Struct(obj)
}

func respondJSON(w http.ResponseWriter, code int, obj any) {
	payload, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_, _ = w.Write(payload)
}

func respondError(w http.ResponseWriter, code int, msg string) {
	respondJSON(w, code, errorResponse{Error: msg})
}

func (h *TaskHandler) handleUseCaseError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, entities.ErrTaskNotFound) {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	h.log.Error("Internal error", "error", err, "method", r.Method, "path", r.URL.Path)
	respondError(w, http.StatusInternalServerError, "internal server error")
}

func parseTimeQuery(r *http.Request, param string) (*time.Time, error) {
	raw := r.URL.Query().Get(param)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (h *TaskHandler) GetTasks(w http.ResponseWriter, r *http.Request) {
	dueAfter, err := parseTimeQuery(r, "dueAfter")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid dueAfter: expected RFC3339 format")
		return
	}
	dueBefore, err := parseTimeQuery(r, "dueBefore")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid dueBefore: expected RFC3339 format")
		return
	}

	var completed *bool
	if raw := r.URL.Query().Get("completed"); raw != "" {
		val := raw == "true"
		completed = &val
	}

	filter := &dto.TaskFilters{
		DueAfter:  dueAfter,
		DueBefore: dueBefore,
		Completed: completed,
	}

	result, err := h.uc.GetTasks(r.Context(), filter)
	if err != nil {
		h.handleUseCaseError(w, r, err)
		return
	}

	responses := make([]httpDTO.TaskResponse, len(result))
	for i, task := range result {
		responses[i] = httpDTO.TaskResponseFromEntity(task)
	}
	respondJSON(w, http.StatusOK, responses)
}

func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}

	result, err := h.uc.GetTask(r.Context(), id)
	if err != nil {
		h.handleUseCaseError(w, r, err)
		return
	}

	respondJSON(w, http.StatusOK, httpDTO.TaskResponseFromEntity(*result))
}

func (h *TaskHandler) AddTask(w http.ResponseWriter, r *http.Request) {
	var req httpDTO.CreateTaskRequest
	if err := bindJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	task, err := h.uc.AddTask(r.Context(), req.ToEntity())
	if err != nil {
		h.handleUseCaseError(w, r, err)
		return
	}

	respondJSON(w, http.StatusCreated, httpDTO.TaskResponseFromEntity(*task))
}

func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req httpDTO.UpdateTaskRequest
	if err := bindJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	task, err := h.uc.UpdateTask(r.Context(), req.ToEntity(id))
	if err != nil {
		h.handleUseCaseError(w, r, err)
		return
	}

	respondJSON(w, http.StatusOK, httpDTO.TaskResponseFromEntity(*task))
}

func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.uc.DeleteTask(r.Context(), id); err != nil {
		h.handleUseCaseError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
