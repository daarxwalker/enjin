package htmx

// Per-status-code response handling, built with Status.
//
//	htmx.Status("422"): htmx.Modifiers(
//	    htmx.Modifier(htmx.StatusSwap, htmx.SwapReplaceContent),
//	    htmx.Modifier(htmx.StatusTarget, "#errors"),
//	)
//
// Status codes support exact values ("404"), 2-digit wildcards ("50x") and
// 1-digit wildcards ("5xx"), evaluated in that order of specificity.
//
// This replaces the htmx 2 response-targets extension.
//
// Note that htmx 4 swaps every response except 204 and 304, including 4xx and
// 5xx - so error responses have to work as swap content. Use Status per code,
// or set ConfigNoSwap to ["204", "304", "4xx", "5xx"] for the htmx 2 behavior.
const (
	StatusSwap       = "swap"
	StatusTarget     = "target"
	StatusSelect     = "select"
	StatusPush       = "push"
	StatusReplace    = "replace"
	StatusTransition = "transition"
)

// Status code wildcards for Status.
const (
	Status4xx = "4xx"
	Status5xx = "5xx"
)

// Deprecated: the response-targets extension was replaced by Status in
// htmx 4. Status("404", Modifier(StatusTarget, "#err")) instead of
// StatusTarget("404").
func StatusTargetAttr(statusPattern string) string {
	return "hx-target-" + statusPattern
}

// Deprecated: use Status(Status4xx, ...) / Status(Status5xx, ...).
const ErrorTarget = "hx-target-error"
