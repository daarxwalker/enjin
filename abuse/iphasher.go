package abuse

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

const (
	ipHashSaltLen = 32
	ipHashLen     = 8
	ipHashDay     = 24 * 60 * 60
)

type ipHasher struct {
	salt []byte
	day  int64
	mu   sync.Mutex
}

func (h *ipHasher) hashIP(value string, now time.Time) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if day := now.Unix() / ipHashDay; day != h.day || h.salt == nil {
		h.day = day
		h.salt = h.createSalt()
	}
	mac := hmac.New(sha256.New, h.salt)
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil)[:ipHashLen])
}

func (h *ipHasher) createSalt() []byte {
	salt := make([]byte, ipHashSaltLen)
	// Since Go 1.24 crypto/rand.Read crashes the program instead of returning
	// an error, so this branch is unreachable.
	if _, err := rand.Read(salt); err != nil {
		panic(fmt.Errorf("read random ip hash salt failed: %w", err))
	}
	return salt
}
