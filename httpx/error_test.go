package httpx_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/daarxwalker/enjin/httpx"
)

func TestErr(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		tests := []struct {
			name string
			err  *httpx.Err
			want string
		}{
			{"raw", &httpx.Err{Status: http.StatusBadRequest, Raw: errors.New("bad input")}, "bad input"},
			{"status only", &httpx.Err{Status: http.StatusNotFound}, "Not Found"},
			{"unknown status", &httpx.Err{Status: 799}, "http status 799"},
			{"nil", nil, ""},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.err.Error(); got != tt.want {
					t.Fatalf("Error() = %q, want %q", got, tt.want)
				}
			})
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		if err := httpx.Error(http.StatusInternalServerError, context.Canceled); !errors.Is(err, context.Canceled) {
			t.Fatalf("errors.Is(%v, context.Canceled) = false, want true", err)
		}
		if got := errors.Unwrap(httpx.StatusError(http.StatusNotFound)); got != nil {
			t.Fatalf("Unwrap() = %v, want nil", got)
		}
	})
}
