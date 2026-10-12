package httpx

import (
	"context"
	"net/http"
)

func PathPrefix(r *http.Request) string {
	return forwardedHeader(r, "X-Forwarded-Prefix")
}

func IsFromCloudflare(r *http.Request) bool {
	val := r.Context().Value(cloudflareKey{})
	if val == nil {
		return false
	}
	via, ok := val.(bool)
	if !ok {
		return false
	}
	return via
}

func From(ctx context.Context) *ContextValue {
	ctxv := from(ctx)
	if ctxv == nil {
		return nil
	}
	return ctxv
}

func from(ctx context.Context) *ContextValue {
	val := ctx.Value(key)
	if val == nil {
		return nil
	}
	ctxv, ok := val.(*ContextValue)
	if !ok {
		return nil
	}
	return ctxv
}
