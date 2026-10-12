package session

import (
	"errors"
	"net/http"

	"github.com/daarxwalker/enjin/token"
)

const (
	defaultCookieName = "X-Session"
)

func getTokenFromCookie(r *http.Request, cookieName string) (string, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		return "", err
	}
	if cookie == nil || !token.Valid(cookie.Value) {
		return "", nil
	}
	return cookie.Value, nil
}
