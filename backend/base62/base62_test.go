package base62

import (
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 10000; i++ {
		n := rand.Uint64()
		s := Encode(n)
		got, err := Decode(s)
		if err != nil {
			t.Fatalf("Decode returned error for %q: %v", s, err)
		}
		if got != n {
			t.Fatalf("round-trip mismatch: got %d want %d (s=%q)", got, n, s)
		}
	}
}

func TestKnownValues(t *testing.T) {
	tests := []struct{ n uint64; s string }{
		{0, "0"},
		{1, "1"},
		{9, "9"},
		{10, "a"},
		{35, "z"},
		{36, "A"},
		{61, "Z"},
		{62, "10"},
		{3843, "ZZ"},
	}
	for _, tt := range tests {
		got := Encode(tt.n)
		if got != tt.s {
			t.Fatalf("Encode(%d) = %q; want %q", tt.n, got, tt.s)
		}
		back, err := Decode(tt.s)
		if err != nil {
			t.Fatalf("Decode(%q) error: %v", tt.s, err)
		}
		if back != tt.n {
			t.Fatalf("Decode(%q) = %d; want %d", tt.s, back, tt.n)
		}
	}
}

func TestDecodeInvalid(t *testing.T) {
	_, err := Decode("!")
	if err == nil {
		t.Fatalf("expected error for invalid char")
	}
}

func TestDecodeOverflow(t *testing.T) {
	// produce a string that is too large: e.g., 20 chars of 'Z' (highest digit)
	s := strings.Repeat("Z", 20)
	_, err := Decode(s)
	if err == nil {
		t.Fatalf("expected overflow error for %q", s)
	}
}
