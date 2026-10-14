package abuse

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type guardTestClock struct {
	now time.Time
}

type guardTestMetrics struct {
	decisions []guardTestDecision
}

type guardTestDecision struct {
	policy  string
	rule    string
	verdict Verdict
	dryRun  bool
}

type guardTestRecorder struct {
	events []Event
}

type guardFailingStore struct {
	Store
}

func (c *guardTestClock) Now() time.Time {
	return c.now
}

func (m *guardTestMetrics) CountDecision(policy, rule string, verdict Verdict, dryRun bool) {
	m.decisions = append(m.decisions, guardTestDecision{policy: policy, rule: rule, verdict: verdict, dryRun: dryRun})
}

func (rec *guardTestRecorder) RecordEvent(_ context.Context, event Event) {
	rec.events = append(rec.events, event)
}

func (guardFailingStore) HitCounter(Key, time.Duration, time.Time) (int, error) {
	return 0, errors.New("store down")
}

func (guardFailingStore) LoadPenalty(Key, time.Time) (Penalty, error) {
	return Penalty{}, errors.New("store down")
}

func TestGuard(t *testing.T) {
	start := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)

	t.Run("allows requests under limit", func(t *testing.T) {
		guard := newGuardTestGuard(t, NewMemoryStore(t.Context()), &guardTestClock{now: start}, nil)
		policy := Policy{Name: "login", Rules: []Rule{CreateRateLimitRule(CountedByIP, 2, time.Minute, VerdictBlock)}}
		for range 2 {
			require.Equal(t, http.StatusOK, serveGuard(guard, policy, newGuardRequest("203.0.113.7:1234", "Mozilla/5.0")).Code)
		}
	})

	t.Run("blocks with retry after when limit is exceeded", func(t *testing.T) {
		metrics := new(guardTestMetrics)
		guard := newGuardTestGuard(t, NewMemoryStore(t.Context()), &guardTestClock{now: start}, metrics)
		policy := Policy{Name: "login", Rules: []Rule{CreateRateLimitRule(CountedByIP, 1, 10*time.Minute, VerdictBlock)}}
		serveGuard(guard, policy, newGuardRequest("203.0.113.7:1234", "Mozilla/5.0"))
		w := serveGuard(guard, policy, newGuardRequest("203.0.113.7:1234", "Mozilla/5.0"))
		require.Equal(t, http.StatusTooManyRequests, w.Code)
		require.Equal(t, "600", w.Header().Get("Retry-After"))
		require.Equal(t, []guardTestDecision{{policy: "login", rule: "rate-ip-1-10m0s", verdict: VerdictBlock}}, metrics.decisions)
	})

	t.Run("strictest rule wins", func(t *testing.T) {
		guard := newGuardTestGuard(t, NewMemoryStore(t.Context()), &guardTestClock{now: start}, nil)
		policy := Policy{Name: "page", Rules: []Rule{CreateRateLimitRule(CountedByIP, 100, time.Minute, VerdictDelay), CreateUserAgentRule(VerdictBlock)}}
		require.Equal(t, http.StatusTooManyRequests, serveGuard(guard, policy, newGuardRequest("203.0.113.7:1234", "curl/8.7.1")).Code)
	})

	t.Run("dry run reports and lets request through", func(t *testing.T) {
		metrics := new(guardTestMetrics)
		store := NewMemoryStore(t.Context())
		guard := newGuardTestGuard(t, store, &guardTestClock{now: start}, metrics)
		policy := Policy{Name: "page", Rules: []Rule{CreateUserAgentRule(VerdictBlock)}, DryRun: true}
		require.Equal(t, http.StatusOK, serveGuard(guard, policy, newGuardRequest("203.0.113.7:1234", "curl/8.7.1")).Code)
		require.Equal(t, []guardTestDecision{{policy: "page", rule: "user-agent", verdict: VerdictBlock, dryRun: true}}, metrics.decisions)
		penalty, err := store.LoadPenalty(guard.createPenaltyKey("page", "203.0.113.7/32"), start)
		require.NoError(t, err)
		require.Zero(t, penalty)
	})

	t.Run("repeated violations escalate to long block and decay", func(t *testing.T) {
		clock := &guardTestClock{now: start}
		guard := newGuardTestGuard(t, NewMemoryStore(t.Context()), clock, nil)
		strict := Policy{Name: "page", Rules: []Rule{CreateUserAgentRule(VerdictBlock)}}
		serveGuard(guard, strict, newGuardRequest("203.0.113.7:1234", "curl/8.7.1"))
		require.Equal(t, http.StatusOK, serveGuard(guard, strict, newGuardRequest("203.0.113.7:1234", "Mozilla/5.0")).Code)
		clock.now = start.Add(2 * time.Minute)
		serveGuard(guard, strict, newGuardRequest("203.0.113.7:1234", "curl/8.7.1"))
		w := serveGuard(guard, strict, newGuardRequest("203.0.113.7:1234", "Mozilla/5.0"))
		require.Equal(t, http.StatusTooManyRequests, w.Code)
		require.Equal(t, "3600", w.Header().Get("Retry-After"))
		clock.now = start.Add(63 * time.Minute)
		serveGuard(guard, strict, newGuardRequest("203.0.113.7:1234", "curl/8.7.1"))
		require.Equal(t, "86400", serveGuard(guard, strict, newGuardRequest("203.0.113.7:1234", "Mozilla/5.0")).Header().Get("Retry-After"))
		require.Equal(t, http.StatusOK, serveGuard(guard, strict, newGuardRequest("198.51.100.1:1234", "Mozilla/5.0")).Code)
		clock.now = start.Add(63*time.Minute + 25*time.Hour)
		require.Equal(t, http.StatusOK, serveGuard(guard, strict, newGuardRequest("203.0.113.7:1234", "Mozilla/5.0")).Code)
	})

	t.Run("penalty of one policy does not block another", func(t *testing.T) {
		clock := &guardTestClock{now: start}
		guard := newGuardTestGuard(t, NewMemoryStore(t.Context()), clock, nil)
		strict := Policy{Name: "page", Rules: []Rule{CreateUserAgentRule(VerdictBlock)}}
		open := Policy{Name: "open"}
		serveGuard(guard, strict, newGuardRequest("203.0.113.7:1234", "curl/8.7.1"))
		clock.now = start.Add(2 * time.Minute)
		serveGuard(guard, strict, newGuardRequest("203.0.113.7:1234", "curl/8.7.1"))
		require.Equal(t, http.StatusTooManyRequests, serveGuard(guard, strict, newGuardRequest("203.0.113.7:1234", "Mozilla/5.0")).Code)
		require.Equal(t, http.StatusOK, serveGuard(guard, open, newGuardRequest("203.0.113.7:1234", "Mozilla/5.0")).Code)
	})

	t.Run("delay does not raise penalty", func(t *testing.T) {
		clock := &guardTestClock{now: start}
		store := NewMemoryStore(t.Context())
		guard := newGuardTestGuard(t, store, clock, nil)
		policy := Policy{Name: "page", Rules: []Rule{CreateRateLimitRule(CountedByIP, 0, time.Minute, VerdictDelay)}}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		for i := range 3 {
			clock.now = start.Add(time.Duration(i) * 2 * time.Minute)
			serveGuard(guard, policy, newGuardRequest("203.0.113.7:1234", "Mozilla/5.0").WithContext(ctx))
		}
		penalty, err := store.LoadPenalty(guard.createPenaltyKey("page", "203.0.113.7/32"), clock.now)
		require.NoError(t, err)
		require.Zero(t, penalty)
	})

	t.Run("calm hours after block lower the level", func(t *testing.T) {
		clock := &guardTestClock{now: start}
		guard := newGuardTestGuard(t, NewMemoryStore(t.Context()), clock, nil)
		strict := Policy{Name: "page", Rules: []Rule{CreateUserAgentRule(VerdictBlock)}}
		serveGuard(guard, strict, newGuardRequest("203.0.113.7:1234", "curl/8.7.1"))
		clock.now = start.Add(2 * time.Minute)
		serveGuard(guard, strict, newGuardRequest("203.0.113.7:1234", "curl/8.7.1"))
		clock.now = start.Add(2*time.Minute + 3*time.Hour)
		serveGuard(guard, strict, newGuardRequest("203.0.113.7:1234", "curl/8.7.1"))
		require.Equal(t, http.StatusOK, serveGuard(guard, strict, newGuardRequest("203.0.113.7:1234", "Mozilla/5.0")).Code)
	})

	t.Run("delay waits and then serves", func(t *testing.T) {
		guard := newGuardTestGuard(t, NewMemoryStore(t.Context()), &guardTestClock{now: start}, nil)
		policy := Policy{Name: "page", Rules: []Rule{CreateRateLimitRule(CountedByIP, 0, time.Minute, VerdictDelay)}}
		begin := time.Now()
		require.Equal(t, http.StatusOK, serveGuard(guard, policy, newGuardRequest("203.0.113.7:1234", "Mozilla/5.0")).Code)
		require.GreaterOrEqual(t, time.Since(begin), minDelay)
	})

	t.Run("delay stops when client goes away", func(t *testing.T) {
		guard := newGuardTestGuard(t, NewMemoryStore(t.Context()), &guardTestClock{now: start}, nil)
		policy := Policy{Name: "page", Rules: []Rule{CreateRateLimitRule(CountedByIP, 0, time.Minute, VerdictDelay)}}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		r := newGuardRequest("203.0.113.7:1234", "Mozilla/5.0").WithContext(ctx)
		require.Equal(t, http.StatusNoContent, serveGuard(guard, policy, r).Code)
	})

	t.Run("failing store fails open", func(t *testing.T) {
		guard := newGuardTestGuard(t, guardFailingStore{}, &guardTestClock{now: start}, nil)
		policy := Policy{Name: "login", Rules: []Rule{CreateRateLimitRule(CountedByIP, 0, time.Minute, VerdictBlock)}}
		require.Equal(t, http.StatusOK, serveGuard(guard, policy, newGuardRequest("203.0.113.7:1234", "Mozilla/5.0")).Code)
	})

	t.Run("records block event", func(t *testing.T) {
		recorder := new(guardTestRecorder)
		guard := newGuardTestRecordingGuard(t, NewMemoryStore(t.Context()), &guardTestClock{now: start}, recorder)
		policy := Policy{Name: "login", Rules: []Rule{CreateRateLimitRule(CountedByIP, 1, 10*time.Minute, VerdictBlock)}}
		serveGuard(guard, policy, newGuardRequest("203.0.113.7:1234", "Mozilla/5.0"))
		require.Empty(t, recorder.events)
		serveGuard(guard, policy, newGuardRequest("203.0.113.7:1234", "Mozilla/5.0"))
		require.Equal(t, []Event{{
			Time:       start,
			Addr:       netip.MustParseAddr("203.0.113.7"),
			IP:         netip.MustParsePrefix("203.0.113.7/32"),
			Policy:     "login",
			Rule:       "rate-ip-1-10m0s",
			Method:     http.MethodGet,
			Path:       "/products/x",
			UserAgent:  "Mozilla/5.0",
			RetryAfter: 10 * time.Minute,
			Verdict:    VerdictBlock,
		}}, recorder.events)
	})

	t.Run("records dry run event", func(t *testing.T) {
		recorder := new(guardTestRecorder)
		guard := newGuardTestRecordingGuard(t, NewMemoryStore(t.Context()), &guardTestClock{now: start}, recorder)
		policy := Policy{Name: "page", Rules: []Rule{CreateUserAgentRule(VerdictBlock)}, DryRun: true}
		require.Equal(t, http.StatusOK, serveGuard(guard, policy, newGuardRequest("203.0.113.7:1234", "curl/8.7.1")).Code)
		require.Len(t, recorder.events, 1)
		require.True(t, recorder.events[0].DryRun)
		require.Equal(t, "user-agent", recorder.events[0].Rule)
		require.Equal(t, "curl/8.7.1", recorder.events[0].UserAgent)
	})

	t.Run("records penalty block", func(t *testing.T) {
		clock := &guardTestClock{now: start}
		recorder := new(guardTestRecorder)
		guard := newGuardTestRecordingGuard(t, NewMemoryStore(t.Context()), clock, recorder)
		strict := Policy{Name: "page", Rules: []Rule{CreateUserAgentRule(VerdictBlock)}}
		serveGuard(guard, strict, newGuardRequest("203.0.113.7:1234", "curl/8.7.1"))
		clock.now = start.Add(2 * time.Minute)
		serveGuard(guard, strict, newGuardRequest("203.0.113.7:1234", "curl/8.7.1"))
		serveGuard(guard, strict, newGuardRequest("203.0.113.7:1234", "Mozilla/5.0"))
		require.Len(t, recorder.events, 3)
		last := recorder.events[2]
		require.Equal(t, RulePenalty, last.Rule)
		require.Equal(t, time.Hour, last.RetryAfter)
		require.Equal(t, "Mozilla/5.0", last.UserAgent)
	})

	t.Run("verified bot gets bot policy on reads", func(t *testing.T) {
		store := NewMemoryStore(t.Context())
		require.NoError(t, store.SaveMark(Key{Namespace: botNamespace, Value: "66.249.66.1"}, true, time.Hour, start))
		guard := NewGuard(
			store,
			WithClock(&guardTestClock{now: start}),
			WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
			WithBotVerifier(NewBotVerifier(store)),
			WithGoodBotPolicy(Policy{Name: "good-bot", Rules: []Rule{CreateRateLimitRule(CountedByIP, 600, time.Minute, VerdictDelay)}}),
		)
		policy := Policy{Name: "page", Rules: []Rule{CreateRateLimitRule(CountedByIP, 0, time.Minute, VerdictBlock)}}
		bot := "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
		require.Equal(t, http.StatusOK, serveGuard(guard, policy, newGuardRequest("66.249.66.1:1234", bot)).Code)
		require.Equal(t, http.StatusTooManyRequests, serveGuard(guard, policy, newGuardRequest("203.0.113.7:1234", bot)).Code)
	})
}

func newGuardTestGuard(t *testing.T, store Store, clock *guardTestClock, metrics *guardTestMetrics) *Guard {
	t.Helper()
	options := []Option{
		WithClock(clock),
		WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
	}
	if metrics != nil {
		options = append(options, WithMetrics(metrics))
	}
	return NewGuard(store, options...)
}

func newGuardTestRecordingGuard(t *testing.T, store Store, clock *guardTestClock, recorder *guardTestRecorder) *Guard {
	t.Helper()
	return NewGuard(
		store,
		WithClock(clock),
		WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		WithRecorder(recorder),
	)
}

func newGuardRequest(remoteAddr, userAgent string) *http.Request {
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/products/x", nil)
	r.RemoteAddr = remoteAddr
	r.Header.Set("User-Agent", userAgent)
	return r
}

func serveGuard(guard *Guard, policy Policy, r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	w.Code = http.StatusNoContent
	guard.Protect(policy)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)
	return w
}
