package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewShortenHandler_RequestSecurityBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		payload     string
		allowedCodes []int
	}{
		{
			name: "oversized json body is rejected before processing",
			payload: `{"long_url":"https://example.com/` + strings.Repeat("a", 20000) + `"}`,
			allowedCodes: []int{http.StatusBadRequest, http.StatusRequestEntityTooLarge},
		},
		{
			name: "loopback ipv4 is rejected",
			payload: `{"long_url":"http://127.0.0.1:8080/"}`,
			allowedCodes: []int{http.StatusBadRequest},
		},
		{
			name: "private network address is rejected",
			payload: `{"long_url":"http://10.0.0.5/test"}`,
			allowedCodes: []int{http.StatusBadRequest},
		},
		{
			name: "cloud metadata endpoint is rejected",
			payload: `{"long_url":"http://169.254.169.254/latest/meta-data/"}`,
			allowedCodes: []int{http.StatusBadRequest},
		},
		{
			name: "file scheme is rejected",
			payload: `{"long_url":"file:///etc/passwd"}`,
			allowedCodes: []int{http.StatusBadRequest},
		},
		{
			name: "javascript scheme is rejected",
			payload: `{"long_url":"javascript:alert('xss')"}`,
			allowedCodes: []int{http.StatusBadRequest},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(tt.payload))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			NewShortenHandler(nil, nil).ServeHTTP(rec, req)

			if !containsStatus(tt.allowedCodes, rec.Code) {
				t.Fatalf("status code = %d, want one of %v; body=%s", rec.Code, tt.allowedCodes, rec.Body.String())
			}
		})
	}
}

func containsStatus(codes []int, code int) bool {
	for _, c := range codes {
		if c == code {
			return true
		}
	}
	return false
}
