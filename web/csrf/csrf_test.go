package csrf

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/client"
	"github.com/daarxwalker/enjin/infra/cache"
	"github.com/daarxwalker/enjin/internal/testredis"
	"github.com/daarxwalker/enjin/token"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestInputManager(t *testing.T) {
	infraCtx := context.Background()
	redisClient := testredis.New(t)

	t.Run(
		"create token", func(t *testing.T) {
			c := newInputClient(t, redisClient, NewRedisDriver(redisClient))
			require.NotEmpty(t, c.token())
		},
	)

	t.Run(
		"verify token", func(t *testing.T) {
			c := newInputClient(t, redisClient, NewRedisDriver(redisClient))
			csrfToken := c.token()
			require.Equal(t, http.StatusOK, c.submit(csrfToken))
			require.Equal(t, http.StatusForbidden, c.submit(csrfToken))
		},
	)

	t.Run(
		"reject missing token", func(t *testing.T) {
			c := newInputClient(t, redisClient, NewRedisDriver(redisClient))
			c.token()
			require.Equal(t, http.StatusForbidden, c.submit(""))
			require.Equal(t, http.StatusForbidden, c.submit("unknown"))
		},
	)

	t.Run(
		"non positive ttl and limit keep defaults", func(t *testing.T) {
			c := newInputClient(t, redisClient, NewRedisDriver(redisClient), WithTTL(0), WithLimit(-1))
			csrfToken := c.token()
			require.NotEmpty(t, csrfToken)
			require.Equal(t, http.StatusOK, c.submit(csrfToken))
		},
	)

	t.Run(
		"reject token in query string", func(t *testing.T) {
			c := newInputClient(t, redisClient, NewRedisDriver(redisClient))
			csrfToken := c.token()
			res, err := c.http.Post(c.endpoint+"?"+url.Values{Field: {csrfToken}}.Encode(), "application/x-www-form-urlencoded", strings.NewReader(""))
			require.NoError(t, err)
			require.NoError(t, res.Body.Close())
			require.Equal(t, http.StatusForbidden, res.StatusCode)
			require.Equal(t, http.StatusOK, c.submit(csrfToken))
		},
	)

	t.Run(
		"verify multiple tokens", func(t *testing.T) {
			c := newInputClient(t, redisClient, NewRedisDriver(redisClient))
			first, second := c.token(), c.token()
			require.NotEqual(t, first, second)
			require.Equal(t, http.StatusOK, c.submit(second))
			require.Equal(t, http.StatusOK, c.submit(first))
			require.Equal(t, http.StatusForbidden, c.submit(first))
			require.Equal(t, http.StatusForbidden, c.submit(second))
		},
	)

	t.Run(
		"keep tokens after rejection", func(t *testing.T) {
			c := newInputClient(t, redisClient, NewRedisDriver(redisClient))
			csrfToken := c.token()
			require.Equal(t, http.StatusForbidden, c.submit("unknown"))
			require.Equal(t, http.StatusOK, c.submit(csrfToken))
		},
	)

	t.Run(
		"reject expired token", func(t *testing.T) {
			c := newInputClient(t, redisClient, NewRedisDriver(redisClient))
			valid := c.token()
			expired := token.New()
			c.seed(map[string]int64{
				valid:   time.Now().Add(time.Hour).UnixMilli(),
				expired: time.Now().Add(-time.Minute).UnixMilli(),
			})
			require.Equal(t, http.StatusForbidden, c.submit(expired))
			require.Equal(t, http.StatusOK, c.submit(valid))
		},
	)

	t.Run(
		"evict oldest token over limit", func(t *testing.T) {
			c := newInputClient(t, redisClient, NewRedisDriver(redisClient), WithLimit(2))
			c.token()
			oldest, older := token.New(), token.New()
			c.seed(map[string]int64{
				oldest: time.Now().Add(time.Hour).UnixMilli(),
				older:  time.Now().Add(2 * time.Hour).UnixMilli(),
			})
			newest := c.token()
			require.Equal(t, http.StatusForbidden, c.submit(oldest))
			require.Equal(t, http.StatusOK, c.submit(older))
			require.Equal(t, http.StatusOK, c.submit(newest))
		},
	)

	t.Run(
		"require token", func(t *testing.T) {
			c := newInputClient(t, redisClient, NewRedisDriver(redisClient))
			csrfToken := c.token()
			require.Equal(t, http.StatusForbidden, c.submitRequired(""))
			require.Equal(t, http.StatusForbidden, c.submitRequired("unknown"))
			require.Equal(t, http.StatusOK, c.submitRequired(csrfToken))
			require.Equal(t, http.StatusForbidden, c.submitRequired(csrfToken))
		},
	)

	t.Run(
		"destroy key after last token", func(t *testing.T) {
			c := newInputClient(t, redisClient, NewRedisDriver(redisClient))
			csrfToken := c.token()
			require.EqualValues(t, 1, redisClient.Exists(infraCtx, createCacheKey(c.clientID())).Val())
			require.Equal(t, http.StatusOK, c.submit(csrfToken))
			require.EqualValues(t, 0, redisClient.Exists(infraCtx, createCacheKey(c.clientID())).Val())
		},
	)

	t.Run(
		"consume token once concurrently", func(t *testing.T) {
			c := newInputClient(t, redisClient, NewRedisDriver(redisClient))
			csrfToken := c.token()
			statuses := make(chan int, 20)
			var wg sync.WaitGroup
			for range cap(statuses) {
				wg.Go(func() {
					statuses <- c.submit(csrfToken)
				})
			}
			wg.Wait()
			close(statuses)
			var accepted int
			for status := range statuses {
				if status == http.StatusOK {
					accepted++
				}
			}
			require.Equal(t, 1, accepted)
		},
	)

	t.Run(
		"reject on driver error", func(t *testing.T) {
			c := newInputClient(t, redisClient, failingDriver{})
			require.Equal(t, http.StatusForbidden, c.submit("token"))
		},
	)

	t.Run(
		"token from header", func(t *testing.T) {
			c := newInputClient(t, redisClient, NewRedisDriver(redisClient))
			csrfToken := c.token()
			require.Equal(t, http.StatusOK, c.postHeader(c.endpoint+"require", csrfToken))
			require.Equal(t, http.StatusForbidden, c.postHeader(c.endpoint+"require", csrfToken))
		},
	)

	t.Run(
		"require with reject handler", func(t *testing.T) {
			reject := http.HandlerFunc(
				func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusConflict)
				},
			)
			c := newInputClient(t, redisClient, NewRedisDriver(redisClient), WithRejectHandler(reject))
			require.Equal(t, http.StatusConflict, c.submitRequired("unknown"))
			require.Equal(t, http.StatusOK, c.submitRequired(c.token()))
		},
	)

	t.Run(
		"verify without middleware rejects", func(t *testing.T) {
			m := NewInputManager(NewRedisDriver(redisClient))
			var rejected bool
			h := m.Verify(
				http.HandlerFunc(
					func(_ http.ResponseWriter, r *http.Request) {
						rejected = RejectedFrom(r.Context())
					},
				),
			)
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(url.Values{Field: {"token"}}.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			h.ServeHTTP(httptest.NewRecorder(), r)
			require.True(t, rejected)
		},
	)
}

