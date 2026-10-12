package head

import (
	"context"
)

func Attach(ctx context.Context, options ...Option) {
	ctxVal := from(ctx)
	if ctxVal == nil {
		return
	}
	for _, option := range options {
		option(ctxVal)
	}
}
