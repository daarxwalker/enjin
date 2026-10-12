package validate

type ctxKey struct{}

var key = ctxKey{}

type ctxVal struct {
	manager *Manager
	result  Result
}
