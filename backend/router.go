package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"url-shortner/handler"
	devicemw "url-shortner/middleware"
	"url-shortner/redisid"
	"url-shortner/store"
)

// SetupRouter constructs the HTTP router, registers middleware and routes,
// and returns the router as an http.Handler. All route definitions live here
// (not in main).
func SetupRouter(db *store.DB, gen *redisid.Generator) http.Handler {
	r := chi.NewRouter()

	// Common middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	// API route group
	r.Route("/api", func(r chi.Router) {
		// POST /api/shorten - create a short url
		// DeviceTokenMiddleware injects a device token in the request context
		// (and sets a X-DEVICE_TOKEN header if needed). NewShortenHandler handles the request.
		r.With(devicemw.DeviceTokenMiddleware).Post("/shorten", handler.NewShortenHandler(db, gen))
		r.With(devicemw.DeviceTokenMiddleware).Get("/{code}", handler.NewGetHandler(db, gen))
		r.With(devicemw.DeviceTokenMiddleware).Get("/urls", handler.GetDeviceURLs(db))
	})

	return r
}
