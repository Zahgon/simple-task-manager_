package http

import (
	"github.com/rs/zerolog"

	"github.com/SilentPlaces/simple-task-manager/internal/adapters/http/handlers"
	"github.com/SilentPlaces/simple-task-manager/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func NewRouter(taskHandler *handlers.TaskHandler, log zerolog.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware.RequestLogger(log))
	r.Use(middleware.Recovery(log))

	tasks := r.Group("/tasks")
	{
		tasks.GET("", taskHandler.GetTasks)
		tasks.GET("/:id", taskHandler.GetTask)
		tasks.POST("", taskHandler.AddTask)
		tasks.PUT("/:id", taskHandler.UpdateTask)
		tasks.DELETE("/:id", taskHandler.DeleteTask)
	}
	return r
}
