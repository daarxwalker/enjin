package antibot

import "context"

func RejectedFrom(ctx context.Context) bool {
	v := ctx.Value(rejectedKey)
	if v == nil {
		return false
	}
	rejected, ok := v.(bool)
	if !ok {
		return false
	}
	return rejected
}
