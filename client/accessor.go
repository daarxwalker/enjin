package client

import (
	"context"
	"encoding/json"
)

func IDFrom(ctx context.Context) string {
	cv := from(ctx)
	if cv == nil {
		return ""
	}
	return cv.id
}

func ValueFrom[T any](ctx context.Context, key string) T {
	var zero T
	cv := from(ctx)
	if cv == nil {
		return zero
	}
	value, ok := cv.values[key]
	if !ok {
		return zero
	}
	if typedValue, ok := value.(T); ok {
		return typedValue
	}
	raw, ok := value.(json.RawMessage)
	if !ok {
		return zero
	}
	var decoded T
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return zero
	}
	return decoded
}

func from(ctx context.Context) *ctxVal {
	val := ctx.Value(ctxKey)
	if val == nil {
		return nil
	}
	cv, ok := val.(*ctxVal)
	if !ok {
		return nil
	}
	return cv
}
