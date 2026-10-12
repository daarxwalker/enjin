package client

type ctxKeyType struct{}

var ctxKey = ctxKeyType{}

type ctxVal struct {
	values    map[string]any
	unixMilli map[string]int64
	id        string
	changed   bool
}
