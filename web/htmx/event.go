package htmx

// DOM event names, for Trigger and On.
const (
	Click      = "click"
	Input      = "input"
	Change     = "change"
	Blur       = "blur"
	Submit     = "submit"
	KeyUp      = "keyup"
	KeyDown    = "keydown"
	MouseUp    = "mouseup"
	MouseDown  = "mousedown"
	MouseEnter = "mouseenter"
	MouseOver  = "mouseover"
	MouseLeave = "mouseleave"
	Load       = "load"
	Revealed   = "revealed"
	Intersect  = "intersect"
)

// htmx 4 events. The naming scheme is htmx:phase:action[:sub-action].
//
// Every internally dispatched event carries a ctx object with request and
// response information. Events with detail.error go to console.error, those
// with detail.warn to console.warn; the rest are logged only when
// ConfigLogAll is set.
const (
	// Element lifecycle.
	EventBeforeInit    = "htmx:before:init"
	EventAfterInit     = "htmx:after:init"
	EventBeforeProcess = "htmx:before:process"
	EventAfterProcess  = "htmx:after:process"
	EventBeforeOnInit  = "htmx:before:on:init"
	EventBeforeCleanup = "htmx:before:cleanup"
	EventAfterCleanup  = "htmx:after:cleanup"

	// Request lifecycle.
	EventConfigRequest  = "htmx:config:request"
	EventConfirm        = "htmx:confirm"
	EventBeforeRequest  = "htmx:before:request"
	EventBeforeResponse = "htmx:before:response"
	EventAfterRequest   = "htmx:after:request"
	EventFinallyRequest = "htmx:finally:request"
	EventAbort          = "htmx:abort"
	// EventResponseError fires for HTTP error responses.
	EventResponseError = "htmx:response:error"
	// EventError is the consolidated error event: network failures, timeouts,
	// swap and target errors, and exceptions thrown in hx-on handlers.
	EventError = "htmx:error"

	// Swap lifecycle.
	EventBeforeSwap   = "htmx:before:swap"
	EventAfterSwap    = "htmx:after:swap"
	EventFinallySwap  = "htmx:finally:swap"
	EventBeforeSettle = "htmx:before:settle"
	EventAfterSettle  = "htmx:after:settle"

	// History.
	EventBeforeHistoryUpdate  = "htmx:before:history:update"
	EventAfterHistoryUpdate   = "htmx:after:history:update"
	EventAfterHistoryPush     = "htmx:after:history:push"
	EventAfterHistoryReplace  = "htmx:after:history:replace"
	EventBeforeHistoryRestore = "htmx:before:history:restore"

	// View transitions, enabled by ConfigTransitions.
	EventBeforeViewTransition = "htmx:before:viewTransition"
	EventAfterViewTransition  = "htmx:after:viewTransition"
)

// --------------------------------------------------------
// Renamed in htmx 4

// Deprecated: use EventAfterInit.
const (
	EventAfterOnLoad      = EventAfterInit
	EventAfterProcessNode = EventAfterInit
	EventLoad             = EventAfterInit
)

// Deprecated: use EventBeforeInit.
const EventBeforeOnLoad = EventBeforeInit

// Deprecated: use EventBeforeProcess.
const EventBeforeProcessNode = EventBeforeProcess

// Deprecated: use EventBeforeCleanup.
const EventBeforeCleanupElement = EventBeforeCleanup

// Deprecated: use EventAfterSwap - out-of-band swaps fire the same events as
// the main swap in htmx 4.
const EventOobAfterSwap = EventAfterSwap

// Deprecated: use EventBeforeSwap.
const EventOobBeforeSwap = EventBeforeSwap

// Deprecated: use EventBeforeHistoryUpdate.
const EventBeforeHistorySave = EventBeforeHistoryUpdate

// Deprecated: use EventBeforeHistoryRestore.
const (
	EventHistoryCacheMiss = EventBeforeHistoryRestore
	EventHistoryRestore   = EventBeforeHistoryRestore
)

// Deprecated: use EventAfterHistoryPush.
const EventPushedIntoHistory = EventAfterHistoryPush

// Deprecated: use EventAfterHistoryReplace.
const EventReplacedInHistory = EventAfterHistoryReplace

// Deprecated: use EventError, the consolidated error event.
const (
	EventSendError   = EventError
	EventSwapError   = EventError
	EventTargetError = EventError
	EventTimeout     = EventError
	EventXhrAbort    = EventError
)

// Deprecated: use EventFinallyRequest. htmx 4 uses fetch(), not XHR.
const EventXhrLoadEnd = EventFinallyRequest

// --------------------------------------------------------
// Removed in htmx 4

// Deprecated: removed in htmx 4. Use native browser form validation.
const (
	EventValidationValidate = "htmx:validation:validate"
	EventValidationFailed   = "htmx:validation:failed"
	EventValidationHalted   = "htmx:validation:halted"
)

// Deprecated: removed in htmx 4, which uses fetch() instead of XHR.
const (
	EventXhrLoadStart = "htmx:xhr:loadstart"
	EventXhrProgress  = "htmx:xhr:progress"
)

// Deprecated: removed in htmx 4. hx-prompt moved to PromptExt.
const EventPrompt = "htmx:prompt"

// Deprecated: removed in htmx 4. There is no localStorage history cache;
// HistoryCacheExt restores pages from sessionStorage.
const (
	EventHistoryCacheError     = "htmx:historyCacheError"
	EventHistoryCacheMissError = "htmx:historyCacheMissError"
	EventHistoryCacheMissLoad  = "htmx:historyCacheMissLoad"
)

// Deprecated: removed in htmx 4. Use EventError, or the SSEExt events.
const (
	EventNoSSESourceError = "htmx:noSSESourceError"
	EventOnLoadError      = "htmx:onLoadError"
	EventSseError         = "htmx:sseError"
	EventSseOpen          = "htmx:sseOpen"
	EventOobErrorNoTarget = "htmx:oobErrorNoTarget"
	EventBeforeSend       = "htmx:beforeSend"
)
