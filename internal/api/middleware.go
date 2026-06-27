package api

import (
	"crypto/hmac"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"md/internal/config"
)

// loggingMiddleware logs each request.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote", r.RemoteAddr,
		)
	})
}

// apiKeyMiddleware enforces an optional API key (X-API-Key header or ?api_key query param).
//
// When no API key is configured it fails CLOSED unless another auth layer is
// active (OIDC) or anonymous access has been explicitly opted into via
// MD_ALLOW_ANONYMOUS=true. This prevents silently serving the conversion and
// file CRUD API to the public when the operator forgot to set credentials.
func apiKeyMiddleware(cfg *config.Config, oidcEnabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.APIKey == "" {
				// OIDC already enforces auth for these routes; or the operator
				// explicitly enabled anonymous access.
				if oidcEnabled || cfg.AllowAnonymous {
					next.ServeHTTP(w, r)
					return
				}
				slog.Error("refusing request: no authentication configured — set MD_API_KEY, enable OIDC, or set MD_ALLOW_ANONYMOUS=true to serve publicly",
					"path", r.URL.Path)
				writeError(w, http.StatusServiceUnavailable, "service unavailable: authentication is not configured")
				return
			}
			key := r.Header.Get("X-API-Key")
			if key == "" {
				if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
					key = strings.TrimPrefix(auth, "Bearer ")
				}
			}
			if !hmac.Equal([]byte(key), []byte(cfg.APIKey)) {
				writeError(w, http.StatusUnauthorized, "invalid or missing API key")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// securityHeaders adds hardened HTTP security headers.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		// NOTE: 'unsafe-eval' is required by the bundled client-side renderers
		// (mermaid + katex use the Function constructor); removing it breaks
		// the in-app preview. Stored-XSS is instead neutralized server-side by
		// the bluemonday sanitizer in renderMarkdown (see files.go). The extra
		// directives below (object-src/base-uri/frame-ancestors/form-action)
		// are pure hardening with no functional impact.
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net; "+
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdn.jsdelivr.net; "+
				"img-src 'self' data: blob: https:; "+
				"font-src 'self' data: https://fonts.gstatic.com https://cdn.jsdelivr.net; "+
				"connect-src 'self' wss: ws:; "+
				"object-src 'none'; "+
				"base-uri 'self'; "+
				"frame-ancestors 'self'; "+
				"form-action 'self';")
		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

// Flush implements http.Flusher so SSE streaming works through the logging middleware.
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap allows http.ResponseController to access the underlying ResponseWriter.
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}
