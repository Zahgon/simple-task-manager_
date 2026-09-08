package middleware

import (
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
)

// regSafePrefix and regRemoveRepeatedChar are the source router's own
// sanitisers for X-Forwarded-Prefix, copied verbatim from gin.go so the
// Location this middleware emits matches the original byte for byte.
var (
	regSafePrefix         = regexp.MustCompile("[^a-zA-Z0-9/-]+")
	regRemoveRepeatedChar = regexp.MustCompile("/{2,}")
)

// DecodedPath makes the router match on the decoded URL path.
//
// chi prefers URL.RawPath over URL.Path whenever the two differ, so a request
// for "/tasks%2F1" would be routed as the literal three-segment string instead
// of as "/tasks/1". Clearing RawPath restores matching on the decoded path.
func DecodedPath() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.URL.RawPath = ""
			next.ServeHTTP(w, r)
		})
	}
}

// RedirectTrailingSlash redirects a request whose path does not match any route
// but whose path with a trailing slash added (or removed) does. The redirect is
// issued before the rest of the chain runs, so redirected requests are neither
// logged nor counted in the metrics.
func RedirectTrailingSlash(mux *chi.Mux) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodConnect && !matches(mux, r.Method, r.URL.Path) {
				// Decide WHETHER to redirect from the unprefixed path. The
				// source router's tree never sees X-Forwarded-Prefix; the prefix
				// only shapes the Location afterwards. Matching the prefixed path
				// against routes that carry no prefix finds nothing, abandons the
				// redirect, and 404s every request that arrives through a proxy
				// setting the header.
				//
				// The source router adds or removes EXACTLY ONE trailing slash and
				// asks its tree about the result - it collapses nothing. Two extra
				// slashes are not a trailing-slash mistake, so "/tasks///" is a 404
				// while "/tasks//" is a redirect to "/tasks/".
				p := r.URL.Path
				alternate, trimmed := p+"/", false
				if length := len(p); length > 1 && p[length-1] == '/' {
					alternate, trimmed = p[:length-1], true
				}

				if routeExists(mux, r.Method, alternate, trimmed) {
					// The Location keeps the request's own shape, repeated
					// slashes included: the source router trims one trailing
					// slash off the path as it arrived.
					if prefix := path.Clean(r.Header.Get("X-Forwarded-Prefix")); prefix != "." {
						prefix = regSafePrefix.ReplaceAllString(prefix, "")
						prefix = regRemoveRepeatedChar.ReplaceAllString(prefix, "/")
						p = prefix + "/" + r.URL.Path
					}
					target := p + "/"
					if length := len(p); length > 1 && p[length-1] == '/' {
						target = p[:length-1]
					}

					code := http.StatusMovedPermanently
					if r.Method != http.MethodGet {
						code = http.StatusTemporaryRedirect
					}
					r.URL.Path = target
					http.Redirect(w, r, r.URL.String(), code)
					if rw, ok := writerOf(w); ok {
						rw.WriteHeaderNow()
					}
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func matches(mux *chi.Mux, method, target string) bool {
	return mux.Match(chi.NewRouteContext(), method, target)
}

// emptyParamProbe stands in for the empty final segment that the source
// router's parameter nodes accept and chi's do not. It is only ever used to ask
// the router a question; it never reaches a handler or a Location header.
const emptyParamProbe = "\x00emptyparam"

// routeExists reports whether the source router's tree would have matched
// target after the caller adjusted one trailing slash.
//
// The two routers disagree about one thing: a parameter node in the source
// router matches an EMPTY path segment, so "/tasks/" reaches "/tasks/{id}"
// there while chi rejects it. That single difference is why "/tasks//"
// redirects to "/tasks/" while "/tasks///" is a 404 - collapsing slashes to
// paper over it also redirects the paths the source router answers with a 404.
//
// The allowance applies only when a slash was REMOVED. Extending it to an added
// slash would let a method mismatch on a collection ("PUT /tasks") reach the
// item route ("/tasks/{id}", which does serve PUT) and answer 307 where the
// source router answers 404.
func routeExists(mux *chi.Mux, method, target string, trimmed bool) bool {
	if matches(mux, method, target) {
		return true
	}
	return trimmed && strings.HasSuffix(target, "/") &&
		matches(mux, method, target+emptyParamProbe)
}
