package consent

type ctxKey struct{}

var key = ctxKey{}

type ContextValue struct {
	*Cookies
	manager *Manager
	Exists  bool
}
