package filestore

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
)

func detectContentType(body io.Reader) (string, io.Reader, error) {
	reader := bufio.NewReaderSize(body, sniffLen)
	head, err := reader.Peek(sniffLen)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", nil, fmt.Errorf("reader peek failed: %w", err)
	}
	return http.DetectContentType(head), reader, nil
}
