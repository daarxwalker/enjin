package consent

import (
	"context"
)

func From(ctx context.Context) *ContextValue {
	ctxVal := from(ctx)
	if ctxVal == nil {
		return nil
	}
	return ctxVal
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
