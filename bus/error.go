package bus

import "errors"

var (
	ErrClosed = errors.New("bus closed")
	ErrFull   = errors.New("bus full")
	ErrInUse  = errors.New("bus in use")
)
