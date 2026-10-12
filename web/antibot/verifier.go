package antibot

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/daarxwalker/enjin/session"

	"github.com/a-h/templ"
)

type Verifier struct {
	driver Driver
	reject http.Handler
	log    *slog.Logger
}

var _ Driver = (*TurnstileDriver)(nil)

func NewVerifier(driver Driver, options ...Option) *Verifier {
	v := &Verifier{
		driver: driver,
		reject: forbidden,
		log:    slog.Default(),
	}
	for _, option := range options {
		option(v)
	}
	return v
}

func (m *Verifier) CreateWidget(ctx context.Context) templ.Component {
	return m.driver.CreateWidget(ctx)
}

func (m *Verifier) Verify(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if session.Exists(r.Context()) {
				next.ServeHTTP(w, r)
				return
			}
			if !m.verify(r) {
				ctx := context.WithValue(r.Context(), rejectedKey, true)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		},
	)
}

func (m *Verifier) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if session.Exists(r.Context()) {
				next.ServeHTTP(w, r)
				return
			}
			if !m.verify(r) {
				m.reject.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		},
	)
}

func (m *Verifier) verify(r *http.Request) bool {
	verified, verifyErr := m.driver.Verify(r)
	if verifyErr != nil {
		m.log.ErrorContext(r.Context(), fmt.Sprintf("antibot verification failed: %s", verifyErr))
	}
	return verifyErr == nil && verified
}
