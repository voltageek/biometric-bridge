// Package api provides HTTP route registration, CORS middleware, health check,
// and all REST endpoint handlers for the Biometric Bridge.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"biometric-bridge/internal/auth"
	"biometric-bridge/internal/device"
	"biometric-bridge/internal/driver"
	"biometric-bridge/internal/events"
)

// RouterDeps holds all dependencies needed by the router and handlers.
type RouterDeps struct {
	TokenValidator *auth.TokenValidator
	Driver         driver.Driver
	Registry       *device.Registry
	Broker         *events.Broker
	AllowedOrigin  string
	Demo           bool
	RequestLogging bool
}

// NewRouter creates an http.Handler with all routes registered, CORS applied,
// and auth middleware on /api/* routes. The /healthz endpoint is always open.
func NewRouter(deps RouterDeps) http.Handler {
	mux := http.NewServeMux()

	// Health check — no auth
	mux.HandleFunc("GET /healthz", makeHealthzHandler(deps.Demo))

	// Authenticated API routes
	mux.HandleFunc("GET /api/devices", NewDevicesHandler(deps.Driver))
	mux.HandleFunc("POST /api/scan", NewScanHandler(deps.Driver, deps.Registry))
	mux.HandleFunc("POST /api/slap-scan", NewSlapScanHandler(deps.Driver, deps.Registry))
	mux.HandleFunc("POST /api/enroll", NewEnrollHandler(deps.Driver, deps.Registry))

	// WebSocket event stream (auth handled inside the handler via query param)
	mux.Handle("GET /events", events.NewHandler(deps.Broker, deps.TokenValidator))

	// Apply auth middleware (skips /healthz and /events which handle auth internally)
	skipPaths := map[string]bool{"/healthz": true, "/events": true}
	authed := auth.Middleware(deps.TokenValidator, skipPaths)(mux)

	// Apply request logging middleware if enabled
	var handler http.Handler = authed
	if deps.RequestLogging {
		handler = requestLoggingMiddleware(handler)
	}

	// Apply CORS
	return corsMiddleware(deps.AllowedOrigin, handler)
}

// makeHealthzHandler returns a health check handler. When demo is true,
// the response includes {"status":"ok","demo":true,"version":"dev"}.
func makeHealthzHandler(demo bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if demo {
			w.Write([]byte(`{"status":"ok","demo":true,"version":"dev"}`))
			return
		}
		w.Write([]byte(`{"status":"ok"}`))
	}
}

// corsMiddleware applies CORS headers per the http-api.md contract.
func corsMiddleware(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// requestLoggingMiddleware logs HTTP request details (method, path, status, duration, remote address).
func requestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration_ms", duration.Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
