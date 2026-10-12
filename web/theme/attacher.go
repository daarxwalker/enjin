package theme

import (
	"net/http"
)

func Attach(w http.ResponseWriter, r *http.Request, value Value) {
	ctxVal := from(r.Context())
	if ctxVal == nil {
		return
	}
	if _, ok := parseValue(string(value)); !ok {
		return
	}
	m := ctxVal.manager
	ctxVal.value = value
	m.cookie.Attach(w, value.String())
}
