package redirect

import (
	"net/http"
)

func Attach(w http.ResponseWriter, r *http.Request, to string) {
	cv := from(r.Context())
	if cv == nil {
		return
	}
	cv.manager.cookie.Attach(w, to)
}
