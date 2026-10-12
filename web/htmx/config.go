package htmx

import (
	"strings"

	"github.com/a-h/templ"
)

// ConfigMeta renders the global htmx config as a <meta> tag for the document
// head. Use the Config* key constants:
//
//	htmx.ConfigMeta(map[string]any{
//	    htmx.ConfigDefaultSwap:   htmx.SwapReplaceContent,
//	    htmx.ConfigTransitions:   true,
//	})
//
// The JSON is single quoted; json.Marshal escapes <, > and &, single quotes
// are escaped here.
func ConfigMeta(cfg map[string]any) templ.Component {
	return templ.Raw(`<meta name="htmx-config" content='` + strings.ReplaceAll(Map(cfg), "'", "&#39;") + `'/>`)
}

// Config keys for ConfigMeta and htmx.config in JavaScript.
const (
	// ConfigDefaultSwap is the swap style used when Swap is absent.
	// Renamed from defaultSwapStyle.
	ConfigDefaultSwap = "defaultSwap"
	// ConfigDefaultTimeout is the request timeout in ms. htmx 4 defaults to
	// 60000; htmx 2 had no timeout. Set 0 to restore that.
	ConfigDefaultTimeout = "defaultTimeout"
	// ConfigDefaultSettleDelay defaults to 1 in htmx 4, down from 20.
	ConfigDefaultSettleDelay = "defaultSettleDelay"
	ConfigDefaultFocusScroll = "defaultFocusScroll"
	// ConfigHistory takes true, false or "reload". Renamed from
	// historyEnabled; htmx 4 re-fetches instead of caching in localStorage.
	ConfigHistory = "history"
	// ConfigImplicitInheritance set to true restores htmx 2 inheritance, so
	// ModInherited is not needed.
	ConfigImplicitInheritance = "implicitInheritance"
	// ConfigNoSwap lists status codes that are not swapped. htmx 4 defaults
	// to [204, 304] - set ["204", "304", "4xx", "5xx"] for htmx 2 behavior.
	// A Status attribute for a more specific code still wins: with "4xx" here,
	// Status("422") keeps working.
	ConfigNoSwap = "noSwap"
	// ConfigTransitions enables the View Transitions API. Renamed from
	// globalViewTransitions; off by default.
	ConfigTransitions = "transitions"
	// ConfigMode is the fetch() mode, "same-origin" by default. Replaces
	// selfRequestsOnly.
	ConfigMode = "mode"
	// ConfigExtensions is a comma-separated allowlist of registered extension
	// names, matched exactly - use the *Ext constants, not script file names.
	ConfigExtensions = "extensions"
	// ConfigMetaCharacter replaces ":" in attribute and event names, for
	// frameworks such as JSX that cannot handle it.
	ConfigMetaCharacter = "metaCharacter"
	// ConfigPrefix is a secondary attribute prefix, e.g. "data-hx-".
	ConfigPrefix = "prefix"

	// Morph, for SwapMorph / SwapMorphOuter / SwapSync.
	ConfigMorphIgnore       = "morphIgnore"
	ConfigMorphSkip         = "morphSkip"
	ConfigMorphSkipChildren = "morphSkipChildren"
	ConfigMorphScanLimit    = "morphScanLimit"

	// ConfigIncludeIndicatorCSS controls the built-in indicator stylesheet.
	// Renamed from includeIndicatorStyles; htmx 4 uses constructable
	// stylesheets, so no nonce is needed.
	ConfigIncludeIndicatorCSS = "includeIndicatorCSS"
	ConfigIndicatorClass      = "indicatorClass"
	ConfigRequestClass        = "requestClass"

	ConfigInlineScriptNonce = "inlineScriptNonce"
	// ConfigLogAll routes every internal event to console.log.
	ConfigLogAll = "logAll"
	// ConfigAllowEmptySwapAfterOOB controls whether the main swap runs when
	// the response only carried out-of-band content. See also ModSwapEmpty.
	ConfigAllowEmptySwapAfterOOB = "allowEmptySwapAfterOOB"
	// ConfigVersion is read-only.
	ConfigVersion = "version"
)

// --------------------------------------------------------
// Renamed or removed in htmx 4

// Deprecated: renamed to ConfigDefaultSwap.
const ConfigDefaultSwapStyle = "defaultSwapStyle"

// Deprecated: renamed to ConfigTransitions.
const ConfigGlobalViewTransitions = "globalViewTransitions"

// Deprecated: renamed to ConfigHistory.
const ConfigHistoryEnabled = "historyEnabled"

// Deprecated: renamed to ConfigIncludeIndicatorCSS.
const ConfigIncludeIndicatorStyles = "includeIndicatorStyles"

// Deprecated: renamed to ConfigDefaultTimeout.
const ConfigTimeout = "timeout"

// Deprecated: replaced by ConfigMode.
const ConfigSelfRequestsOnly = "selfRequestsOnly"

// Deprecated: replaced by the Status attribute together with ConfigNoSwap.
const ConfigResponseHandling = "responseHandling"

// Deprecated: replaced by Ignore.
const ConfigDisableSelector = "disableSelector"

// Deprecated: removed in htmx 4. Kept so existing config maps still compile;
// setting them has no effect.
const (
	ConfigAddedClass             = "addedClass"
	ConfigSettlingClass          = "settlingClass"
	ConfigSwappingClass          = "swappingClass"
	ConfigAllowEval              = "allowEval"
	ConfigAllowNestedOobSwaps    = "allowNestedOobSwaps"
	ConfigAllowScriptTags        = "allowScriptTags"
	ConfigAttributesToSettle     = "attributesToSettle"
	ConfigDefaultSwapDelay       = "defaultSwapDelay"
	ConfigGetCacheBusterParam    = "getCacheBusterParam"
	ConfigHistoryCacheSize       = "historyCacheSize"
	ConfigIgnoreTitle            = "ignoreTitle"
	ConfigInlineStyleNonce       = "inlineStyleNonce"
	ConfigMethodsThatUseURLParam = "methodsThatUseUrlParams"
	ConfigRefreshOnHistoryMiss   = "refreshOnHistoryMiss"
	ConfigScrollBehavior         = "scrollBehavior"
	ConfigScrollIntoViewOnBoost  = "scrollIntoViewOnBoost"
	ConfigTriggerSpecsCache      = "triggerSpecsCache"
	ConfigUseTemplateFragments   = "useTemplateFragments"
	ConfigWithCredentials        = "withCredentials"
	ConfigWsBinaryType           = "wsBinaryType"
	ConfigWsReconnectDelay       = "wsReconnectDelay"
)

// Extensions builds the comma-separated allowlist for ConfigExtensions.
//
//	htmx.ConfigExtensions: htmx.Extensions(htmx.SSEExt, htmx.PendingExt)
func Extensions(names ...string) string {
	return strings.Join(names, ", ")
}
