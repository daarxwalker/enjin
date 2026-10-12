package antibot

import "net/http"

var forbidden = http.HandlerFunc(
	func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	},
)
