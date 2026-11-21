package theme

import (
	"context"
)

// IsDark is false for System too: only the browser knows which scheme it picks.
func IsDark(ctx context.Context) bool {
	v := From(ctx)
	return v == Dark
}

func IsSystem(ctx context.Context) bool {
	v := From(ctx)
	return v == System
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
