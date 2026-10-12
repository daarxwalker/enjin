package htmx

// Trigger specs and request synchronization.
//
//	htmx.Trigger: htmx.Value(htmx.KeyUp,
//	    htmx.Modifier(htmx.TriggerDelay, "500ms"),
//	    htmx.From("closest form"),
//	)
//
// htmx 4 parses modifier values with HCON, so selectors containing spaces or
// commas must be single quoted. From and TriggerFor do that; elsewhere use
// QuoteSelector.

// Trigger modifier keys.
const (
	TriggerOnce      = "once"
	TriggerChanged   = "changed"
	TriggerDelay     = "delay"
	TriggerThrottle  = "throttle"
	TriggerFrom      = "from"
	TriggerTarget    = "target"
	TriggerConsume   = "consume"
	TriggerEvery     = "every"
	TriggerRoot      = "root"
	TriggerThreshold = "threshold"
)

// From builds a "from:" modifier, quoting the selector when needed.
//
//	From("closest form") // "from:'closest form'"
func From(selector string) string {
	return Modifier(TriggerFrom, QuoteSelector(selector))
}

// TriggerFor builds a "target:" modifier for event delegation, quoting the
// selector when needed.
func TriggerFor(selector string) string {
	return Modifier(TriggerTarget, QuoteSelector(selector))
}

// hx-sync strategies. htmx 4 removed the hx-trigger queue modifier, so all
// queueing goes through the Sync attribute.
const (
	SyncDrop       = "drop"
	SyncAbort      = "abort"
	SyncReplace    = "replace"
	SyncQueue      = "queue"
	SyncQueueAll   = "queue all"
	SyncQueueFirst = "queue first"
	SyncQueueLast  = "queue last"
)

// SyncSpec builds a Sync value, quoting the selector when needed.
//
//	SyncSpec(TargetThis, SyncQueueAll) // "this:queue all"
func SyncSpec(selector, strategy string) string {
	return QuoteSelector(selector) + ":" + strategy
}

// Deprecated: the hx-trigger queue modifier was removed in htmx 4. Use the
// Sync attribute with SyncSpec.
const TriggerQueue = "queue"
