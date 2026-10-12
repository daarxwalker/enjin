package htmx

// Core extensions shipped with htmx 4. There is no hx-ext attribute any more -
// include the extension script after htmx itself:
//
//	<script src="/htmx.js"></script>
//	<script src="/ext/hx-sse.js"></script>
//
// With a bundler, import the script from the package; htmx.org sets the global
// htmx the extensions register on, so its import has to come first:
//
//	import htmx from 'htmx.org'
//	import 'htmx.org/dist/ext/hx-sse.js'
//
// The constants below are the names the extensions register under, which is
// what ConfigExtensions matches exactly. They are not the script file names:
// SSEExt is "sse" but the script is dist/ext/hx-sse.js.
// Extension authors use htmx.registerExtension(name, methodMap).
const (
	// Streaming HTML.
	SSEExt       = "sse"          // hx-sse.js
	WSExt        = "ws"           // hx-ws.js
	MultipartExt = "hx-multipart" // hx-multipart.js

	// UX.
	LiveExt             = "hx-live"           // hx-live.js
	PendingExt          = "hx-pending"        // hx-pending.js
	BrowserIndicatorExt = "browser-indicator" // hx-browser-indicator.js
	PromptExt           = "hx-prompt"         // hx-prompt.js

	// Performance.
	PreloadExt      = "preload"       // hx-preload.js
	PtagExt         = "ptag"          // hx-ptag.js
	HistoryCacheExt = "history-cache" // hx-history-cache.js

	// Swaps.
	HeadExt     = "hx-head"    // hx-head.js
	UpsertExt   = "upsert"     // hx-upsert.js
	TargetsExt  = "hx-targets" // hx-targets.js
	DownloadExt = "download"   // hx-download.js

	// Security.
	CSPExt = "hx-csp" // hx-csp.js

	// Compatibility.
	Compat2Ext      = "compat"        // htmx-2-compat.js
	AlpineCompatExt = "alpine-compat" // hx-alpine-compat.js
)

// --------------------------------------------------------
// hx-sse

// Stream HTML over text/event-stream. Unnamed events swap automatically using
// the connection element's Target and Swap; named events are dispatched as DOM
// events instead, so On or a Trigger with From("body") picks them up.
//
// Migrating from the htmx 2 sse extension:
//
//	sse-connect             -> SSEConnect  (hx-sse:connect)
//	sse-close               -> SSEClose    (hx-sse:close)
//	sse-swap="message"      -> unnamed messages swap on their own
//	hx-trigger="sse:progress" -> Trigger: Value("progress", From("body"))
const (
	SSEConnect = "hx-sse:connect"
	SSEClose   = "hx-sse:close"
)

// Deprecated: renamed to SSEConnect in htmx 4.
const SSEConnectV2 = "sse-connect"

// Deprecated: removed in htmx 4 - unnamed SSE messages swap automatically and
// named ones are dispatched as DOM events.
const SSESwap = "sse-swap"

// --------------------------------------------------------
// hx-ws

// Stream HTML and send data over WebSockets. Incoming HTML swaps into the
// connection element using inherited Target, Swap and Select; use SwapOob or
// the Partial component to reach other elements. Outgoing messages put form
// values and Vals at the top level and reserve the "headers" key for htmx
// metadata.
const (
	WSConnect = "hx-ws:connect"
	WSSend    = "hx-ws:send"
)

// Deprecated: renamed to WSConnect in htmx 4.
const WSConnectV2 = "ws-connect"

// Deprecated: renamed to WSSend in htmx 4.
const WSSendV2 = "ws-send"

// --------------------------------------------------------
// hx-live, hx-pending, hx-browser-indicator, hx-prompt

const (
	// Live holds a reactive expression. Inside it take, forEvent, nextFrame,
	// q, debounce and toggle are available unprefixed; from JavaScript they
	// live on the htmx.live namespace. This is where htmx.takeClass went.
	Live = "hx-live"
	// Pending holds content shown while a request is in flight.
	Pending = "hx-pending"
	// BrowserIndicator shows the browser tab's own spinner.
	BrowserIndicator = "hx-browser-indicator"
	// PromptAttr restores the removed hx-prompt attribute, along with the
	// HeaderPrompt request header.
	PromptAttr = "hx-prompt"
)

// --------------------------------------------------------
// hx-preload, hx-ptag

// Preload takes the event that triggers preloading, e.g. MouseOver.
const Preload = "hx-preload"

// HeaderPTag lets the server skip unchanged polling responses: the client
// echoes the tag it holds and the server answers 304.
const HeaderPTag = "HX-PTag"

