// Package middleware provides HTTP middleware functions for the GO-IT server.
package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

// RequestLogger is a chi-compatible middleware that logs HTTP requests with
// method, path, status code, duration and request ID.
//
// Log format: [2026-11-10 12:00:00] GET /cases/frontend 200 45ms
func RequestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			srw := chiMiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Extract request ID from request context
			reqID := r.Context().Value(chiMiddleware.RequestIDKey)
			if id, ok := reqID.(string); ok {
				srw.Header().Set("X-Request-ID", id)
			}

			defer func() {
				duration := time.Since(start)
				status := srw.Status()

				log.Info("request",
					"method", r.Method,
					"path", r.URL.Path,
					"status", status,
					"duration", duration,
					"request_id", reqID,
				)
			}()

			next.ServeHTTP(srw, r)
		})
	}
}
