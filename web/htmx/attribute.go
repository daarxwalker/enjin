package htmx

// Attribute names for htmx 4.
//
// Attributes whose inheritance is useful in practice are exposed as variadic
// functions taking modifiers, because htmx 4 made inheritance explicit:
//
//	htmx.Target()                  // "hx-target"
//	htmx.Target(htmx.ModInherited) // "hx-target:inherited"
//
// The remaining attributes stay constants; use Mod to attach a modifier to
// them, e.g. Mod(Trigger, ModInherited).

// Inheritance modifiers. ModInherited makes the attribute apply to every
// descendant, ModAppend adds to an inherited value instead of replacing it.
// They combine: Vals(ModInherited, ModAppend) -> "hx-vals:inherited:append".
const (
	ModInherited = "inherited"
	ModAppend    = "append"
)

// Request-issuing attributes. These are per-element and never inherited.
const (
	Get    = "hx-get"
	Post   = "hx-post"
	Put    = "hx-put"
	Patch  = "hx-patch"
	Delete = "hx-delete"
	// Query issues a QUERY request, which sends parameters in the body.
	Query = "hx-query"
	// Action holds the URL only, mirroring the native form attribute, and
	// pairs with Method. Enables progressive enhancement: without htmx the
	// browser falls back to native action/method.
	Action = "hx-action"
	// Method holds the HTTP verb for Action and overrides the native method
	// and formmethod attributes.
	Method = "hx-method"
)

// Attributes without inheritance modifiers.
const (
	Select     = "hx-select"
	SelectOob  = "hx-select-oob"
	SwapOob    = "hx-swap-oob"
	Trigger    = "hx-trigger"
	PushURL    = "hx-push-url"
	ReplaceURL = "hx-replace-url"
	HistoryElt = "hx-history-elt"
	Encoding   = "hx-encoding"
	Preserve   = "hx-preserve"
	// Validate set to "true" enables HTML5 validation on non-form elements.
	Validate = "hx-validate"
	// Ignore stops htmx from processing the element and its descendants. It
	// cannot be undone by injected content, which makes it the tool for
	// rendering untrusted HTML. This was hx-disable in htmx 2.
	Ignore = "hx-ignore"
	// MorphSkip freezes the element during a morph swap.
	MorphSkip = "hx-morph-skip"
	// MorphSkipChildren morphs the element's attributes but keeps its children.
	MorphSkipChildren = "hx-morph-skip-children"
	// PartialType marks a <template> as a partial for template languages that
	// strip unknown tags: <template hx-type="partial">. templ keeps unknown
	// tags, so prefer the Partial component.
	PartialType = "hx-type"
	// TypePartial is the PartialType value.
	TypePartial = "partial"
)

const (
	TargetThis = "this"
)

// Confirm shows a confirmation dialog before the request.
func Confirm(mods ...string) string {
	return Mod("hx-confirm", mods...)
}

// Target selects the element the response is swapped into.
func Target(mods ...string) string {
	return Mod("hx-target", mods...)
}

// Swap controls how the response is swapped in. See SwapValue for building
// a value with modifiers.
func Swap(mods ...string) string {
	return Mod("hx-swap", mods...)
}

// Include adds the values of other elements to the request.
func Include(mods ...string) string {
	return Mod("hx-include", mods...)
}

// Vals adds parameters to the request. Use Map for a JSON value or JSMap for
// a computed one - the js: prefix replaces the removed hx-vars.
func Vals(mods ...string) string {
	return Mod("hx-vals", mods...)
}

// Headers adds request headers.
func Headers(mods ...string) string {
	return Mod("hx-headers", mods...)
}

// Config sets per-element request config, as HCON ("timeout:5s") or JSON
// ({"timeout":5000}). Replaces the removed hx-request. The fetch mode key is
// deliberately not supported by htmx. See ConfigMeta for the global config.
func Config(mods ...string) string {
	return Mod("hx-config", mods...)
}

// Boost turns links and forms into htmx requests. The value is "true" or
// "false"; it is only useful with ModInherited.
func Boost(mods ...string) string {
	return Mod("hx-boost", mods...)
}

// Disable takes an extended selector of elements to set the disabled
// attribute on while the request is in flight.
//
// Careful: in htmx 2 hx-disable meant "skip htmx processing" - that is Ignore
// now, and hx-disable took over the role of hx-disabled-elt.
func Disable(mods ...string) string {
	return Mod("hx-disable", mods...)
}

// Indicator selects the element that gets ClassRequest during the request.
func Indicator(mods ...string) string {
	return Mod("hx-indicator", mods...)
}

// Sync controls request queueing. See SyncSpec for the value; the hx-trigger
// queue modifier was removed in htmx 4 in favor of this attribute.
func Sync(mods ...string) string {
	return Mod("hx-sync", mods...)
}

// On builds an inline event handler attribute, e.g. On("click") ->
// "hx-on:click". Events are also available: On(Click), On(EventAfterSwap).
//
// If a framework cannot handle ":" in attribute names, set
// htmx.config.metaCharacter and mirror it here.
func On(values ...string) string {
	return Mod("hx-on", values...)
}

// Status configures per-status-code response handling, e.g.
// Status("422") -> "hx-status:422". See status.go for the value keys.
func Status(code string, mods ...string) string {
	return Mod("hx-status:"+code, mods...)
}

// --------------------------------------------------------
// Removed in htmx 4

// Deprecated: renamed to hx-disable in htmx 4; use Disable.
const DisabledElt = "hx-disabled-elt"

// Deprecated: removed in htmx 4. Filter parameters in the
// EventConfigRequest handler instead.
const Params = "hx-params"

// Deprecated: removed in htmx 4. Use Map with Vals, or JSMap for computed
// values.
const Vars = "hx-vars"

// Deprecated: moved to the PromptExt extension in htmx 4.
const Prompt = "hx-prompt"

// Deprecated: removed in htmx 4. Extensions are loaded by including their
// script; ConfigExtensions restricts which ones may load.
const Ext = "hx-ext"

// Deprecated: removed in htmx 4. Inheritance is explicit - use ModInherited.
const Disinherit = "hx-disinherit"

// Deprecated: removed in htmx 4. Inheritance is explicit - use ModInherited.
const Inherit = "hx-inherit"

// Deprecated: renamed to hx-config in htmx 4; use Config.
const Request = "hx-request"

// Deprecated: removed in htmx 4 along with the localStorage history cache.
const History = "hx-history"
