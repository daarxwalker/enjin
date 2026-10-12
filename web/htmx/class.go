package htmx

// CSS classes htmx applies during a request and swap. All five still exist in
// htmx 4, even though the config keys that renamed them were removed.
const (
	ClassAdded     = "htmx-added"
	ClassIndicator = "htmx-indicator"
	ClassRequest   = "htmx-request"
	ClassSettling  = "htmx-settling"
	ClassSwapping  = "htmx-swapping"
)
