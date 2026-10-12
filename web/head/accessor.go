package head

import (
	"context"
)

func From(ctx context.Context) *ContextValue {
	if c := from(ctx); c != nil {
		return c
	}
	return new(ContextValue)
}

func from(ctx context.Context) *ContextValue {
	c, _ := ctx.Value(key).(*ContextValue)
	return c
}
