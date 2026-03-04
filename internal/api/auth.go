package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// authMiddleware returns an http.Handler that validates Bearer token
// authentication on all requests except GET /api/v1/health.
func authMiddleware(token string, next http.Handler) http.Handler {
	tokenBytes := []byte(token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health endpoint is always exempt from auth.
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/health" {
			next.ServeHTTP(w, r)
			return
		}

		provided := extractBearer(r.Header.Get("Authorization"))
		if len(provided) == 0 || subtle.ConstantTimeCompare([]byte(provided), tokenBytes) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// extractBearer returns the token from a "Bearer <token>" header value.
func extractBearer(header string) string {
	const prefix = "Bearer "
	if len(header) > len(prefix) && strings.HasPrefix(header, prefix) {
		return header[len(prefix):]
	}
	return ""
}
