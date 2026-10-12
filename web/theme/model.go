package theme

type Value string

const (
	Light Value = "light"
	Dark  Value = "dark"
)

func (v Value) String() string {
	return string(v)
}
