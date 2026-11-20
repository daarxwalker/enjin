package disk

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// contextReader stops a long upload as soon as its context ends.
type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

// Read passes io.EOF through unwrapped: io.Copy compares it with ==, so a
// wrapped EOF would turn every finished upload into a failure.
func (r contextReader) Read(p []byte) (int, error) {
	var zero int

	if err := r.ctx.Err(); err != nil {
		return zero, fmt.Errorf("read upload body stopped: %w", err)
	}
	read, err := r.reader.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		return read, fmt.Errorf("read upload body failed: %w", err)
	}
	return read, err
}
