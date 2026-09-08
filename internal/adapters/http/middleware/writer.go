package middleware

import (
	"bufio"
	"errors"
	"net"
	"net/http"
)

const (
	noWritten     = -1
	defaultStatus = http.StatusOK
)

// ResponseWriter is the response writer handed to the middleware chain and to
// the handlers. It defers the status code write until the body is written (or
// until the chain unwinds), which is what allows the request logger to observe
// both the final status code and the number of bytes written.
type ResponseWriter interface {
	http.ResponseWriter
	http.Flusher
	http.Hijacker

	// Status returns the status code of the current request.
	Status() int
	// Size returns the number of bytes already written to the response body,
	// or -1 when nothing has been written yet.
	Size() int
	// Written tells whether the response header has been flushed.
	Written() bool
	// WriteHeaderNow forces the deferred status code to be written.
	WriteHeaderNow()
	// DeferBody records a body to be written once the middleware chain has
	// unwound rather than immediately. The source router's serveError does
	// exactly this for its default 404, which is why the request logger there
	// reports an unwritten body for an unrouted request.
	DeferBody(code int, body []byte)
}

type responseWriter struct {
	http.ResponseWriter
	size         int
	status       int
	deferred     []byte
	deferredCode int
}

func (w *responseWriter) WriteHeader(code int) {
	if code > 0 && w.status != code {
		if w.Written() {
			return
		}
		w.status = code
	}
}

func (w *responseWriter) WriteHeaderNow() {
	if !w.Written() {
		w.size = 0
		w.ResponseWriter.WriteHeader(w.status)
	}
}

func (w *responseWriter) Write(data []byte) (n int, err error) {
	w.WriteHeaderNow()
	n, err = w.ResponseWriter.Write(data)
	w.size += n
	return
}

func (w *responseWriter) WriteString(s string) (n int, err error) {
	w.WriteHeaderNow()
	n, err = w.ResponseWriter.Write([]byte(s))
	w.size += n
	return
}

func (w *responseWriter) DeferBody(code int, body []byte) {
	w.deferredCode, w.deferred = code, body
}

func (w *responseWriter) writeDeferred() {
	if w.deferred == nil || w.Written() || w.status != w.deferredCode {
		return
	}
	_, _ = w.Write(w.deferred)
}

func (w *responseWriter) Status() int {
	return w.status
}

func (w *responseWriter) Size() int {
	return w.size
}

func (w *responseWriter) Written() bool {
	return w.size != noWritten
}

func (w *responseWriter) Flush() {
	w.WriteHeaderNow()
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.size < 0 {
		w.size = 0
	}
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("the ResponseWriter doesn't support the Hijacker interface")
	}
	return hijacker.Hijack()
}

// Writer installs the deferred response writer for the rest of the chain and
// flushes the recorded status code once the chain has unwound.
func Writer() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &responseWriter{ResponseWriter: w, size: noWritten, status: defaultStatus}
			next.ServeHTTP(rw, r)
			rw.writeDeferred()
			rw.WriteHeaderNow()
		})
	}
}

// writerOf returns the deferred response writer installed by Writer.
func writerOf(w http.ResponseWriter) (ResponseWriter, bool) {
	rw, ok := w.(ResponseWriter)
	return rw, ok
}
