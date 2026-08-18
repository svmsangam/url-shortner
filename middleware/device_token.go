package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
	"url-shortner/utils"
)

// deviceTokenContextKey is an unexported type for context keys in this package.
// Using an unexported type avoids collisions with other context keys.
type deviceTokenContextKey struct{}

// DeviceTokenContextKey is the context key that middleware uses to store the
// device token string. Use GetDeviceToken(ctx) to retrieve it.
var DeviceTokenContextKey = deviceTokenContextKey{}

// GetDeviceToken returns the device token stored in ctx (if any). The boolean
// indicates whether a token was present.
func GetDeviceToken(ctx context.Context) (string, bool) {
	v := ctx.Value(DeviceTokenContextKey)
	if v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// DeviceTokenMiddleware checks for an HttpOnly cookie named "device_token".
// If missing, it generates a UUID v4, sets it as a Secure, HttpOnly cookie on
// the response, and stores the token string in the request context. The token
// is available in handlers via GetDeviceToken(r.Context()).
func DeviceTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const cookieName = "device_token"

		var token string
		c, err := r.Cookie(cookieName)
		if err == nil && c != nil && c.Value != "" {
			token = c.Value
		} else {
			// generate a UUID v4
			u, genErr := utils.GenerateUUID()
			if genErr != nil {
				// Fallback: try to produce a random hex string (non-UUID) — still
				// acceptable as a token even if UUID generation unexpectedly fails.
				b := make([]byte, 16)
				if _, re := rand.Read(b); re == nil {
					token = hex.EncodeToString(b)
				} else {
					// As last resort, set token to an empty string and continue; this
					// should be extremely rare.
					token = ""
				}
			} else {
				token = u
				// Set secure, HttpOnly cookie. Adjust attributes (Domain, Path,
				// MaxAge, Expires, SameSite) as needed by your application.
				cookie := &http.Cookie{
					Name:     cookieName,
					Value:    token,
					Path:     "/",
					HttpOnly: true,
					Secure:   true,
					// Long expiration: 10 years
					Expires:  time.Now().Add(10 * 365 * 24 * time.Hour),
					SameSite: http.SameSiteLaxMode,
				}
				http.SetCookie(w, cookie)
			}
		}

		// Store token in request context for downstream handlers
		ctx := context.WithValue(r.Context(), DeviceTokenContextKey, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// // newUUID4 generates a RFC4122-compliant UUIDv4 using crypto/rand and returns
// // it as the canonical 36-char hyphenated string (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx).
// func newUUID4() (string, error) {
// 	b := make([]byte, 16)
// 	if _, err := rand.Read(b); err != nil {
// 		return "", fmt.Errorf("uuid: rand read failed: %w", err)
// 	}
// 	// Set version (4) and variant (RFC4122)
// 	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
// 	b[8] = (b[8] & 0x3f) | 0x80 // Variant is 10xxxxxx

// 	// Format into canonical representation
// 	var out [36]byte
// 	hex.Encode(out[0:8], b[0:4])
// 	out[8] = '-'
// 	hex.Encode(out[9:13], b[4:6])
// 	out[13] = '-'
// 	hex.Encode(out[14:18], b[6:8])
// 	out[18] = '-'
// 	hex.Encode(out[19:23], b[8:10])
// 	out[23] = '-'
// 	hex.Encode(out[24:36], b[10:16])

// 	return string(out[:]), nil
// }
