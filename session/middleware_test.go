package session_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"slices"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/app"
	"github.com/daarxwalker/enjin/httpx"
	"github.com/daarxwalker/enjin/infra/cache"
	"github.com/daarxwalker/enjin/internal/testredis"
	"github.com/daarxwalker/enjin/security"
	"github.com/daarxwalker/enjin/session"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestRestrict(t *testing.T) {
	admin := &security.Role{Name: "admin"}
	store := session.NewStore(
		cache.NewClient(cache.NewRedisDriver(testredis.New(t))),
		session.WithRoles(map[string]*security.Role{admin.Name: admin}),
	)
	srv := newTestServer(t, store, admin)

	t.Run(
		"reject missing session", func(t *testing.T) {
			require.Equal(t, http.StatusUnauthorized, srv.get("/admin", ""))
		},
	)

	t.Run(
		"reject session without role", func(t *testing.T) {
			token := srv.seed(session.WithID("user"))
			require.Equal(t, http.StatusForbidden, srv.get("/admin", token))
		},
	)

	t.Run(
		"accept session with required role", func(t *testing.T) {
			token := srv.seed(session.WithID("admin"), session.WithRole(admin))
			require.Equal(t, http.StatusOK, srv.get("/admin", token))
		},
	)

	t.Run(
		"accept any account without role", func(t *testing.T) {
			token := srv.seed(session.WithID("user"))
			require.Equal(t, http.StatusOK, srv.get("/account", token))
		},
	)
}

func TestRedirectBack(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		target   string
		prefix   string
		options  []session.MiddlewareOption
		location string
	}{
		{
			name:     "adds the rejected path",
			method:   http.MethodGet,
			target:   "/account",
			options:  []session.MiddlewareOption{session.WithRedirectName("login"), session.WithRedirectBack()},
			location: "/login?redirect=%2Faccount",
		},
		{
			name:     "keeps the rejected query",
			method:   http.MethodGet,
			target:   "/account?tab=orders&page=2",
			options:  []session.MiddlewareOption{session.WithRedirectBack(), session.WithRedirectName("login")},
			location: "/login?redirect=%2Faccount%3Ftab%3Dorders%26page%3D2",
		},
		{
			name:     "keeps the query of the target",
			method:   http.MethodGet,
			target:   "/account",
			options:  []session.MiddlewareOption{session.WithRedirectName("login", map[string]any{"step": "email"}), session.WithRedirectBack()},
			location: "/login?redirect=%2Faccount&step=email",
		},
		{
			name:     "includes the path prefix",
			method:   http.MethodGet,
			target:   "/account",
			prefix:   "/shop",
			options:  []session.MiddlewareOption{session.WithRedirectName("login"), session.WithRedirectBack()},
			location: "/shop/login?redirect=%2Fshop%2Faccount",
		},
		{
			name:     "skips unsafe methods",
			method:   http.MethodPost,
			target:   "/account",
			options:  []session.MiddlewareOption{session.WithRedirectName("login"), session.WithRedirectBack()},
			location: "/login",
		},
		{
			name:     "off by default",
			method:   http.MethodGet,
			target:   "/account",
			options:  []session.MiddlewareOption{session.WithRedirectName("login")},
			location: "/login",
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				mux := chi.NewRouter()
				mux.Use(httpx.NewProxy(httpx.WithTrustedProxies(netip.MustParsePrefix("192.0.2.0/24"))).Middleware, httpx.Middleware)
				r := app.NewRouter(mux, app.NewDefaultHandler())
				r.Get("login", "/login", func(w http.ResponseWriter, _ *http.Request) error {
					w.WriteHeader(http.StatusOK)
					return nil
				})
				restricted := r.With(session.Restrict(security.AnyAccount(), tt.options...))
				handler := func(w http.ResponseWriter, _ *http.Request) error {
					w.WriteHeader(http.StatusOK)
					return nil
				}
				restricted.Get("account", "/account", handler)
				restricted.Post("accountUpdate", "/account", handler)
				require.NoError(t, r.Err())

				req := httptest.NewRequestWithContext(t.Context(), tt.method, tt.target, nil)
				if tt.prefix != "" {
					req.Header.Set("X-Forwarded-Prefix", tt.prefix)
				}
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)
				require.Equal(t, http.StatusSeeOther, w.Code)
				require.Equal(t, tt.location, w.Header().Get("Location"))
			},
		)
	}

	t.Run(
		"without target responds with an error", func(t *testing.T) {
			handler := session.Restrict(security.AnyAccount(), session.WithRedirectBack())(http.NotFoundHandler())
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/account", nil))
			require.Equal(t, http.StatusUnauthorized, w.Code)
		},
	)
}

type testServer struct {
	t        *testing.T
	store    *session.Store
	http     *http.Client
	endpoint string
}

