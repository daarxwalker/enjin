package htmx

// Swap styles for the Swap attribute. Build a value with modifiers using
// Value and Modifier:
//
//	htmx.Swap(): htmx.Value(htmx.SwapReplace, htmx.Modifier(htmx.ModSettle, "200ms"))
const (
	SwapReplace        = "outerHTML"
	SwapReplaceContent = "innerHTML"
	// SwapReplaceText sets the text content without parsing the response as
	// HTML.
	SwapReplaceText = "textContent"
	SwapBefore      = "before"
	SwapAfter       = "after"
	SwapPrepend     = "prepend"
	SwapAppend      = "append"
	// SwapDelete removes the target and ignores the response content.
	SwapDelete = "delete"
	SwapNone   = "none"
	// SwapMorph and SwapMorphOuter morph using idiomorph, preserving state
	// and focus. In htmx 2 this needed the morph extension.
	SwapMorph      = "innerMorph"
	SwapMorphOuter = "outerMorph"
	// SwapSync morphs the target's attributes, then replaces its children.
	SwapSync = "outerSync"
	// SwapUpsert updates elements by id and inserts new ones. Needs UpsertExt.
	SwapUpsert = "upsert"
	// SwapDownload saves the response as a file. Needs DownloadExt.
	SwapDownload = "download"
)

// Deprecated
const (
	SwapBeforeBegin = "beforebegin"
	SwapAfterBegin  = "afterbegin"
	SwapBeforeEnd   = "beforeend"
	SwapAfterEnd    = "afterend"
)

// Swap value modifier keys.
const (
	// ModSwapDelay delays the swap, e.g. Modifier(ModSwapDelay, "1s").
	ModSwapDelay = "swap"
	// ModSettle delays the settle phase. The default dropped from 20ms to 1ms
	// in htmx 4.
	ModSettle      = "settle"
	ModTransition  = "transition"
	ModIgnoreTitle = "ignoreTitle"
	ModFocusScroll = "focusScroll"
	// ModShow scrolls the target into view; ModScroll scrolls it. The htmx 2
	// combined form show:#other:top is gone - pass the position here and the
	// element in ModShowTarget / ModScrollTarget.
	ModShow         = "show"
	ModShowTarget   = "showTarget"
	ModScroll       = "scroll"
	ModScrollTarget = "scrollTarget"
	// ModStrip controls whether the response's outer element is stripped.
	ModStrip = "strip"
	// ModSwapEmpty forces the main swap even when the response only carries
	// <hx-partial> or out-of-band elements.
	ModSwapEmpty = "swapEmpty"
	// ModSwapTarget overrides the target for this swap.
	ModSwapTarget = "target"
)

// Positions for ModShow and ModScroll.
const (
	PositionTop          = "top"
	PositionBottom       = "bottom"
	PositionNone         = "none"
	PositionWindowTop    = "window:top"
	PositionWindowBottom = "window:bottom"
)

// Deprecated: these were never events - they are ModScroll/ModShow position
// values. Use PosTop, PosBottom, PosWindowTop, PosWindowBottom.
const (
	EventScrollTop    = "scroll:top"
	EventScrollBottom = "scroll:bottom"
	EventWindowTop    = "window:top"
	EventWindowBottom = "window:bottom"
)
