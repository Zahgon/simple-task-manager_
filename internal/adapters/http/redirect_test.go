package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SilentPlaces/simple-task-manager/internal/adapters/http/handlers"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/ports/logger"
	usecasetasks "github.com/SilentPlaces/simple-task-manager/internal/domain/usecase/tasks"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTrailingSlashRedirectHonoursForwardedPrefix pins the header the source
// router reads when it builds a redirect target, and - just as importantly -
// that the header does not reach the route lookup. Deciding whether to redirect
// from the prefixed path finds no route, abandons the redirect, and 404s every
// request that arrives through a proxy mounting the service under a prefix.
// Nothing else in the suite sends this header, which is how that survived a
// green run.
func TestTrailingSlashRedirectHonoursForwardedPrefix(t *testing.T) {
	cases := []struct {
		name     string
		method   string
		target   string
		prefix   string
		code     int
		location string
	}{
		{"collection", http.MethodGet, "/tasks/", "/api", http.StatusMovedPermanently, "/api/tasks"},
		{"item", http.MethodGet, "/tasks/" + seededTaskID + "/", "/api", http.StatusMovedPermanently, "/api/tasks/" + seededTaskID},
		{"non-GET keeps 307", http.MethodPost, "/tasks/", "/api", http.StatusTemporaryRedirect, "/api/tasks"},
		{"nested prefix", http.MethodGet, "/tasks/", "/api/v1", http.StatusMovedPermanently, "/api/v1/tasks"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := do(t, c.method, c.target, "", map[string]string{"X-Forwarded-Prefix": c.prefix})
			require.Equal(t, c.code, rec.Code,
				"the prefix must not decide whether a redirect happens")
			assert.Equal(t, c.location, rec.Header().Get("Location"))
		})
	}
}

// TestForwardedPrefixIsSanitised covers the two substitutions the source router
// applies to the header before using it: characters outside [a-zA-Z0-9/-] are
// dropped and runs of slashes collapse. The prefix is attacker-controlled, so
// copying it unfiltered into a Location header is a header-injection vector.
func TestForwardedPrefixIsSanitised(t *testing.T) {
	cases := []struct {
		name     string
		prefix   string
		location string
	}{
		{"unsafe characters dropped", "/ap<i>", "/api/tasks"},
		{"repeated slashes cleaned", "//api///v1", "/api/v1/tasks"},
		{"dot segments cleaned", "/api/../api", "/api/tasks"},
		{"empty header is ignored", "", "/tasks"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := do(t, http.MethodGet, "/tasks/", "", map[string]string{"X-Forwarded-Prefix": c.prefix})
			assert.Equal(t, http.StatusMovedPermanently, rec.Code)
			assert.Equal(t, c.location, rec.Header().Get("Location"))
		})
	}
}

// TestRepeatedSlashesStillRedirect pins the source router's behaviour for a
// path carrying a doubled slash. Its routing tree reports a trailing-slash
// redirect for "/tasks//" and answers 301 to "/tasks/" - one slash trimmed off
// the path as it arrived, not a cleaned path. Testing only the single-slash
// case leaves this shape answering 404 with nothing to notice.
func TestRepeatedSlashesStillRedirect(t *testing.T) {
	rec := do(t, http.MethodGet, "/tasks//", "", nil)
	require.Equal(t, http.StatusMovedPermanently, rec.Code)
	assert.Equal(t, "/tasks/", rec.Header().Get("Location"),
		"the Location keeps the request's own shape, minus one trailing slash")
	assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
}

// TestUnroutableRepeatedSlashesStillNotFound is the negative half, and the
// three /tasks shapes are the ones that matter. The source router adds or
// removes exactly ONE trailing slash before consulting its tree; two extra
// slashes are not a trailing-slash mistake. An earlier version of this table
// held only "/unknown//", "/unknown//deeper/" and "//" - paths that resolve to
// nothing however they are normalised - so it passed either way and let a
// regression ship that redirected all three /tasks shapes below.
func TestUnroutableRepeatedSlashesStillNotFound(t *testing.T) {
	for _, target := range []string{
		"/tasks///", "/tasks/1//", "/metrics//",
		"/unknown//", "/unknown//deeper/", "//",
	} {
		rec := do(t, http.MethodGet, target, "", nil)
		assert.Equal(t, http.StatusNotFound, rec.Code, target)
		assert.Equal(t, "404 page not found", rec.Body.String(), target)
	}
}

