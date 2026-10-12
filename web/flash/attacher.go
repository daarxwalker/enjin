package flash

import (
	"context"
)

func Attach(ctx context.Context, messages ...Message) {
	ctxVal := from(ctx)
	if ctxVal == nil {
		return
	}
	updatedMessages := ctxVal.Messages
	for _, msg := range messages {
		updatedMessages = append(updatedMessages, msg)
	}
	ctxVal.Messages = updatedMessages
}
