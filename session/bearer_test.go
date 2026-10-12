package session_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/daarxwalker/enjin/infra/cache"
	"github.com/daarxwalker/enjin/internal/testredis"
	"github.com/daarxwalker/enjin/security"
	"github.com/daarxwalker/enjin/session"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestBearer(t *testing.T) {
	setup := func(t *testing.T, options ...session.StoreOption) (*session.Store, http.Handler) {
		t.Helper()
		store := session.NewStore(cache.NewClient(cache.NewRedisDriver(testredis.New(t))), options...)
		r := chi.NewRouter()
		r.Use(store.Middleware)
		r.With(session.Restrict(security.AnyAccount())).Get(
			"/account", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
		)
		return store, r
	}

	seed := func(t *testing.T, store *session.Store, id string) string {
		t.Helper()
		token, err := store.Seed(context.Background(), session.WithID(id))
		require.NoError(t, err)
		return token
	}

	serve := func(handler http.Handler, authorization string, cookie *http.Cookie) (int, []*http.Cookie) {
		r := httptest.NewRequest(http.MethodGet, "/account", nil)
		if authorization != "" {
			r.Header.Set("Authorization", authorization)
		}
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Code, w.Result().Cookies()
	}

	t.Run(
		"accept bearer token without cookie", func(t *testing.T) {
			store, handler := setup(t, session.WithBearer())
			status, cookies := serve(handler, "Bearer "+seed(t, store, "user"), nil)
			require.Equal(t, http.StatusOK, status)
			require.Empty(t, cookies)
		},
	)

	t.Run(
		"accept case insensitive scheme", func(t *testing.T) {
			store, handler := setup(t, session.WithBearer())
			status, _ := serve(handler, "bearer "+seed(t, store, "user"), nil)
			require.Equal(t, http.StatusOK, status)
		},
	)

	t.Run(
		"fall back to cookie", func(t *testing.T) {
			store, handler := setup(t, session.WithBearer())
			token := seed(t, store, "user")
			status, cookies := serve(handler, "", &http.Cookie{Name: store.Cookie(), Value: token})
			require.Equal(t, http.StatusOK, status)
			require.Len(t, cookies, 1)
			require.Equal(t, token, cookies[0].Value)
		},
	)

	t.Run(
		"fall back to cookie on invalid header", func(t *testing.T) {
			store, handler := setup(t, session.WithBearer())
			token := seed(t, store, "user")
			for _, authorization := range []string{"Basic " + token, "Bearer invalid", "Bearer"} {
				status, _ := serve(handler, authorization, &http.Cookie{Name: store.Cookie(), Value: token})
				require.Equal(t, http.StatusOK, status, authorization)
			}
		},
	)

	t.Run(
		"ignore header without bearer option", func(t *testing.T) {
			store, handler := setup(t)
			status, _ := serve(handler, "Bearer "+seed(t, store, "user"), nil)
			require.Equal(t, http.StatusUnauthorized, status)
		},
	)

	t.Run(
		"reject unknown token", func(t *testing.T) {
			_, handler := setup(t, session.WithBearer())
			store, _ := setup(t)
			status, _ := serve(handler, "Bearer "+seed(t, store, "user"), nil)
			require.Equal(t, http.StatusUnauthorized, status)
		},
	)
}
