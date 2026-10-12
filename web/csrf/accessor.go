package csrf

import "context"

func RejectedFrom(ctx context.Context) bool {
	cv := from(ctx)
	if cv == nil {
		return false
	}
	return cv.rejected
}

func from(ctx context.Context) *ctxVal {
	v := ctx.Value(ctxKey)
	if v == nil {
		return nil
	}
	cv, ok := v.(*ctxVal)
	if !ok {
		return nil
	}
	return cv
}
