package httpx_test

import (
	"mime"
	"net/http/httptest"
	"testing"

	"github.com/daarxwalker/enjin/httpx"
)

func TestResponse(t *testing.T) {
	t.Run("stream", func(t *testing.T) {
		tests := []struct {
			name     string
			filename string
			want     string
		}{
			{"plain", "a.json", "attachment; filename=a.json"},
			{"space", "my file.json", `attachment; filename="my file.json"`},
			{"quote", `a"b.json`, `attachment; filename="a\"b.json"`},
			{"non ascii", "účet.json", "attachment; filename*=utf-8''%C3%BA%C4%8Det.json"},
			{"control", "a\r\nX: y.json", "attachment; filename*=utf-8''a%0D%0AX%3A%20y.json"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				if err := httpx.Stream(w, tt.filename, []byte("x")); err != nil {
					t.Fatalf("Stream() error = %v", err)
				}
				got := w.Header().Get("Content-Disposition")
				if got != tt.want {
					t.Fatalf("Content-Disposition = %q, want %q", got, tt.want)
				}
				_, params, err := mime.ParseMediaType(got)
				if err != nil {
					t.Fatalf("ParseMediaType(%q) error = %v", got, err)
				}
				if params["filename"] != tt.filename {
					t.Fatalf("filename = %q, want %q", params["filename"], tt.filename)
				}
			})
		}
	})
}
