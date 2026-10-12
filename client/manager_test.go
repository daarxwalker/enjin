package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/infra/cache"
	"github.com/daarxwalker/enjin/internal/testredis"
	"github.com/daarxwalker/enjin/token"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestManager(t *testing.T) {
	infraCtx := context.Background()
	redisClient := testredis.New(t)

	t.Run(
		"expire value without ttl with cookie", func(t *testing.T) {
			c := newTestClient(t, redisClient)
			c.get(infraCtx, "/attach?key=a")
			ttl := c.ttl(infraCtx)
			require.Greater(t, ttl, DefaultTTL-time.Minute)
			require.LessOrEqual(t, ttl, DefaultTTL)
		},
	)

	t.Run(
		"expire values with ttl with latest value", func(t *testing.T) {
			c := newTestClient(t, redisClient)
			c.get(infraCtx, "/attach?key=a&ttl=1m")
			c.get(infraCtx, "/attach?key=b&ttl=10m")
			ttl := c.ttl(infraCtx)
			require.Greater(t, ttl, 9*time.Minute)
			require.LessOrEqual(t, ttl, 10*time.Minute)
		},
	)

	t.Run(
		"replace ttl when value attached without ttl", func(t *testing.T) {
			c := newTestClient(t, redisClient)
			c.get(infraCtx, "/attach?key=a&ttl=1m")
			c.get(infraCtx, "/attach?key=a")
			require.Greater(t, c.ttl(infraCtx), DefaultTTL-time.Minute)
		},
	)

	t.Run(
		"keep stored values on unchanged request", func(t *testing.T) {
			c := newTestClient(t, redisClient)
			c.get(infraCtx, "/attach?key=a&ttl=10m")
			require.NoError(t, redisClient.PExpire(infraCtx, c.cacheKey(), time.Hour).Err())
			c.get(infraCtx, "/")
			require.Greater(t, c.ttl(infraCtx), 59*time.Minute)
			require.Equal(t, "a", c.get(infraCtx, "/value"))
		},
	)

	t.Run(
		"drop expired value", func(t *testing.T) {
			c := newTestClient(t, redisClient)
			c.get(infraCtx, "/attach?key=a&ttl=1ms")
			time.Sleep(5 * time.Millisecond)
			require.Empty(t, c.get(infraCtx, "/value"))
			require.EqualValues(t, 0, redisClient.Exists(infraCtx, c.cacheKey()).Val())
		},
	)

	t.Run(
		"delete key after last value destroyed", func(t *testing.T) {
			c := newTestClient(t, redisClient)
			c.get(infraCtx, "/attach?key=a")
			require.EqualValues(t, 1, redisClient.Exists(infraCtx, c.cacheKey()).Val())
			c.get(infraCtx, "/destroy?key=a")
			require.EqualValues(t, 0, redisClient.Exists(infraCtx, c.cacheKey()).Val())
		},
	)

	t.Run(
		"typed values survive the cache", func(t *testing.T) {
			c := newTestClient(t, redisClient)
			want := `42 {Name:a Count:2} [x y] ""`
			require.Equal(t, want, c.get(infraCtx, "/attach-typed"))
			require.Equal(t, want, c.get(infraCtx, "/typed"))
		},
	)

	t.Run(
		"keep valid id", func(t *testing.T) {
			c := newTestClient(t, redisClient)
			c.get(infraCtx, "/")
			id := c.id()
			require.True(t, token.Valid(id))
			c.get(infraCtx, "/")
			require.Equal(t, id, c.id())
		},
	)

	t.Run(
		"replace invalid id", func(t *testing.T) {
			for _, invalid := range []string{
				"short",
				"..%2F..%2Fx",
				strings.Repeat("A", 4000),
				"ABCDEFGHabcdefgh",
				strings.Repeat("!", token.Length),
			} {
				c := newTestClient(t, redisClient)
				c.setID(invalid)
				c.get(infraCtx, "/attach?key=a")
				require.True(t, token.Valid(c.id()))
				require.EqualValues(t, 0, redisClient.Exists(infraCtx, "client:"+invalid).Val())
			}
		},
	)
}

type typedValue struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func formatTypedValues(ctx context.Context) string {
	return fmt.Sprintf("%d %+v %v %q", ValueFrom[int](ctx, "int"), ValueFrom[typedValue](ctx, "struct"), ValueFrom[[]string](ctx, "slice"), ValueFrom[string](ctx, "int"))
}

type testClient struct {
	t        *testing.T
	redis    *redis.Client
	http     *http.Client
	endpoint *url.URL
	cookie   string
}

func newTestClient(t *testing.T, redisClient *redis.Client) *testClient {
	t.Helper()
	m := NewManager(cache.NewClient(cache.NewRedisDriver(redisClient)))

	mux := http.NewServeMux()
	mux.HandleFunc(
		"/", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	)
	mux.HandleFunc(
		"/attach", func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			if raw := r.URL.Query().Get("ttl"); raw != "" {
				ttl, err := time.ParseDuration(raw)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				AttachValue(r.Context(), key, key, ttl)
			} else {
				AttachValue(r.Context(), key, key)
			}
			w.WriteHeader(http.StatusOK)
		},
	)
	mux.HandleFunc(
		"/destroy", func(w http.ResponseWriter, r *http.Request) {
			DestroyValue(r.Context(), r.URL.Query().Get("key"))
			w.WriteHeader(http.StatusOK)
		},
	)
	mux.HandleFunc(
		"/value", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(ValueFrom[string](r.Context(), "a")))
		},
	)
	mux.HandleFunc(
		"/attach-typed", func(w http.ResponseWriter, r *http.Request) {
			AttachValue(r.Context(), "int", 42)
			AttachValue(r.Context(), "struct", typedValue{Name: "a", Count: 2})
			AttachValue(r.Context(), "slice", []string{"x", "y"})
			_, _ = fmt.Fprint(w, formatTypedValues(r.Context()))
		},
	)
	mux.HandleFunc(
		"/typed", func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprint(w, formatTypedValues(r.Context()))
		},
	)

	srv := httptest.NewServer(m.Middleware(mux))
	t.Cleanup(srv.Close)

	endpoint, err := url.Parse(srv.URL)
	require.NoError(t, err)
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	return &testClient{
		t:        t,
		redis:    redisClient,
		http:     &http.Client{Jar: jar},
		endpoint: endpoint,
	}
}

func (c *testClient) get(ctx context.Context, path string) string {
	c.t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint.String()+path, nil)
	require.NoError(c.t, err)
	if c.cookie != "" {
		req.Header.Set("Cookie", DefaultCookie+"="+c.cookie)
		c.cookie = ""
	}
	res, err := c.http.Do(req)
	require.NoError(c.t, err)
	body, err := io.ReadAll(res.Body)
	require.NoError(c.t, err)
	require.NoError(c.t, res.Body.Close())
	require.Equal(c.t, http.StatusOK, res.StatusCode)
	return string(body)
}

func (c *testClient) id() string {
	c.t.Helper()
	for _, cookie := range c.http.Jar.Cookies(c.endpoint) {
		if cookie.Name == DefaultCookie {
			return cookie.Value
		}
	}
	c.t.Fatal("client cookie not found")
	return ""
}

func (c *testClient) setID(id string) {
	c.t.Helper()
	c.cookie = id
}

func (c *testClient) cacheKey() string {
	c.t.Helper()
	return "client:" + c.id()
}

func (c *testClient) ttl(ctx context.Context) time.Duration {
	c.t.Helper()
	ttl, err := c.redis.PTTL(ctx, c.cacheKey()).Result()
	require.NoError(c.t, err)
	return ttl
}
