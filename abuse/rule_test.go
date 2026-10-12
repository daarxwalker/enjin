package abuse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestRule(t *testing.T) {
	start := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)

	t.Run("rate limit allows up to limit and then returns verdict", func(t *testing.T) {
		store := NewMemoryStore(t.Context())
		rule := CreateRateLimitRule(CountedByIP, 3, time.Minute, VerdictDelay)
		id := Identity{IPKey: "203.0.113.7/32"}
		for i := range 3 {
			decision, err := rule.CheckRequest(newRuleRequest(""), id, store, "p|r", start.Add(time.Duration(i)*time.Second))
			require.NoError(t, err)
			require.Equal(t, VerdictAllow, decision.Verdict)
		}
		decision, err := rule.CheckRequest(newRuleRequest(""), id, store, "p|r", start.Add(4*time.Second))
		require.NoError(t, err)
		require.Equal(t, VerdictDelay, decision.Verdict)
		require.Equal(t, rule.Name(), decision.Rule)
		require.Equal(t, time.Minute, decision.RetryAfter)
	})

	t.Run("rate limit skips empty scope value", func(t *testing.T) {
		store := NewMemoryStore(t.Context())
		rule := CreateRateLimitRule(CountedByUserID, 0, time.Minute, VerdictBlock)
		decision, err := rule.CheckRequest(newRuleRequest(""), Identity{IPKey: "203.0.113.7/32"}, store, "p|r", start)
		require.NoError(t, err)
		require.Equal(t, VerdictAllow, decision.Verdict)
	})

	t.Run("keyed scope counts per key", func(t *testing.T) {
		store := NewMemoryStore(t.Context())
		rule := CreateRateLimitRule(CreateCountedByFormField("email"), 1, time.Hour, VerdictBlock)
		first, err := rule.CheckRequest(newRuleRequest("a@example.com"), Identity{}, store, "p|r", start)
		require.NoError(t, err)
		other, err := rule.CheckRequest(newRuleRequest("b@example.com"), Identity{}, store, "p|r", start)
		require.NoError(t, err)
		second, err := rule.CheckRequest(newRuleRequest("a@example.com"), Identity{}, store, "p|r", start)
		require.NoError(t, err)
		require.Equal(t, VerdictAllow, first.Verdict)
		require.Equal(t, VerdictAllow, other.Verdict)
		require.Equal(t, VerdictBlock, second.Verdict)
	})

	t.Run("form field value is normalized and stored as fixed length hash", func(t *testing.T) {
		countedBy := CreateCountedByFormField("email")
		long := countedBy.readValue(newRuleRequest(strings.Repeat("a", 100_000)), Identity{})
		require.Len(t, long, 32)
		require.Equal(t, countedBy.readValue(newRuleRequest("a@example.com"), Identity{}), countedBy.readValue(newRuleRequest("A@Example.com"), Identity{}))
		require.Empty(t, countedBy.readValue(newRuleRequest(""), Identity{}))
	})

	t.Run("non positive window is rejected", func(t *testing.T) {
		store := NewMemoryStore(t.Context())
		_, err := CreateRateLimitRule(CountedByIP, 1, 0, VerdictBlock).CheckRequest(newRuleRequest(""), Identity{IPKey: "203.0.113.7/32"}, store, "p|r", start)
		require.ErrorIs(t, err, ErrInvalidWindow)
		_, err = CreateDistinctURLParamRule(CountedByIP, "slug", 1, -time.Second, VerdictBlock).CheckRequest(newRuleSlugRequest("a"), Identity{IPKey: "203.0.113.7/32"}, store, "p|r", start)
		require.ErrorIs(t, err, ErrInvalidWindow)
	})

	t.Run("distinct counts different resources only", func(t *testing.T) {
		store := NewMemoryStore(t.Context())
		rule := CreateDistinctURLParamRule(CountedByIP, "slug", 2, time.Hour, VerdictBlock)
		id := Identity{IPKey: "203.0.113.7/32"}
		for _, slug := range []string{"a", "a", "b", "b", "a"} {
			decision, err := rule.CheckRequest(newRuleSlugRequest(slug), id, store, "p|r", start)
			require.NoError(t, err)
			require.Equal(t, VerdictAllow, decision.Verdict)
		}
		decision, err := rule.CheckRequest(newRuleSlugRequest("c"), id, store, "p|r", start)
		require.NoError(t, err)
		require.Equal(t, VerdictBlock, decision.Verdict)
	})

	t.Run("distinct forgets resources older than window", func(t *testing.T) {
		store := NewMemoryStore(t.Context())
		rule := CreateDistinctURLParamRule(CountedByIP, "slug", 1, time.Hour, VerdictBlock)
		id := Identity{IPKey: "203.0.113.7/32"}
		_, err := rule.CheckRequest(newRuleSlugRequest("a"), id, store, "p|r", start)
		require.NoError(t, err)
		decision, err := rule.CheckRequest(newRuleSlugRequest("b"), id, store, "p|r", start.Add(2*time.Hour))
		require.NoError(t, err)
		require.Equal(t, VerdictAllow, decision.Verdict)
	})

	t.Run("user agent flags libraries and empty agents", func(t *testing.T) {
		rule := CreateUserAgentRule(VerdictBlock)
		for _, test := range []struct {
			userAgent string
			want      Verdict
		}{
			{userAgent: "", want: VerdictBlock},
			{userAgent: "curl/8.7.1", want: VerdictBlock},
			{userAgent: "python-requests/2.32", want: VerdictBlock},
			{userAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 HeadlessChrome/129.0", want: VerdictBlock},
			{userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_6) AppleWebKit/605.1.15 Version/18.0 Safari/605.1.15", want: VerdictAllow},
			{userAgent: "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", want: VerdictAllow},
		} {
			r := newRuleRequest("")
			r.Header.Set("User-Agent", test.userAgent)
			decision, err := rule.CheckRequest(r, Identity{}, nil, "", start)
			require.NoError(t, err)
			require.Equal(t, test.want, decision.Verdict, test.userAgent)
		}
	})
}

func newRuleRequest(email string) *http.Request {
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/?email="+email, nil)
	r.Header.Set("User-Agent", "Mozilla/5.0")
	return r
}

func newRuleSlugRequest(slug string) *http.Request {
	r := newRuleRequest("")
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("slug", slug)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeContext))
}