// TestTrailingSlashRedirectPreservesQueryString guards the redirect against
// being rewritten to use the path alone. The source router redirects to the
// full URL, so a client that follows the Location keeps its query parameters;
// dropping them silently discards filters. folder27 has had this guard from the
// start; this tree did not, and a mutation that switched the redirect to
// r.URL.Path survived because of it.
func TestTrailingSlashRedirectPreservesQueryString(t *testing.T) {
	cases := []struct {
		name     string
		target   string
		headers  map[string]string
		location string
	}{
		{"plain", "/tasks/?completed=true&limit=5", nil, "/tasks?completed=true&limit=5"},
		{"with prefix", "/tasks/?completed=true", map[string]string{"X-Forwarded-Prefix": "/api"}, "/api/tasks?completed=true"},
		{"empty value", "/tasks/?completed=", nil, "/tasks?completed="},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := do(t, http.MethodGet, c.target, "", c.headers)
			require.Equal(t, http.StatusMovedPermanently, rec.Code)
			assert.Equal(t, c.location, rec.Header().Get("Location"))
		})
	}
}

// TestConnectIsNotRedirected pins the observable behaviour: a CONNECT to a path
// that would otherwise redirect falls through to the 404 handler.
//
// Honest limitation: this test cannot fail if the `r.Method != http.MethodConnect`
// guard is deleted. No CONNECT route is registered, so the route lookup that
// follows never matches and the request falls through either way. The guard
// mirrors the source router's own carve-out and is unreachable here - a mutation
// removing it is equivalent, not an untested gap.
func TestConnectIsNotRedirected(t *testing.T) {
	rec := do(t, http.MethodConnect, "/tasks/", "", nil)
	assert.NotEqual(t, http.StatusTemporaryRedirect, rec.Code,
		"CONNECT must never be redirected")
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Empty(t, rec.Header().Get("Location"))
}

// TestRedirectsAreNeitherCountedNorLogged pins the mount order the redirect
// middleware's own comment claims: it runs ahead of the metrics and logging
// middlewares, so a redirected request is neither counted nor logged, while a
// 404 is both. Moving RedirectTrailingSlash behind Metrics() breaks that
// silently - the request still redirects with the same status, body and
// Location, so every assertion about the response keeps passing while the
// dashboards start counting traffic that was never served.
//
// The counter is read as a delta around the request rather than as an absolute,
// because httpRequestsTotal is registered on the global default registry and
// every other test in the package contributes to it.
func TestRedirectsAreNeitherCountedNorLogged(t *testing.T) {
	logger := &countingLogger{}
	repo := NewTaskRepositoryStub()
	useCase := usecasetasks.NewTaskUseCase(repo)
	handler := handlers.NewTaskHandler(useCase, LoggerStub{})
	router := NewRouter(handler, logger)

	before := requestCount(t)
	redirected := httptest.NewRecorder()
	router.ServeHTTP(redirected, httptest.NewRequest(http.MethodGet, "/tasks/", nil))
	require.Equal(t, http.StatusMovedPermanently, redirected.Code)
	assert.Equal(t, before, requestCount(t),
		"a redirected request must not be counted in http_requests_total")
	assert.Zero(t, logger.calls, "a redirected request must not reach the request logger")

	notFound := httptest.NewRecorder()
	router.ServeHTTP(notFound, httptest.NewRequest(http.MethodGet, "/unknown", nil))
	require.Equal(t, http.StatusNotFound, notFound.Code)
	assert.Greater(t, requestCount(t), before, "a 404 must be counted")
	assert.NotZero(t, logger.calls, "a 404 must reach the request logger")
}

// requestCount totals every http_requests_total series currently registered.
func requestCount(t *testing.T) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	total := 0.0
	for _, family := range families {
		if family.GetName() != "http_requests_total" {
			continue
		}
		for _, metric := range family.GetMetric() {
			total += metric.GetCounter().GetValue()
		}
	}
	return total
}

// countingLogger records how many requests reached the request logger. The
// middleware routes by status - Error at 5xx, Warn at 4xx, Info otherwise - so
// all three have to be counted or a 404 would look unlogged.
type countingLogger struct {
	calls int
}

func (l *countingLogger) count(msg string) {
	if msg == "Request" {
		l.calls++
	}
}

func (l *countingLogger) Info(msg string, _ ...any)  { l.count(msg) }
func (l *countingLogger) Warn(msg string, _ ...any)  { l.count(msg) }
func (l *countingLogger) Error(msg string, _ ...any) { l.count(msg) }
func (l *countingLogger) Debug(string, ...any)       {}
func (l *countingLogger) Fatal(string, ...any)       {}
func (l *countingLogger) With(...any) logger.Logger  { return l }
