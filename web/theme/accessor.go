package theme

import (
	"context"
)

func IsDark(ctx context.Context) bool {
	v := From(ctx)
	return v == Dark
}

func From(ctx context.Context) Value {
	ctxVal := from(ctx)
	if ctxVal == nil {
		return Light
	}
	return ctxVal.value
}

func from(ctx context.Context) *ContextValue {
	v := ctx.Value(key)
	if v == nil {
		return nil
	}
	ctxVal, ok := v.(*ContextValue)
	if !ok {
		return nil
	}
	return ctxVal
}