func TestCookieManager(t *testing.T) {
	t.Run(
		"issue readable cookie", func(t *testing.T) {
			c := newCookieClient(t)
			cookie := c.cookie()
			require.NotEmpty(t, cookie.Value)
			require.False(t, cookie.HttpOnly)
		},
	)

	t.Run(
		"keep cookie across requests", func(t *testing.T) {
			c := newCookieClient(t)
			first := c.cookie().Value
			require.Equal(t, first, c.cookie().Value)
		},
	)

	t.Run(
		"verify matching header", func(t *testing.T) {
			c := newCookieClient(t)
			csrfToken := c.cookie().Value
			require.Equal(t, http.StatusOK, c.submit(csrfToken))
			require.Equal(t, http.StatusOK, c.submit(csrfToken))
		},
	)

	t.Run(
		"reject missing or mismatched header", func(t *testing.T) {
			c := newCookieClient(t)
			c.cookie()
			require.Equal(t, http.StatusForbidden, c.submit(""))
			require.Equal(t, http.StatusForbidden, c.submit("unknown"))
		},
	)

	t.Run(
		"require matching header", func(t *testing.T) {
			c := newCookieClient(t)
			csrfToken := c.cookie().Value
			require.Equal(t, http.StatusForbidden, c.submitRequired("unknown"))
			require.Equal(t, http.StatusOK, c.submitRequired(csrfToken))
		},
	)

	t.Run(
		"reject missing cookie", func(t *testing.T) {
			c := newCookieClient(t)
			csrfToken := c.cookie().Value
			c.http.Jar, _ = cookiejar.New(nil)
			require.Equal(t, http.StatusForbidden, c.submit(csrfToken))
		},
	)
}

type failingDriver struct{}

func (failingDriver) Add(context.Context, string, string, time.Time, int) error {
	return errors.New("add failed")
}

func (failingDriver) Take(context.Context, string, string, time.Time) (bool, error) {
	return true, errors.New("take failed")
}

type inputClient struct {
	t        *testing.T
	redis    *redis.Client
	http     *http.Client
	endpoint string
}

