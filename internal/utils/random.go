package utils

import (
	"crypto/rand"
	"encoding/base64"
)

func GenerateRandomState() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "fallback_state"
	}
	return base64.URLEncoding.EncodeToString(b)
}
