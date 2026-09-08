package middleware

import (
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/SilentPlaces/simple-task-manager/internal/domain/ports/logger"
)

func Recovery(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					if isBrokenPipe(rec) {
						log.Warn("Broken pipe", "path", r.URL.Path, "error", rec)
						return
					}

					log.Error("Panic recovered", "panic", rec, "method", r.Method, "path", r.URL.Path)

					w.WriteHeader(http.StatusInternalServerError)
					if rw, ok := writerOf(w); ok {
						rw.WriteHeaderNow()
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func isBrokenPipe(r any) bool {
	if ne, ok := r.(*net.OpError); ok {
		if se, ok := ne.Err.(*os.SyscallError); ok {
			return strings.Contains(strings.ToLower(se.Error()), "broken pipe") ||
				strings.Contains(strings.ToLower(se.Error()), "connection reset by peer")
		}
	}
	return false
}
