package middleware

import (
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func Recovery(log zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				if isBrokenPipe(r) {
					log.Warn().
						Str("path", c.Request.URL.Path).
						Interface("error", r).
						Msg("Broken pipe")
					c.Abort()
					return
				}

				log.Error().
					Interface("panic", r).
					Str("method", c.Request.Method).
					Str("path", c.Request.URL.Path).
					Msg("Panic recovered")

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
