package middleware

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed, labeled by method, path and status.",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

// chiParam matches a chi path parameter such as "{id}" or "{id:[0-9]+}".
var chiParam = regexp.MustCompile(`\{([^{}:]+)(?::[^{}]*)?\}`)

func Metrics() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			next.ServeHTTP(w, r)

			path := routePattern(r)
			if path == "" {
				path = r.URL.Path
			}

			duration := time.Since(start).Seconds()
			status := strconv.Itoa(statusOf(w))

			httpRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
			httpRequestDurationSeconds.WithLabelValues(r.Method, path).Observe(duration)
		})
	}
}

// routePattern returns the matched route pattern using the ":id" placeholder
// syntax the metrics have always been labeled with.
func routePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return ""
	}
	return chiParam.ReplaceAllString(rctx.RoutePattern(), ":$1")
}

func statusOf(w http.ResponseWriter) int {
	if rw, ok := writerOf(w); ok {
		return rw.Status()
	}
	return defaultStatus
}
