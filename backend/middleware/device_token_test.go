package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestDeviceTokenMiddleware_HeaderValidation(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"

	tests := []struct {
		name                string
		headerValue         string
		wantContext         bool
		wantResponseHeader  bool
		expectNormalizedVal string
	}{
		{
			name:                "valid uuid v4 passes through without changing value",
			headerValue:         validUUID,
			wantContext:         true,
			wantResponseHeader:  false,
			expectNormalizedVal: validUUID,
		},
		{
			name:                "valid uuid with whitespace is trimmed and accepted",
			headerValue:         "  " + validUUID + "  ",
			wantContext:         true,
			wantResponseHeader:  false,
			expectNormalizedVal: validUUID,
		},
		{
			name:               "missing header generates a fresh uuid v4",
			headerValue:        "",
			wantContext:        true,
			wantResponseHeader: true,
		},
		{
			name:               "short string is discarded and replaced",
			headerValue:        "short",
			wantContext:        true,
			wantResponseHeader: true,
		},
		{
			name:               "sql injection payload is discarded and replaced",
			headerValue:        "'; DROP TABLE users; --",
			wantContext:        true,
			wantResponseHeader: true,
		},
		{
			name:               "random text is discarded and replaced",
			headerValue:        "not-a-valid-uuid-value",
			wantContext:        true,
			wantResponseHeader: true,
		},
		{
			name:               "html script is discarded and replaced",
			headerValue:        "<script>alert('xss')</script>",
			wantContext:        true,
			wantResponseHeader: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.headerValue != "" {
				req.Header.Set(HeaderDeviceToken, tt.headerValue)
			}

			rr := httptest.NewRecorder()
			var token string
			var ok bool

			h := DeviceTokenMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				token, ok = GetDeviceToken(r.Context())
				w.WriteHeader(http.StatusNoContent)
			}))

			h.ServeHTTP(rr, req)

			if ok != tt.wantContext {
				t.Fatalf("GetDeviceToken ok = %v, want %v", ok, tt.wantContext)
			}
			if !tt.wantContext {
				return
			}

			if _, err := uuid.Parse(token); err != nil {
				t.Fatalf("device token %q is not a valid UUID v4: %v", token, err)
			}
			if tt.expectNormalizedVal != "" && token != tt.expectNormalizedVal {
				t.Fatalf("device token = %q, want %q", token, tt.expectNormalizedVal)
			}

			responseHeader := rr.Header().Get(HeaderDeviceToken)
			if tt.wantResponseHeader != (responseHeader != "") {
				t.Fatalf("response header present = %v, want %v (header=%q)", responseHeader != "", tt.wantResponseHeader, responseHeader)
			}
			if responseHeader != "" {
				if _, err := uuid.Parse(responseHeader); err != nil {
					t.Fatalf("generated response header %q is not valid UUID v4: %v", responseHeader, err)
				}
			}
		})
	}
}

func TestDeviceTokenMiddleware_SanitizesHeaderWhitespace(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderDeviceToken, "  "+validUUID+"  ")

	var token string
	var ok bool

	DeviceTokenMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok = GetDeviceToken(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(httptest.NewRecorder(), req)

	if !ok {
		t.Fatal("device token should be present after trimming whitespace")
	}
	if strings.TrimSpace(token) != validUUID {
		t.Fatalf("token %q should equal trimmed UUID %q", token, validUUID)
	}
	if token != validUUID {
		t.Fatalf("token %q should not preserve surrounding spaces", token)
	}
}
