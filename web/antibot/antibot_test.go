package antibot

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/a-h/templ"
	"github.com/stretchr/testify/require"
)

func TestVerifier(t *testing.T) {
	t.Run(
		"verify flags rejection", func(t *testing.T) {
			tests := []struct {
				name     string
				driver   fakeDriver
				rejected bool
			}{
				{name: "verified", driver: fakeDriver{verified: true}, rejected: false},
				{name: "not verified", driver: fakeDriver{}, rejected: true},
				{name: "driver error", driver: fakeDriver{verified: true, err: errors.New("failed")}, rejected: true},
			}
			for _, tt := range tests {
				t.Run(
					tt.name, func(t *testing.T) {
						var rejected bool
						handler := NewVerifier(tt.driver).Verify(
							http.HandlerFunc(
								func(_ http.ResponseWriter, r *http.Request) {
									rejected = RejectedFrom(r.Context())
								},
							),
						)
						res := serve(handler)
						require.Equal(t, http.StatusOK, res.Code)
						require.Equal(t, tt.rejected, rejected)
					},
				)
			}
		},
	)

	t.Run(
		"require rejects", func(t *testing.T) {
			tests := []struct {
				name   string
				driver fakeDriver
				status int
			}{
				{name: "verified", driver: fakeDriver{verified: true}, status: http.StatusOK},
				{name: "not verified", driver: fakeDriver{}, status: http.StatusForbidden},
				{name: "driver error", driver: fakeDriver{verified: true, err: errors.New("failed")}, status: http.StatusForbidden},
			}
			for _, tt := range tests {
				t.Run(
					tt.name, func(t *testing.T) {
						handler := NewVerifier(tt.driver).Require(
							http.HandlerFunc(
								func(w http.ResponseWriter, _ *http.Request) {
									w.WriteHeader(http.StatusOK)
								},
							),
						)
						require.Equal(t, tt.status, serve(handler).Code)
					},
				)
			}
		},
	)

	t.Run(
		"require with reject handler", func(t *testing.T) {
			reject := http.HandlerFunc(
				func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusSeeOther)
				},
			)
			ok := http.HandlerFunc(
				func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
			)
			require.Equal(t, http.StatusSeeOther, serve(NewVerifier(fakeDriver{}, WithRejectHandler(reject)).Require(ok)).Code)
			require.Equal(t, http.StatusOK, serve(NewVerifier(fakeDriver{verified: true}, WithRejectHandler(reject)).Require(ok)).Code)
		},
	)
}

type fakeDriver struct {
	err      error
	verified bool
}

func (d fakeDriver) CreateWidget(_ context.Context) templ.Component {
	return templ.NopComponent
}

func (d fakeDriver) Verify(_ *http.Request) (bool, error) {
	return d.verified, d.err
}

func serve(handler http.Handler) *httptest.ResponseRecorder {
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/", nil))
	return res
}
