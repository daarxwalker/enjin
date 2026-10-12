package app

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/daarxwalker/enjin/httpx"
)

func TestDefaultHandler(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(w http.ResponseWriter, r *http.Request) error
		status   int
		body     string
		hidden   string
		logged   string
		logLines int
		stack    bool
	}{
		{
			name:     "internal",
			fn:       func(_ http.ResponseWriter, _ *http.Request) error { return errors.New("secret db detail") },
			status:   http.StatusInternalServerError,
			body:     http.StatusText(http.StatusInternalServerError),
			hidden:   "secret",
			logged:   "secret db detail",
			logLines: 1,
		},
		{
			name: "public client",
			fn: func(_ http.ResponseWriter, _ *http.Request) error {
				return httpx.Error(http.StatusBadRequest, errors.New("bad input"))
			},
			status: http.StatusBadRequest,
			body:   "bad input",
		},
		{
			name:   "status only",
			fn:     func(_ http.ResponseWriter, _ *http.Request) error { return httpx.StatusError(http.StatusNotFound) },
			status: http.StatusNotFound,
			body:   http.StatusText(http.StatusNotFound),
		},
		{
			name: "server error with raw",
			fn: func(_ http.ResponseWriter, _ *http.Request) error {
				return httpx.Error(http.StatusInternalServerError, errors.New("wrapped detail"))
			},
			status:   http.StatusInternalServerError,
			body:     http.StatusText(http.StatusInternalServerError),
			hidden:   "wrapped",
			logged:   "wrapped detail",
			logLines: 1,
		},
		{
			name:     "panic",
			fn:       func(_ http.ResponseWriter, _ *http.Request) error { panic("boom") },
			status:   http.StatusInternalServerError,
			body:     http.StatusText(http.StatusInternalServerError),
			hidden:   "boom",
			logged:   "boom",
			logLines: 1,
			stack:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			h := NewDefaultHandler(WithHandlerLogger(slog.New(slog.NewTextHandler(&buf, nil))))
			w := httptest.NewRecorder()
			h.Handle(tt.fn)(w, httptest.NewRequest(http.MethodGet, "/x?token=abc", nil))

			if w.Code != tt.status {
				t.Fatalf("status = %d, want %d", w.Code, tt.status)
			}
			if !strings.Contains(w.Body.String(), tt.body) {
				t.Fatalf("body = %q, want %q", w.Body.String(), tt.body)
			}
			if tt.hidden != "" && strings.Contains(w.Body.String(), tt.hidden) {
				t.Fatalf("body = %q leaks %q", w.Body.String(), tt.hidden)
			}
			log := buf.String()
			if lines := strings.Count(log, "\n"); lines != tt.logLines {
				t.Fatalf("log lines = %d, want %d: %s", lines, tt.logLines, log)
			}
			if !strings.Contains(log, tt.logged) {
				t.Fatalf("log = %q, want %q", log, tt.logged)
			}
			if strings.Contains(log, "token") {
				t.Fatalf("log = %q leaks query", log)
			}
			if tt.stack != strings.Contains(log, "stack=") {
				t.Fatalf("log = %q, stack = %v", log, tt.stack)
			}
		})
	}

	t.Run("on error hides internal", func(t *testing.T) {
		var got *httpx.Err
		h := NewDefaultHandler(
			WithHandlerLogger(slog.New(slog.DiscardHandler)),
			WithHandlerOnError(func(_ http.ResponseWriter, _ *http.Request, err *httpx.Err) {
				got = err
			}),
		)
		h.Handle(func(_ http.ResponseWriter, _ *http.Request) error {
			return errors.New("secret")
		})(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

		if got == nil || got.Status != http.StatusInternalServerError || got.Raw != nil {
			t.Fatalf("onError got %+v, want status 500 without raw", got)
		}
	})
}
