package mail

import "context"

type Driver interface {
	Send(ctx context.Context, envelope Envelope, raw []byte) error
}
