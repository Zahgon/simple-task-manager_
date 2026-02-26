package handlers

import (
	"errors"
	"net/http"
	"time"

	httpDTO "github.com/SilentPlaces/simple-task-manager/internal/adapters/http/dto"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/entities"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/ports/logger"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/usecase/dto"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/usecase/tasks"
	"github.com/gin-gonic/gin"
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

func respondError(c *gin.Context, code int, msg string) {
	c.AbortWithStatusJSON(code, errorResponse{Error: msg})
}

func (h *TaskHandler) handleUseCaseError(c *gin.Context, err error) {
	if errors.Is(err, entities.ErrTaskNotFound) {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}
	h.log.Error("Internal error", "error", err, "method", c.Request.Method, "path", c.Request.URL.Path)
	respondError(c, http.StatusInternalServerError, "internal server error")
}

func parseTimeQuery(c *gin.Context, param string) (*time.Time, error) {
	raw := c.Query(param)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (h *TaskHandler) GetTasks(c *gin.Context) {
	dueAfter, err := parseTimeQuery(c, "dueAfter")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid dueAfter: expected RFC3339 format")
		return
	}
	dueBefore, err := parseTimeQuery(c, "dueBefore")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid dueBefore: expected RFC3339 format")
		return
	}

	var completed *bool
	if raw := c.Query("completed"); raw != "" {
		val := raw == "true"
		completed = &val
	}

	filter := &dto.TaskFilters{
		DueAfter:  dueAfter,
		DueBefore: dueBefore,
		Completed: completed,
	}

	result, err := h.uc.GetTasks(c.Request.Context(), filter)
	if err != nil {
		h.handleUseCaseError(c, err)
		return
	}

	responses := make([]httpDTO.TaskResponse, len(result))
	for i, task := range result {
		responses[i] = httpDTO.TaskResponseFromEntity(task)
	}
	c.JSON(http.StatusOK, responses)
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		respondError(c, http.StatusBadRequest, "id is required")
		return
	}

	result, err := h.uc.GetTask(c.Request.Context(), id)
	if err != nil {
		h.handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, httpDTO.TaskResponseFromEntity(*result))
}

func (h *TaskHandler) AddTask(c *gin.Context) {
	var req httpDTO.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	task, err := h.uc.AddTask(c.Request.Context(), req.ToEntity())
	if err != nil {
		h.handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusCreated, httpDTO.TaskResponseFromEntity(*task))
}

func (h *TaskHandler) UpdateTask(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		respondError(c, http.StatusBadRequest, "id is required")
		return
	}

	var req httpDTO.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	task, err := h.uc.UpdateTask(c.Request.Context(), req.ToEntity(id))
	if err != nil {
		h.handleUseCaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, httpDTO.TaskResponseFromEntity(*task))
}

func (h *TaskHandler) DeleteTask(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		respondError(c, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.uc.DeleteTask(c.Request.Context(), id); err != nil {
		h.handleUseCaseError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
