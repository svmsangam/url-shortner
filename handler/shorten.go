package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
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

const maxShortenBodyBytes int64 = 10 * 1024

func validateLongURL(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return errors.New("long_url is required")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("invalid long_url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("long_url must be an absolute http(s) URL")
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s", scheme)
	}

	host := strings.TrimSpace(strings.ToLower(parsed.Hostname()))
	if host == "" {
		return errors.New("long_url has no host")
	}
	if host == "localhost" || host == "localhost.localdomain" {
		return fmt.Errorf("forbidden host: %s", host)
	}

	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.Equal(net.ParseIP("169.254.169.254")) {
			return fmt.Errorf("forbidden IP address: %s", host)
		}
	}

	return nil
}

// NewShortenHandler returns an http.HandlerFunc that creates a short URL.
// db: Cassandra DB wrapper; rdb: Redis client
func NewShortenHandler(db *store.DB, gen *redisid.Generator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxShortenBodyBytes)
		}

		var req ShortenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if err := validateLongURL(req.LongURL); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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
