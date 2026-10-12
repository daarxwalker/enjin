package session

import (
	"net/http"
	"strings"

	"github.com/daarxwalker/enjin/token"
)

func getTokenFromHeader(r *http.Request) string {
	scheme, value, found := strings.Cut(r.Header.Get("Authorization"), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || !token.Valid(value) {
		return ""
	}
	return value
}
