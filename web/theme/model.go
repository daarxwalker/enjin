package theme

type Value string

const (
	Light Value = "light"
	Dark  Value = "dark"
	// System leaves the choice to the browser's prefers-color-scheme.
	System Value = "system"
)

func (v Value) String() string {
	return string(v)
}
