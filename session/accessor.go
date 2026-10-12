package session

import "context"

func Exists(ctx context.Context) bool {
	s := From(ctx)
	if s == nil {
		return false
	}
	return s.ID != ""
}

func From(ctx context.Context) *Profile {
	ctxv := from(ctx)
	if ctxv == nil {
		return nil
	}
	return ctxv.profile
}

func from(ctx context.Context) *ctxVal {
	val := ctx.Value(ctxKey)
	if val == nil {
		return nil
	}
	ctxv, ok := val.(*ctxVal)
	if !ok {
		return nil
	}
	return ctxv
}
