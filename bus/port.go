package bus

import "context"

type Driver interface {
	Publish(ctx context.Context, subject string, data []byte) error
	TryPublish(subject string, data []byte) error
	Subscribe(subject, queue string, handler func(ctx context.Context, data []byte) error) (Subscription, error)
}

type Subscription interface {
	Close(ctx context.Context) error
}

type adapter[T any] interface {
	Publish(ctx context.Context, message T) error
	TryPublish(message T) error
	Subscribe(bufSize int) (<-chan T, error)
	Unsubscribe(ch <-chan T)
	Close(ctx context.Context) error
}

var (
	_ adapter[any] = (*memory[any])(nil)
	_ adapter[any] = (*remote[any])(nil)
)
