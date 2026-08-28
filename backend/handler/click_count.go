package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gocql/gocql"

	"url-shortner/store"
)

type clickCountResponse struct {
	ShortCode string `json:"short_code"`
	Clicks    int64  `json:"clicks"`
}

// NewGetClickCountHandler returns an http.HandlerFunc that returns the click
// count for a short code as JSON.
func NewGetClickCountHandler(db *store.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := chi.URLParam(r, "code")
		count, err := db.GetClickCount(r.Context(), code)
		if err != nil {
			if errors.Is(err, gocql.ErrNotFound) {
				http.Error(w, "No clicks found for the provided short code", http.StatusNotFound)
				return
			}
			log.Printf("Error getting click count for code %s: %v", code, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(clickCountResponse{
			ShortCode: code,
			Clicks:    count,
		})
	}
}
