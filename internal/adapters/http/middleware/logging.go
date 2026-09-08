package middleware

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/SilentPlaces/simple-task-manager/internal/domain/ports/logger"
)

func RequestLogger(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			path := r.URL.Path
			query := r.URL.RawQuery

			next.ServeHTTP(w, r)

			status := statusOf(w)
			latency := time.Since(start)

			msg := "Request"
			fields := []any{
				"status", status,
				"method", r.Method,
				"path", path,
				"query", query,
				"ip", clientIP(r),
				"latency", latency,
				"body_size", sizeOf(w),
			}
			switch {
			case status >= 500:
				log.Error(msg, fields...)
			case status >= 400:
				log.Warn(msg, fields...)
			default:
				log.Info(msg, fields...)
			}
		})
	}
}

func sizeOf(w http.ResponseWriter) int {
	if rw, ok := writerOf(w); ok {
		return rw.Size()
	}
	return noWritten
}

// remoteIPHeaders are consulted, in order, when the immediate peer is a trusted
// proxy.
var remoteIPHeaders = []string{"X-Forwarded-For", "X-Real-IP"}

// trustedProxies matches every peer: any proxy in front of the service is
// trusted to report the originating client address.
var trustedProxies = []*net.IPNet{
	mustParseCIDR("0.0.0.0/0"),
	mustParseCIDR("::/0"),
}

func mustParseCIDR(cidr string) *net.IPNet {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	return network
}

// clientIP resolves the originating client address, preferring the forwarding
// headers set by a trusted proxy over the address of the immediate peer.
func clientIP(r *http.Request) string {
	remote := net.ParseIP(remoteIP(r))
	if remote == nil {
		return ""
	}

	if isTrustedProxy(remote) {
		for _, header := range remoteIPHeaders {
			if ip, valid := validateHeader(r.Header.Get(header)); valid {
				return ip
			}
		}
	}

	return remote.String()
}

func remoteIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		return ""
	}
	return ip
}

func isTrustedProxy(ip net.IP) bool {
	for _, network := range trustedProxies {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// validateHeader walks a forwarding header from right to left and returns the
// first address that is either the left-most entry or reported by an untrusted
// proxy. A malformed entry discards the whole header.
func validateHeader(header string) (clientIP string, valid bool) {
	if header == "" {
		return "", false
	}
	items := strings.Split(header, ",")
	for i := len(items) - 1; i >= 0; i-- {
		ipStr := strings.TrimSpace(items[i])
		ip := net.ParseIP(ipStr)
		if ip == nil {
			break
		}
		if (i == 0) || (!isTrustedProxy(ip)) {
			return ipStr, true
		}
	}
	return "", false
}
