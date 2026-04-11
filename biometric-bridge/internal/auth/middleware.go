package auth

import (
	"log/slog"
	"net/http"
	"strings"
)

// Middleware returns an HTTP middleware that validates JWT Bearer tokens on all
// requests except those matching skipPaths (e.g., "/healthz"). On failure, it
// returns 401 with {"error":"unauthorized"}.
func Middleware(v *TokenValidator, skipPaths map[string]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skipPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			token := extractBearerToken(r)
			if token == "" {
				slog.Debug("auth rejected: missing token", "path", r.URL.Path, "remote", r.RemoteAddr)
				writeUnauthorized(w)
				return
			}

			if _, err := v.Validate(token); err != nil {
				slog.Debug("auth rejected: invalid token", "path", r.URL.Path, "remote", r.RemoteAddr, "error", err)
				writeUnauthorized(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ExtractWSToken extracts and validates a JWT from the "token" query parameter
// on a WebSocket upgrade request. Returns the parsed claims or an error.
func ExtractWSToken(v *TokenValidator, r *http.Request) (*Claims, error) {
	token := r.URL.Query().Get("token")
	if token == "" {
		return nil, &TokenError{Message: "missing token query parameter"}
	}
	return v.Validate(token)
}

// TokenError is returned when token extraction or validation fails.
type TokenError struct {
	Message string
}

func (e *TokenError) Error() string {
	return e.Message
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"unauthorized"}`))
}
