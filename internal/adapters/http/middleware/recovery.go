package middleware

import (
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/SilentPlaces/simple-task-manager/internal/domain/ports/logger"
	"github.com/gin-gonic/gin"
)

func Recovery(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				if isBrokenPipe(r) {
					log.Warn("Broken pipe", "path", c.Request.URL.Path, "error", r)
					c.Abort()
					return
				}

				log.Error("Panic recovered", "panic", r, "method", c.Request.Method, "path", c.Request.URL.Path)

				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
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
