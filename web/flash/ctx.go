package flash

type ctxKey struct{}

var (
	key = ctxKey{}
)

type ContextValue struct {
	Messages
	read bool
}