func newInputClient(t *testing.T, redisClient *redis.Client, driver Driver, options ...InputOption) *inputClient {
	t.Helper()
	csrf := NewInputManager(driver, options...)

	r := chi.NewRouter()
	r.Use(client.NewManager(cache.NewClient(cache.NewRedisDriver(redisClient))).Middleware, csrf.Middleware)
	r.Get(
		"/", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(csrf.Create(r.Context())))
		},
	)
	r.With(csrf.Verify).Post(
		"/", func(w http.ResponseWriter, r *http.Request) {
			if RejectedFrom(r.Context()) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusOK)
		},
	)

	r.With(csrf.Require).Post(
		"/require", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	return &inputClient{
		t:        t,
		redis:    redisClient,
		http:     &http.Client{Jar: jar},
		endpoint: srv.URL + "/",
	}
}

func (c *inputClient) token() string {
	c.t.Helper()
	res, err := c.http.Get(c.endpoint)
	require.NoError(c.t, err)
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	require.NoError(c.t, err)
	require.Equal(c.t, http.StatusOK, res.StatusCode)
	return string(body)
}

func (c *inputClient) submit(token string) int {
	c.t.Helper()
	return c.post(c.endpoint, token)
}

func (c *inputClient) submitRequired(token string) int {
	c.t.Helper()
	return c.post(c.endpoint+"require", token)
}

func (c *inputClient) postHeader(endpoint, token string) int {
	c.t.Helper()
	req, err := http.NewRequest(http.MethodPost, endpoint, nil)
	require.NoError(c.t, err)
	req.Header.Set(Header, token)
	res, err := c.http.Do(req)
	require.NoError(c.t, err)
	require.NoError(c.t, res.Body.Close())
	return res.StatusCode
}

func (c *inputClient) post(endpoint, token string) int {
	c.t.Helper()
	body := make(url.Values)
	body.Set(Field, token)
	res, err := c.http.Post(endpoint, "application/x-www-form-urlencoded", strings.NewReader(body.Encode()))
	require.NoError(c.t, err)
	require.NoError(c.t, res.Body.Close())
	return res.StatusCode
}

func (c *inputClient) clientID() string {
	c.t.Helper()
	endpoint, err := url.Parse(c.endpoint)
	require.NoError(c.t, err)
	for _, cookie := range c.http.Jar.Cookies(endpoint) {
		if cookie.Name == client.DefaultCookie {
			return cookie.Value
		}
	}
	c.t.Fatal("client cookie not found")
	return ""
}

func (c *inputClient) seed(tokens map[string]int64) {
	c.t.Helper()
	members := make([]redis.Z, 0, len(tokens))
	for csrfToken, expiresAt := range tokens {
		members = append(members, redis.Z{Score: float64(expiresAt), Member: csrfToken})
	}
	cacheKey := createCacheKey(c.clientID())
	_, err := c.redis.TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
		pipe.Del(context.Background(), cacheKey)
		pipe.ZAdd(context.Background(), cacheKey, members...)
		return nil
	})
	require.NoError(c.t, err)
}

type cookieClient struct {
	t        *testing.T
	http     *http.Client
	endpoint string
}

func newCookieClient(t *testing.T) *cookieClient {
	t.Helper()
	csrf := NewCookieManager()

	r := chi.NewRouter()
	r.Use(csrf.Middleware)
	r.Get(
		"/", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	)
	r.With(csrf.Verify).Post(
		"/", func(w http.ResponseWriter, r *http.Request) {
			if RejectedFrom(r.Context()) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusOK)
		},
	)
	r.With(csrf.Require).Post(
		"/required", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	return &cookieClient{
		t:        t,
		http:     &http.Client{Jar: jar},
		endpoint: srv.URL + "/",
	}
}

func (c *cookieClient) cookie() *http.Cookie {
	c.t.Helper()
	res, err := c.http.Get(c.endpoint)
	require.NoError(c.t, err)
	require.NoError(c.t, res.Body.Close())
	require.Equal(c.t, http.StatusOK, res.StatusCode)
	for _, cookie := range res.Cookies() {
		if cookie.Name == DefaultCookie {
			return cookie
		}
	}
	c.t.Fatal("csrf cookie not found")
	return nil
}

func (c *cookieClient) submit(token string) int {
	c.t.Helper()
	return c.post(c.endpoint, token)
}

func (c *cookieClient) submitRequired(token string) int {
	c.t.Helper()
	return c.post(c.endpoint+"required", token)
}

func (c *cookieClient) post(endpoint, token string) int {
	c.t.Helper()
	req, err := http.NewRequest(http.MethodPost, endpoint, nil)
	require.NoError(c.t, err)
	if len(token) > 0 {
		req.Header.Set(Header, token)
	}
	res, err := c.http.Do(req)
	require.NoError(c.t, err)
	require.NoError(c.t, res.Body.Close())
	return res.StatusCode
}
