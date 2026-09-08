package http

import (
	"net/http"

	"github.com/SilentPlaces/simple-task-manager/internal/adapters/http/handlers"
	"github.com/SilentPlaces/simple-task-manager/internal/adapters/http/middleware"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/ports/logger"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(taskHandler *handlers.TaskHandler, log logger.Logger) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Writer())
	r.Use(middleware.DecodedPath())
	r.Use(middleware.RedirectTrailingSlash(r))
	r.Use(middleware.Metrics())
	r.Use(middleware.RequestLogger(log))
	r.Use(middleware.Recovery(log))

	r.NotFound(routeNotFound)
	r.MethodNotAllowed(routeNotFound)

	r.Get("/metrics", promhttp.Handler().ServeHTTP)

	// Registered flat rather than through chi.Router.Route: mounting a
	// subrouter would also answer "/tasks/" for every method, which would
	// swallow the trailing slash redirect and relabel the metrics.
	r.Get("/tasks", taskHandler.GetTasks)
	r.Get("/tasks/{id}", taskHandler.GetTask)
	r.Post("/tasks", taskHandler.AddTask)
	r.Put("/tasks/{id}", taskHandler.UpdateTask)
	r.Delete("/tasks/{id}", taskHandler.DeleteTask)

	return r
}

// routeNotFound answers every unrouted request, whether the path is unknown or
// the path is known but the method is not.
func routeNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusNotFound)
	body := []byte("404 page not found")
	// The source router writes this body only after the middleware chain has
	// unwound, so its request logger records body_size=-1 for an unrouted
	// request. Writing it here instead would record 18.
	if rw, ok := w.(middleware.ResponseWriter); ok {
		rw.DeferBody(http.StatusNotFound, body)
		return
	}
	_, _ = w.Write(body)
}
