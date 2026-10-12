package redirect

import (
	"context"
	"net/http"

	"github.com/daarxwalker/enjin/httpx"
)

func FromQuery(r *http.Request) string {
	paramValue := r.URL.Query().Get("redirect")
	if !httpx.IsLocalPath(paramValue) {
		return ""
	}
	return paramValue
}

func FromCookie(w http.ResponseWriter, r *http.Request) string {
	cv := from(r.Context())
	if cv == nil {
		return ""
	}
	cookie, err := r.Cookie(cv.manager.cookie.Name)
	if err != nil {
		return ""
	}
	cv.manager.cookie.Destroy(w)
	if !httpx.IsLocalPath(cookie.Value) {
		return ""
	}
	return cookie.Value
}

func from(ctx context.Context) *ctxVal {
	val := ctx.Value(key)
	if val == nil {
		return nil
	}
	cv, ok := val.(*ctxVal)
	if !ok {
		return nil
	}
	return cv
}
