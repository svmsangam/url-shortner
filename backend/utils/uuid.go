// Package utils contains small shared helpers that do not own application
// state, including device-token generation.
package utils

import "github.com/google/uuid"

// GenerateUUID generates and returns a unique UUID v4 string.
func GenerateUUID() (string, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return uuid.String(), nil
}
