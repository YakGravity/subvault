package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
)

// CSRFMiddleware wraps gorilla/csrf for use with Gin.
// It skips CSRF validation for exempt paths (API v1, static files, etc.).
func CSRFMiddleware(secret []byte, secure bool) gin.HandlerFunc {
	protect := csrf.Protect(
		secret,
		csrf.Secure(secure),
		csrf.Path("/"),
		csrf.ErrorHandler(http.HandlerFunc(csrfErrorHandler)),
	)

	return func(c *gin.Context) {
		if isCSRFExempt(c.Request.URL.Path) {
			c.Next()
			return
		}

		// For plaintext HTTP, mark the request so gorilla/csrf skips Referer checks
		if !secure {
			c.Request = csrf.PlaintextHTTPRequest(c.Request)
		}

		// Wrap gin's ResponseWriter for gorilla/csrf
		handler := protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token and store in Gin context
			token := csrf.Token(r)
			c.Set("csrf_token", token)

			// Set token in response header for HTMX fragment updates
			c.Header("X-CSRF-Token", token)

			// Update the request in gin context (gorilla/csrf stores data in context)
			c.Request = r

			c.Next()
		}))

		handler.ServeHTTP(c.Writer, c.Request)

		// If gorilla/csrf aborted (403), prevent Gin from continuing
		if c.Writer.Status() == http.StatusForbidden {
			c.Abort()
		}
	}
}

// isCSRFExempt returns true for paths that should bypass CSRF validation.
func isCSRFExempt(path string) bool {
	if strings.HasPrefix(path, "/api/v1/") {
		return true
	}
	if strings.HasPrefix(path, "/static/") {
		return true
	}
	if strings.HasPrefix(path, "/cal/") {
		return true
	}

	exemptPaths := []string{"/favicon.ico", "/healthz", "/manifest.json"}
	for _, p := range exemptPaths {
		if path == p {
			return true
		}
	}

	return false
}

// csrfErrorHandler handles CSRF validation failures.
func csrfErrorHandler(w http.ResponseWriter, r *http.Request) {
	slog.Warn("CSRF validation failed",
		"method", r.Method,
		"path", r.URL.Path,
		"reason", csrf.FailureReason(r),
	)

	if isHTMXRequest(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`<div class="alert alert-error">Security validation failed. Please reload the page and try again.</div>`))
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden - CSRF token invalid\n"))
	}
}

func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
