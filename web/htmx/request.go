package htmx

import (
	"net/http"
)

// Is reports whether the request came from htmx.
func Is(r *http.Request) bool {
	return r.Header.Get(HeaderRequest) == "true"
}

// IsBoosted reports whether the request came from a boosted link or form.
func IsBoosted(r *http.Request) bool {
	return r.Header.Get(HeaderBoosted) == "true"
}

// IsPartial reports whether htmx expects a fragment rather than a full page.
func IsPartial(r *http.Request) bool {
	return r.Header.Get(HeaderRequestType) == HeaderRequestTypePartial
}

// IsFull reports whether htmx expects a full page, as it does for boosted
// navigation and history restores.
func IsFull(r *http.Request) bool {
	return r.Header.Get(HeaderRequestType) == HeaderRequestTypeFull
}

// IsHistoryRestore reports whether htmx is restoring a history entry.
func IsHistoryRestore(r *http.Request) bool {
	return r.Header.Get(HeaderHistoryRestoreRequest) == "true"
}

// CurrentURLFrom returns the browser URL the request was made from.
func CurrentURLFrom(r *http.Request) string {
	return r.Header.Get(HeaderCurrentURL)
}

// SourceFrom returns the triggering element as tagName#id, e.g.
// "button#submit".
func SourceFrom(r *http.Request) string {
	return r.Header.Get(HeaderSource)
}

// TargetFrom returns the target element as tagName#id, e.g. "div#messages".
func TargetFrom(r *http.Request) string {
	return r.Header.Get(HeaderTarget)
}
