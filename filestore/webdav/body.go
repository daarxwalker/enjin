package webdav

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
)

// uploadBody is what net/http gets instead of the caller's body. Its Close
// only reports that the transport is done reading: net/http closes the body
// it gets (which would close a caller's *os.File) and may still read it after
// Do returns, so a retry has to wait for Close before it seeks back.
type uploadBody struct {
	reader io.Reader
	closed chan struct{}
	once   sync.Once
}

func newUploadBody(reader io.Reader) *uploadBody {
	return &uploadBody{
		reader: reader,
		closed: make(chan struct{}),
	}
}

// Read passes io.EOF through unwrapped: net/http compares it with ==.
func (b *uploadBody) Read(p []byte) (int, error) {
	read, err := b.reader.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		return read, fmt.Errorf("read upload body failed: %w", err)
	}
	return read, err
}

func (b *uploadBody) Close() error {
	b.once.Do(func() {
		close(b.closed)
	})
	return nil
}

func (b *uploadBody) waitForClose(ctx context.Context) error {
	select {
	case <-b.closed:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for the upload body to be released stopped: %w", ctx.Err())
	}
}
