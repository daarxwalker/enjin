package token

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const (
	size   = 32
	Length = 43
)

func New() string {
	b := make([]byte, size)
	// Since Go 1.24 crypto/rand.Read crashes the program instead of returning
	// an error, so this branch is unreachable.
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Errorf("read random token bytes failed: %w", err))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func Valid(s string) bool {
	if len(s) != Length {
		return false
	}
	_, err := base64.RawURLEncoding.Strict().DecodeString(s)
	return err == nil
}
