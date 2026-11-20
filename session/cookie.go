package session

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/daarxwalker/enjin/token"
)

const (
	defaultCookieName = "X-Session"
)

func getTokenFromCookie(r *http.Request, cookieName string) (string, error) {
	var zero string

	cookie, err := r.Cookie(cookieName)
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		return zero, fmt.Errorf("read cookie %s failed: %w", cookieName, err)
	}
	if cookie == nil || !token.Valid(cookie.Value) {
		return "", nil
	}
	return cookie.Value, nil
}
