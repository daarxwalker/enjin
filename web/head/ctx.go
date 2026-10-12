package head

type ContextValue struct {
	Title   string
	Metas   []Meta
	Links   []Link
	Scripts []Script
}

type ctxKey struct{}

var key = ctxKey{}
