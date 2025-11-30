package middleware

import (
	"net/http"
	"os"
)

// CORS middleware handles Cross-Origin Resource Sharing
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get allowed origins from environment variable or use default
		allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
		if allowedOrigin == "" {
			// Default: allow localhost for development
			allowedOrigin = "http://localhost:3000"
		}

		// Get the origin from the request
		requestOrigin := r.Header.Get("Origin")

		// Validate origin: only allow if it matches the configured origin
		// For same-origin requests (no Origin header), allow if it's the configured origin
		if requestOrigin != "" {
			// Cross-origin request: check if origin matches allowed origin
			if requestOrigin == allowedOrigin {
				w.Header().Set("Access-Control-Allow-Origin", requestOrigin)
			}
			// If origin doesn't match, don't set the header (browser will block the request)
		} else {
			// Same-origin request: allow if it's from the configured origin
			// Note: For same-origin requests, CORS headers are not strictly necessary,
			// but we set them for consistency
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		}

		// Set CORS headers (only if origin was allowed)
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true") // Required for cookies
			w.Header().Set("Access-Control-Max-Age", "3600")
		}

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

