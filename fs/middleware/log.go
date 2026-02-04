package middleware

import (
	"net/http"
	"time"

	ds "github.com/benjamonnguyen/deadsimple"
)

// Log logs each HTTP request with method, path, timestamp, and duration.
func Log(logger ds.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"time", start.Format(time.RFC3339),
			"duration", time.Since(start),
		)
	})
}
