package token

import (
	"crypto/rand"
	"encoding/base64"
)

const (
	size   = 32
	Length = 43
)

func New() string {
	b := make([]byte, size)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func Valid(s string) bool {
	if len(s) != Length {
		return false
	}
	_, err := base64.RawURLEncoding.Strict().DecodeString(s)
	return err == nil
}
