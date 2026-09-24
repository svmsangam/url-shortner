// Package middleware provides request guards and cross-origin transport policy
// shared by the HTTP router and its API handlers.
package middleware

import "net/http"

// CORSMiddleware permits the browser client to call the REST API and terminates
// preflight requests before they reach application handlers.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173") // Your React URL
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Device-Token")
		w.Header().Set("Access-Control-Expose-Headers", "X-Device-Token")

		// Intercept preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
