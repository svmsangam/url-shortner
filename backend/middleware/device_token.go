package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"strings"

	"url-shortner/utils"

	"github.com/google/uuid"
)

const HeaderDeviceToken = "X-Device-Token"

// deviceTokenContextKey is an unexported type for context keys in this package.
type deviceTokenContextKey struct{}

// DeviceTokenContextKey is the context key that middleware uses to store the
// device token string. Use GetDeviceToken(ctx) to retrieve it.
var DeviceTokenContextKey = deviceTokenContextKey{}

// isValidUUID checks if a string matches a valid UUID v4 format.
func isValidUUID(u string) bool {
	parsed, err := uuid.Parse(u)
	if err != nil {
		return false
	}
	return parsed.Version() == 4
}

// GetDeviceToken retrieves the device token stored in ctx (if any) and validates
// that it conforms to UUID v4. Returns ("", false) if missing or invalid.
func GetDeviceToken(ctx context.Context) (string, bool) {
	v := ctx.Value(DeviceTokenContextKey)
	if v == nil {
		return "", false
	}
	s, ok := v.(string)
	if !ok || !isValidUUID(s) {
		return "", false
	}
	return s, true
}

// DeviceTokenMiddleware checks for the "X-Device-Token" header.
// If missing or not a valid UUID v4, it generates a new UUID v4, attaches
// "X-Device-Token" to the response headers, and stores the validated token
// in the request context for downstream handlers.
func DeviceTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string
		headerVal := strings.TrimSpace(r.Header.Get(HeaderDeviceToken))

		if headerVal != "" && isValidUUID(headerVal) {
			log.Printf("Using existing device token: %s", headerVal)
			token = headerVal
		} else {
			// Generate a new UUID v4 whenever the client sends an invalid or missing value.
			u, genErr := utils.GenerateUUID()
			if genErr != nil {
				// Fallback: produce a random hex string (non-UUID)
				b := make([]byte, 16)
				if _, re := rand.Read(b); re == nil {
					token = hex.EncodeToString(b)
				} else {
					token = ""
				}
			} else {
				token = u
			}
			log.Printf("Generated new device token: %s", token)
			// Return the newly generated token back to client via response header
			if token != "" {
				w.Header().Set(HeaderDeviceToken, token)
			}
		}

		// Store token in request context for downstream handlers
		ctx := context.WithValue(r.Context(), DeviceTokenContextKey, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
