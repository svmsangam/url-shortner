package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gocql/gocql"

	"url-shortner/base62"
	"url-shortner/middleware"
	"url-shortner/redisid"
	"url-shortner/store"
)

type deviceURLRecord struct {
	ShortCode string    `json:"short_code"`
	LongURL   string    `json:"long_url"`
	CreatedAt time.Time `json:"created_at"`
}

type deviceURLsResponse struct {
	Records    []deviceURLRecord `json:"records"`
	TotalCount int               `json:"total_count"`
}

// GetDeviceURLs returns all short URLs associated with the request's device.
func GetDeviceURLs(db *store.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceToken, ok := middleware.GetDeviceToken(r.Context())
		if !ok {
			http.Error(w, "missing device token", http.StatusUnauthorized)
			return
		}

		urls, err := db.GetDeviceURLs(r.Context(), deviceToken)
		if err != nil {
			http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
			return
		}

		records := make([]deviceURLRecord, 0, len(urls))
		for _, url := range urls {
			records = append(records, deviceURLRecord{
				ShortCode: url.ShortCode,
				LongURL:   url.LongURL,
				CreatedAt: url.CreatedAt,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(deviceURLsResponse{
			Records:    records,
			TotalCount: len(records),
		})
	}
}

// NewGetHandler returns an http.HandlerFunc that looks up a short code and
// returns the corresponding long URL as JSON {"redirect_url": "..."}.
// It implements a cache-aside pattern using redisid.Generator's cache helpers.
func NewGetHandler(db *store.DB, gen *redisid.Generator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := middleware.GetDeviceToken(r.Context())
		if !ok {
			http.Error(w, "missing device token", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()

		// Extract code either from path param or from query parameter 'url'.
		code := chi.URLParam(r, "code")
		if code == "" {
			// allow full short URL via query param 'url', e.g., ?url=http://host/abc123
			raw := r.URL.Query().Get("url")
			if raw == "" {
				http.Error(w, "missing short code or url parameter", http.StatusBadRequest)
				return
			}
			// Take last path segment
			slash := strings.LastIndex(raw, "/")
			if slash >= 0 && slash < len(raw)-1 {
				code = raw[slash+1:]
			} else {
				code = raw
			}
		}

		// Validate code by attempting to decode Base62
		if _, err := base62.Decode(code); err != nil {
			http.Error(w, "invalid short code", http.StatusBadRequest)
			return
		}

		// 1) Try cache
		if gen != nil {
			if longURL, err := gen.GetURL(ctx, code); err != nil {
				http.Error(w, fmt.Sprintf("cache error: %v", err), http.StatusInternalServerError)
				return
			} else if longURL != "" {
				log.Printf("cache hit for code %s -> %s", code, longURL)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]string{"redirect_url": longURL})
				return
			}
		}

		// 2) Cache miss -> query Cassandra
		longURL, err := db.GetLongURLByShortCode(ctx, code)
		if err != nil {
			if errors.Is(err, gocql.ErrNotFound) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
			return
		}

		// 3) Write to cache (best-effort) and return
		if gen != nil {
			log.Printf("Cache updated")
			_ = gen.SetURL(ctx, code, longURL, 24*time.Hour)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"redirect_url": longURL})
	}
}
