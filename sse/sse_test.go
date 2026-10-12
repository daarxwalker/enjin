package sse_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/sse"

	"github.com/stretchr/testify/require"
)

func TestSSE(t *testing.T) {
	t.Run(
		"begin", func(t *testing.T) {
			w := httptest.NewRecorder()
			_, err := sse.Begin(w)
			require.NoError(t, err)
			require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
			require.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
			require.Equal(t, "no", w.Header().Get("X-Accel-Buffering"))
			require.Equal(t, ": ping\n\n", w.Body.String())
			require.True(t, w.Flushed)
		},
	)

	t.Run(
		"begin outlives write timeout", func(t *testing.T) {
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				flusher, err := sse.Begin(w)
				if err != nil {
					t.Error(err)
					return
				}
				time.Sleep(300 * time.Millisecond)
				if err := sse.Data(w, flusher, "late"); err != nil {
					t.Error(err)
				}
			}))
			server.Config.WriteTimeout = 100 * time.Millisecond
			server.Start()
			defer server.Close()
			res, err := http.Get(server.URL)
			require.NoError(t, err)
			defer func() { require.NoError(t, res.Body.Close()) }()
			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			require.Contains(t, string(body), "data: late")
		},
	)

	t.Run(
		"begin without flusher", func(t *testing.T) {
			_, err := sse.Begin(struct{ http.ResponseWriter }{httptest.NewRecorder()})
			require.Error(t, err)
		},
	)

	t.Run(
		"data and event", func(t *testing.T) {
			tests := []struct {
				name      string
				eventName string
				data      string
				want      []string
			}{
				{name: "single line", data: "hello", want: []string{"data: hello"}},
				{name: "multi line", data: "a\nb", want: []string{"data: a", "data: b"}},
				{name: "carriage return", data: "a\revent: evil\r\nb", want: []string{"data: a", "data: event: evil", "data: b"}},
				{name: "named event", eventName: "update", data: "x", want: []string{"event: update", "data: x"}},
			}
			for _, tt := range tests {
				t.Run(
					tt.name, func(t *testing.T) {
						w := httptest.NewRecorder()
						require.NoError(t, sse.Event(w, w, tt.eventName, tt.data))
						lines := strings.Split(strings.TrimSuffix(w.Body.String(), "\n\n"), "\n")
						var fields []string
						for _, line := range lines {
							if strings.HasPrefix(line, "id: ") {
								continue
							}
							fields = append(fields, line)
						}
						require.Equal(t, tt.want, fields)
						require.True(t, strings.HasSuffix(w.Body.String(), "\n\n"))
					},
				)
			}
		},
	)

	t.Run(
		"reject event name with line break", func(t *testing.T) {
			w := httptest.NewRecorder()
			require.Error(t, sse.Event(w, w, "update\ndata: evil", "x"))
			require.Empty(t, w.Body.String())
		},
	)
}
