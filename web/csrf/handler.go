package csrf

import "net/http"

var forbidden = http.HandlerFunc(
	func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	},
)

func submittedToken(r *http.Request) string {
	if value := r.PostFormValue(Field); value != "" {
		return value
	}
	return r.Header.Get(Header)
}
