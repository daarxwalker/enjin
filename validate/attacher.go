package validate

import "context"

func Attach(ctx context.Context, result Result) {
	cv := from(ctx)
	if cv == nil {
		return
	}
	cv.manager.save(ctx, result)
}
