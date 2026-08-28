package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gocql/gocql"

	"url-shortner/base62"
	"url-shortner/redisid"
	"url-shortner/store"
)

// NewRedirectHandler returns a public handler that redirects a short code to
// its stored long URL and records the click asynchronously.
func NewRedirectHandler(db *store.DB, gen *redisid.Generator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := chi.URLParam(r, "code")
		if _, err := base62.Decode(code); err != nil {
			http.Error(w, "invalid short code", http.StatusBadRequest)
			return
		}

		var longURL string
		if gen != nil {
			cachedURL, err := gen.GetURL(r.Context(), code)
			if err != nil {
				log.Printf("warning: redirect cache lookup failed for %s: %v", code, err)
			} else {
				longURL = cachedURL
			}
		}

		if longURL == "" {
			var err error
			longURL, err = db.GetLongURLByShortCode(r.Context(), code)
			if err != nil {
				if errors.Is(err, gocql.ErrNotFound) {
					http.Error(w, "not found", http.StatusNotFound)
					return
				}
				http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
				return
			}

			if gen != nil {
				if err := gen.SetURL(r.Context(), code, longURL, 24*time.Hour); err != nil {
					log.Printf("warning: redirect cache update failed for %s: %v", code, err)
				}
			}
		}

		go func(code string) {
			_ = db.IncrementClickCount(context.Background(), code)
		}(code)

		http.Redirect(w, r, longURL, http.StatusFound)
	}
}
