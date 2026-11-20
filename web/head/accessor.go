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
	val := ctx.Value(key)
	if val == nil {
		return nil
	}
	c, ok := val.(*ContextValue)
	if !ok {
		return nil
	}
	return c
}
