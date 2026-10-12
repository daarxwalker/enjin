package redirect

type ctxKey struct{}

var key = ctxKey{}

type ctxVal struct {
	manager *Manager
}
