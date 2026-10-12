package htmx

import (
	"net/http"
	"strconv"
)

// SelfSwap swaps the response over the element that issued the request.
//
// htmx 4 resolves HX-Retarget "this" through hx-target: it only finds the
// requesting element when that element, or an ancestor it inherits from,
// declares hx-target="this". Otherwise retarget an id with AttachTarget.
func SelfSwap(w http.ResponseWriter) {
	AttachTarget(w, "this")
	AttachSwap(w, SwapReplace)
}

func Redirect(w http.ResponseWriter, url string) error {
	AttachRedirect(w, url)
	return nil
}

func AttachLocation(w http.ResponseWriter, location string) {
	w.Header().Set(HeaderLocation, location)
}

func AttachPushURL(w http.ResponseWriter, url string) {
	w.Header().Set(HeaderPushURL, url)
}

func AttachRedirect(w http.ResponseWriter, url string) {
	w.Header().Set(HeaderRedirect, url)
}

func AttachRefresh(w http.ResponseWriter, refresh ...bool) {
	var r bool
	if len(refresh) > 0 {
		r = refresh[0]
	} else {
		r = true
	}
	w.Header().Set(HeaderRefresh, strconv.FormatBool(r))
}

func AttachSwap(w http.ResponseWriter, swap string) {
	w.Header().Set(HeaderReswap, swap)
}

func AttachTarget(w http.ResponseWriter, target string) {
	w.Header().Set(HeaderRetarget, target)
}

func AttachSelect(w http.ResponseWriter, target string) {
	w.Header().Set(HeaderReselect, target)
}

func AttachTrigger(w http.ResponseWriter, target string) {
	w.Header().Set(HeaderTrigger, target)
}

// Deprecated: removed in htmx 4. Use AttachTrigger - HX-Trigger is the only
// trigger response header left.
func AttachTriggerAfterSettle(w http.ResponseWriter, target string) {
	w.Header().Set(HeaderTrigger, target)
}

// Deprecated: removed in htmx 4. Use AttachTrigger - HX-Trigger is the only
// trigger response header left.
func AttachTriggerAfterSwap(w http.ResponseWriter, target string) {
	w.Header().Set(HeaderTrigger, target)
}