// Deprecated: htmx 4 exposes hx-preload as a plain attribute; use Preload and
// set the value yourself.
func PreloadOn(event ...string) string {
	if len(event) == 0 {
		return Preload
	}
	return Preload + "=" + event[0]
}

// Deprecated: removed in htmx 4.
const PreloadImagesAttr = "preload-images"

// --------------------------------------------------------
// hx-head

// Head merges <head> tags from the response.
const (
	Head      = "hx-head"
	HeadMerge = "merge"
)

// Deprecated: the head-support extension is HeadExt ("hx-head") in htmx 4.
const HeadSupportExt = "head-support"

// Deprecated: removed in htmx 4.
const (
	HeadAppend = "append"
	HeadReEval = "re-eval"
)

// Deprecated: removed in htmx 4; head merging no longer fires these.
const (
	HeadEventRemovingElement = "htmx:removingHeadElement"
	HeadEventAddingElement   = "htmx:addingHeadElement"
	HeadEventBeforeMerge     = "htmx:beforeHeadMerge"
	HeadEventAfterMerge      = "htmx:afterHeadMerge"
)

// --------------------------------------------------------
// hx-targets

// Targets takes a selector matching many elements, all of which receive the
// response. For distributing different content to different elements use the
// Partial component instead.
const Targets = "hx-targets"

// --------------------------------------------------------
// hx-alpine-compat

// Alpine and htmx 4 need this extension to coexist: Alpine's mutation
// observer otherwise fires during the settle phase and initializes components
// against incomplete DOM, morph swaps lose Alpine's reactive data, and
// Alpine's dynamic id bindings confuse the morph algorithm.
//
// Load order matters - Alpine must come after the extension script, which the
// defer attribute guarantees:
//
//	<script src="/htmx.js"></script>
//	<script src="/ext/hx-alpine-compat.js"></script>
//	<script defer src="/alpine.js"></script>
//
// When Alpine is bundled, import the extension before calling Alpine.start().

// --------------------------------------------------------
// Removed in htmx 4

// Deprecated: morphing moved into the core. Use SwapMorph, SwapMorphOuter or
// SwapSync, tuned with ConfigMorphSkip and friends.
const MorphExt = "morph"

// Deprecated: morphing moved into the core; use SwapMorph or SwapMorphOuter
// together with AlpineCompatExt.
const AlpineMorphExt = "alpine-morph"

// Deprecated: use the Status attribute.
const ResponseTargetsExt = "response-targets"

// Deprecated: use Targets or the Partial component.
const MultiSwapExt = "multi-swap"

// Deprecated: use Pending for custom pending content and Disable to disable
// elements during a request.
const LoadingStatesExt = "loading-states"

// Deprecated: replaced by the hx-prompt extension, PromptExt.
const PromptExtV2 = "prompt"

// Deprecated: removed in htmx 4. Set ConfigLogAll instead.
const DebugExt = "debug"

// Deprecated: removed in htmx 4. Indicator CSS uses constructable
// stylesheets, and inline scripts take ConfigInlineScriptNonce.
const SafeNonceExt = "safe-nonce"

// Deprecated: removed in htmx 4 with no replacement in the core extension
// set. Reach for the EventConfigRequest event, Config or HistoryCacheExt
// depending on what the extension was doing for you.
const (
	EventHeaderExt   = "event-header"
	RestoredExt      = "restored"
	Restored         = "restored"
	ReplaceParamsExt = "replace-params"
	PathParamsExt    = "path-params"
	PathDepsExt      = "path-deps"
	NoCacheExt       = "no-cache"
	Nonce            = "nonce"
)

// Deprecated: htmx 4 ships Compat2Ext (htmx-2-compat.js); there is no htmx 1
// compatibility layer any more.
const Compat1Ext = "htmx-1-compat"

// Deprecated: removed in htmx 4; use Pending and Disable.
const (
	Loading            = "data-loading"
	LoadingClass       = "data-loading-class"
	LoadingClassRemove = "data-loading-class-remove"
	LoadingDisable     = "data-loading-disable"
	LoadingAriaBusy    = "data-loading-aria-busy"
	LoadingDelay       = "data-loading-delay"
	LoadingTarget      = "data-loading-target"
	LoadingPath        = "data-loading-path"
	LoadingStates      = "data-loading-states"
)

// Deprecated: morphing moved into the core; use SwapMorph or SwapMorphOuter
// as the Swap value.
func Morph(values ...string) string {
	return Mod("morph", values...)
}

// Deprecated: use Targets or the Partial component.
func MultiSwap(targets ...string) string {
	if len(targets) == 0 {
		return ""
	}
	return "multi:" + joinComma(targets)
}