func newTestServer(t *testing.T, store *session.Store, admin *security.Role) *testServer {
	t.Helper()
	r := chi.NewRouter()
	r.Use(store.Middleware)
	r.With(session.Restrict(security.RequireRole(admin))).Get(
		"/admin", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	)
	r.With(session.Restrict(security.AnyAccount())).Get(
		"/account", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	return &testServer{
		t:        t,
		store:    store,
		http:     srv.Client(),
		endpoint: srv.URL,
	}
}

func (s *testServer) seed(options ...session.AttachOption) string {
	s.t.Helper()
	token, err := s.store.Seed(context.Background(), options...)
	require.NoError(s.t, err)
	return token
}

func (s *testServer) get(path string, token string) int {
	s.t.Helper()
	req, err := http.NewRequest(http.MethodGet, s.endpoint+path, nil)
	require.NoError(s.t, err)
	if token != "" {
		req.AddCookie(&http.Cookie{Name: s.store.Cookie(), Value: token})
	}
	res, err := s.http.Do(req)
	require.NoError(s.t, err)
	require.NoError(s.t, res.Body.Close())
	return res.StatusCode
}

type racingLogoutCache struct {
	*cache.Client
}

func (c racingLogoutCache) GetJSON(ctx context.Context, key string, target any) (bool, error) {
	found, err := c.Client.GetJSON(ctx, key, target)
	if deleteErr := c.Delete(ctx, key); deleteErr != nil {
		return false, deleteErr
	}
	return found, err
}

type failingCache struct {
	*cache.Client
}

func (failingCache) GetJSON(context.Context, string, any) (bool, error) {
	return false, errors.New("cache down")
}

func TestMiddleware(t *testing.T) {
	t.Run(
		"destroy clears the profile within the request", func(t *testing.T) {
			redisClient := testredis.New(t)
			store := session.NewStore(cache.NewClient(cache.NewRedisDriver(redisClient)))
			token, err := store.Seed(t.Context(), session.WithID("user"))
			require.NoError(t, err)
			var exists bool
			cookies := serveSession(t, store, token, func(w http.ResponseWriter, r *http.Request) {
				session.Destroy(w, r)
				exists = session.Exists(r.Context())
			})
			require.False(t, exists)
			require.Equal(t, -1, findSessionCookie(t, cookies, store).MaxAge)
			require.Zero(t, redisClient.Exists(t.Context(), "session:"+token).Val())
		},
	)

	t.Run(
		"invalid stored profile is destroyed", func(t *testing.T) {
			redisClient := testredis.New(t)
			store := session.NewStore(cache.NewClient(cache.NewRedisDriver(redisClient)))
			token, err := store.Seed(t.Context(), session.WithID("user"))
			require.NoError(t, err)
			require.NoError(t, redisClient.Set(t.Context(), "session:"+token, "{", time.Hour).Err())
			cookies := serveSession(t, store, token, func(http.ResponseWriter, *http.Request) {})
			require.Equal(t, -1, findSessionCookie(t, cookies, store).MaxAge)
			require.Zero(t, redisClient.Exists(t.Context(), "session:"+token).Val())
		},
	)

	t.Run(
		"cache failure keeps the cookie", func(t *testing.T) {
			store := session.NewStore(failingCache{cache.NewClient(cache.NewRedisDriver(testredis.New(t)))})
			var exists bool
			cookies := serveSession(t, store, "token", func(_ http.ResponseWriter, r *http.Request) {
				exists = session.Exists(r.Context())
			})
			require.False(t, exists)
			require.Empty(t, cookies)
		},
	)

	t.Run(
		"session removed by a parallel logout is not restored", func(t *testing.T) {
			redisClient := testredis.New(t)
			store := session.NewStore(racingLogoutCache{cache.NewClient(cache.NewRedisDriver(redisClient))})
			token, err := store.Seed(t.Context(), session.WithID("user"))
			require.NoError(t, err)
			var exists bool
			cookies := serveSession(t, store, token, func(_ http.ResponseWriter, r *http.Request) {
				exists = session.Exists(r.Context())
			})
			require.False(t, exists)
			require.Equal(t, -1, findSessionCookie(t, cookies, store).MaxAge)
			require.Zero(t, redisClient.Exists(t.Context(), "session:"+token).Val())
		},
	)

	t.Run(
		"active session slides its expiration", func(t *testing.T) {
			redisClient := testredis.New(t)
			store := session.NewStore(cache.NewClient(cache.NewRedisDriver(redisClient)))
			token, err := store.Seed(t.Context(), session.WithID("user"))
			require.NoError(t, err)
			require.NoError(t, redisClient.Expire(t.Context(), "session:"+token, time.Minute).Err())
			var exists bool
			cookies := serveSession(t, store, token, func(_ http.ResponseWriter, r *http.Request) {
				exists = session.Exists(r.Context())
			})
			require.True(t, exists)
			require.Equal(t, token, findSessionCookie(t, cookies, store).Value)
			require.Greater(t, redisClient.TTL(t.Context(), "session:"+token).Val(), time.Hour)
		},
	)
}

func serveSession(t *testing.T, store *session.Store, token string, handler http.HandlerFunc) []*http.Cookie {
	t.Helper()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: store.Cookie(), Value: token})
	w := httptest.NewRecorder()
	store.Middleware(handler).ServeHTTP(w, r)
	res := w.Result()
	require.NoError(t, res.Body.Close())
	return res.Cookies()
}

func findSessionCookie(t *testing.T, cookies []*http.Cookie, store *session.Store) *http.Cookie {
	t.Helper()
	for _, c := range slices.Backward(cookies) {
		if c.Name == store.Cookie() {
			return c
		}
	}
	require.FailNow(t, "session cookie not set")
	return nil
}
