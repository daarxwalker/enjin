package theme

type ctxKey struct{}

var (
	key = ctxKey{}
)

type ContextValue struct {
	manager *Manager
	value   Value
}
