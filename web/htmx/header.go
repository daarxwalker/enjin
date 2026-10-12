package htmx

// Request headers htmx 4 sends.
const (
	HeaderRequest    = "HX-Request"
	HeaderBoosted    = "HX-Boosted"
	HeaderCurrentURL = "HX-Current-URL"
	// HeaderRequestType is "full" or "partial".
	HeaderRequestType = "HX-Request-Type"
	// HeaderSource identifies the triggering element as tagName#id, e.g.
	// "button#submit". Replaces the htmx 2 request-side HX-Trigger and
	// HX-Trigger-Name.
	HeaderSource = "HX-Source"
	// HeaderTarget identifies the target element. The format changed in
	// htmx 4 from a bare id to tagName#id, e.g. "div#messages".
	HeaderTarget                = "HX-Target"
	HeaderHistoryRestoreRequest = "HX-History-Restore-Request"
)

// HeaderRequestType values.
const (
	HeaderRequestTypeFull    = "full"
	HeaderRequestTypePartial = "partial"
)

// Response headers htmx 4 honors. See response.go for the setters.
const (
	HeaderLocation = "HX-Location"
	HeaderPushURL  = "HX-Push-Url"
	HeaderRedirect = "HX-Redirect"
	HeaderRefresh  = "HX-Refresh"
	HeaderReswap   = "HX-Reswap"
	HeaderRetarget = "HX-Retarget"
	HeaderReselect = "HX-Reselect"
	HeaderTrigger  = "HX-Trigger"
)

// Deprecated: removed in htmx 4; the triggering element arrives in
// HeaderSource as tagName#id.
const HeaderTriggerName = "HX-Trigger-Name"

// HeaderPrompt is only sent when PromptExt is loaded - hx-prompt moved out of
// the core in htmx 4.
const HeaderPrompt = "HX-Prompt"

// Deprecated: removed in htmx 4. Both fold into HeaderTrigger.
const (
	HeaderTriggerAfterSwap   = "HX-Trigger-After-Swap"
	HeaderTriggerAfterSettle = "HX-Trigger-After-Settle"
)
