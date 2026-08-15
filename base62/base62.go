package base62

import (
	"errors"
	"math"
)

const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const base = uint64(62)

var decodeMap [256]uint8

func init() {
	for i := range decodeMap {
		decodeMap[i] = 0xFF
	}
	for i := 0; i < len(alphabet); i++ {
		decodeMap[alphabet[i]] = uint8(i)
	}
}

// Encode converts a uint64 to a Base62 string using alphabet 0-9, a-z, A-Z.
// Example: Encode(0) == "0"
func Encode(u uint64) string {
	if u == 0 {
		return "0"
	}
	// Maximum length needed is ceil(log_{62}(2^64)) ~= 11
	buf := make([]byte, 0, 11)
	for u > 0 {
		rem := u % base
		buf = append(buf, alphabet[rem])
		u /= base
	}
	// reverse
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}

// Decode parses a Base62 string produced by Encode back into a uint64.
// Returns an error if the string contains invalid characters or represents a value
// that does not fit into a uint64.
func Decode(s string) (uint64, error) {
	if len(s) == 0 {
		return 0, errors.New("empty string")
	}
	var val uint64
	for i := 0; i < len(s); i++ {
		c := s[i]
		idx := decodeMap[c]
		if idx == 0xFF {
			return 0, errors.New("invalid character in input")
		}
		d := uint64(idx)
		// check overflow: val*base + d > MaxUint64
		if val > (math.MaxUint64-d)/base {
			return 0, errors.New("overflow")
		}
		val = val*base + d
	}
	return val, nil
}
