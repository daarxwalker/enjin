package sse

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/daarxwalker/enjin/token"
)

var lineBreaks = strings.NewReplacer("\r\n", "\n", "\r", "\n")

func Begin(w http.ResponseWriter) (http.Flusher, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("response writer does not support flushing")
	}
	if err := http.NewResponseController(w).SetWriteDeadline(time.Time{}); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return nil, fmt.Errorf("clear sse write deadline failed: %w", err)
	}

	h := w.Header()
	h.Set("X-Accel-Buffering", "no")
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")

	if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
		return nil, fmt.Errorf("write sse init failed: %w", err)
	}

	flusher.Flush()
	return flusher, nil
}

func Data(w http.ResponseWriter, flusher http.Flusher, data string) error {
	if _, err := fmt.Fprintf(w, "id: %s\n", token.New()); err != nil {
		return fmt.Errorf("write sse id failed: %w", err)
	}
	for line := range strings.SplitSeq(lineBreaks.Replace(data), "\n") {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return fmt.Errorf("write sse data failed: %w", err)
		}
	}
	if _, err := fmt.Fprintf(w, "\n"); err != nil {
		return fmt.Errorf("write sse newline failed: %w", err)
	}
	flusher.Flush()
	return nil
}

func Event(w http.ResponseWriter, flusher http.Flusher, name, data string) error {
	if strings.ContainsAny(name, "\r\n") {
		return fmt.Errorf("sse: invalid event name %q", name)
	}
	if name != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", name); err != nil {
			return err
		}
	}
	return Data(w, flusher, data)
}
