package validate

import (
	"context"
)

func From(ctx context.Context) Result {
	cv := from(ctx)
	if cv == nil {
		return nil
	}
	result := cv.result
	cv.manager.destroy(ctx)
	return result
}

func from(ctx context.Context) *ctxVal {
	val := ctx.Value(key)
	if val == nil {
		return nil
	}
	cv, ok := val.(*ctxVal)
	if !ok {
		return nil
	}
	return cv
}
