package consent

import (
	"net/http"
	"net/url"
	"strconv"
)

func Attach(w http.ResponseWriter, r *http.Request, cookies Cookies) {
	ctxVal := from(r.Context())
	if ctxVal == nil {
		return
	}
	m := ctxVal.manager
	v := make(url.Values)
	v.Set(Analytic, strconv.FormatBool(cookies.Analytic))
	v.Set(Preference, strconv.FormatBool(cookies.Preference))
	v.Set(Marketing, strconv.FormatBool(cookies.Marketing))
	m.cookie.Attach(w, v.Encode())
}
