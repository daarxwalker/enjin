package client

import (
	"context"
	"time"
)

func AttachValue[T any](ctx context.Context, key string, value T, ttl ...time.Duration) {
	cv := from(ctx)
	if cv == nil {
		return
	}
	cv.values[key] = value
	if len(ttl) > 0 {
		cv.unixMilli[key] = time.Now().Add(ttl[0]).UnixMilli()
	} else {
		delete(cv.unixMilli, key)
	}
	cv.changed = true
}

func DestroyValue(ctx context.Context, key string) {
	cv := from(ctx)
	if cv == nil {
		return
	}
	if _, ok := cv.values[key]; !ok {
		return
	}
	delete(cv.values, key)
	delete(cv.unixMilli, key)
	cv.changed = true
}
