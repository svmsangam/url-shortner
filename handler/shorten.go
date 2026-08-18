package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"url-shortner/middleware"
	"url-shortner/redisid"
	"url-shortner/store"
)

// ShortenRequest is the expected JSON body for POST /api/v1/shorten
type ShortenRequest struct {
	LongURL string `json:"long_url"`
}

// ShortenResponse is returned on success with 201 Created
type ShortenResponse struct {
	ShortURL    string    `json:"short_url"`
	ShortCode   string    `json:"short_code"`
	LongURL     string    `json:"long_url"`
	CreatedAt   time.Time `json:"created_at"`
	DeviceToken string    `json:"device_token,omitempty"`
}

// NewShortenHandler returns an http.HandlerFunc that creates a short URL.
// db: Cassandra DB wrapper; rdb: Redis client
func NewShortenHandler(db *store.DB, gen *redisid.Generator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var req ShortenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if req.LongURL == "" {
			http.Error(w, "long_url is required", http.StatusBadRequest)
			return
		}

		deviceToken, _ := middleware.GetDeviceToken(ctx)

		mapping, err := db.CreateShortLink(ctx, gen, req.LongURL, deviceToken)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to create short link: %v", err), http.StatusInternalServerError)
			return
		}

		// Build short URL using request host and scheme
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		shortURL := fmt.Sprintf("%s://%s/%s", scheme, r.Host, mapping.ShortCode)

		w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			resp := ShortenResponse{
				ShortURL:    shortURL,
				ShortCode:   mapping.ShortCode,
				LongURL:     mapping.LongURL,
				CreatedAt:   mapping.CreatedAt,
				DeviceToken: deviceToken, // Include device token in response if present
			}
			_ = json.NewEncoder(w).Encode(resp)
	}
}
