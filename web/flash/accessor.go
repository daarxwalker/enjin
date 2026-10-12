package flash

import (
	"context"
)

func From(ctx context.Context) Messages {
	ctxVal := from(ctx)
	if ctxVal == nil {
		return nil
	}
	ctxVal.read = true
	return ctxVal.Messages
}

func from(ctx context.Context) *ContextValue {
	val := ctx.Value(key)
	if val == nil {
		return nil
	}
	ctxVal, ok := val.(*ContextValue)
	if !ok {
		return nil
	}
	return ctxVal
}
