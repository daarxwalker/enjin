package htmx

import (
	"github.com/a-h/templ"
)

// PartialProps configures one <hx-partial> block. Either ID or Target has to
// be set; ID is the shorthand for Target "#id".
type PartialProps struct {
	Attrs templ.Attributes
	// ID targets the element with that id, shorthand for Target.
	ID string
	// Target is an extended selector for the element to swap into.
	Target string
	// Swap is the swap style; htmx defaults to innerHTML.
	Swap string
}

// GetTarget resolves Target, falling back to ID.
func (props PartialProps) GetTarget() string {
	if props.Target != "" {
		return props.Target
	}
	if props.ID != "" {
		return "#" + props.ID
	}
	return ""
}
