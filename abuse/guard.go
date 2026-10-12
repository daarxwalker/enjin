package abuse

import (
	"context"
	"fmt"
	"hash/maphash"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"
)

const (
	penaltyNamespace     = "penalty"
	penaltyRule          = "penalty"
	penaltyRaiseInterval = time.Minute
	penaltyDecayInterval = time.Hour
	penaltyTTL           = 25 * time.Hour
	penaltyShortBlock    = time.Hour
	penaltyLongBlock     = 24 * time.Hour
	penaltyShortLevel    = 2
	penaltyLongLevel     = 3
	minDelay             = 500 * time.Millisecond
	delaySpread          = 1500 * time.Millisecond
	blockDefaultRetry    = time.Minute
)

type Guard struct {
	store         Store
	clock         Clock
	metrics       Metrics
	block         http.Handler
	bots          *BotVerifier
	log           *slog.Logger
	hasher        *ipHasher
	goodBotPolicy Policy
	seed          maphash.Seed
}

type boundPolicy struct {
	namespaces []string
	policy     Policy
}

func NewGuard(store Store, options ...Option) *Guard {
	g := &Guard{
		store:  store,
		clock:  systemClock{},
		log:    slog.Default(),
		hasher: new(ipHasher),
		seed:   maphash.MakeSeed(),
	}
	for _, option := range options {
		option(g)
	}
	return g
}

func (g *Guard) Protect(policy Policy) func(http.Handler) http.Handler {
	bound := g.bindPolicy(policy)
	goodBot := g.bindPolicy(g.goodBotPolicy)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			g.serveProtected(w, r, next, bound, goodBot)
		})
	}
}

func (g *Guard) serveProtected(w http.ResponseWriter, r *http.Request, next http.Handler, bound, goodBot boundPolicy) {
	now := g.clock.Now()
	id := createIdentity(r)
	if g.bots != nil && id.Addr.IsValid() {
		id.GoodBot = g.bots.verifyBot(r, id.Addr, now)
	}
	current := bound
	if id.GoodBot && g.isReadMethod(r.Method) && len(goodBot.policy.Rules) > 0 {
		current = goodBot
	}
	decision := g.evaluatePolicy(r, id, current, now)
	decision = g.escalatePenalty(r.Context(), id, current.policy, decision, now)
	if decision.Verdict == VerdictAllow {
		next.ServeHTTP(w, r)
		return
	}
	g.reportDecision(r, id, current.policy, decision, now)
	if current.policy.DryRun {
		next.ServeHTTP(w, r)
		return
	}
	switch decision.Verdict {
	case VerdictDelay:
		if g.waitDelay(r.Context(), id, now) {
			next.ServeHTTP(w, r)
		}
	case VerdictBlock:
		g.respondBlock(w, r, decision)
	case VerdictAllow:
		next.ServeHTTP(w, r)
	}
}

func (g *Guard) bindPolicy(policy Policy) boundPolicy {
	namespaces := make([]string, len(policy.Rules))
	for i, rule := range policy.Rules {
		namespaces[i] = policy.Name + "|" + rule.Name()
	}
	return boundPolicy{policy: policy, namespaces: namespaces}
}

func (g *Guard) evaluatePolicy(r *http.Request, id Identity, bound boundPolicy, now time.Time) Decision {
	var result Decision
	for i, rule := range bound.policy.Rules {
		decision, err := rule.CheckRequest(r, id, g.store, bound.namespaces[i], now)
		if err != nil {
			g.log.WarnContext(r.Context(), fmt.Sprintf("abuse rule check failed, allowing: %s", err), slog.String("policy", bound.policy.Name))
			continue
		}
		result = g.pickStricterDecision(result, decision)
	}
	return result
}

func (g *Guard) escalatePenalty(ctx context.Context, id Identity, policy Policy, decision Decision, now time.Time) Decision {
	if id.IPKey == "" {
		return decision
	}
	key := g.createPenaltyKey(policy.Name, id.IPKey)
	penalty, err := g.store.LoadPenalty(key, now)
	if err != nil {
		g.log.WarnContext(ctx, fmt.Sprintf("abuse penalty load failed, allowing: %s", err), slog.String("policy", policy.Name))
		return decision
	}
	if now.Before(penalty.BlockedUntil) {
		return g.pickStricterDecision(decision, Decision{Verdict: VerdictBlock, Rule: penaltyRule, RetryAfter: penalty.BlockedUntil.Sub(now)})
	}
	if decision.Verdict != VerdictBlock {
		return decision
	}
	raised := g.raisePenalty(penalty, now)
	if !policy.DryRun {
		if saveErr := g.store.SavePenalty(key, raised, penaltyTTL, now); saveErr != nil {
			g.log.WarnContext(ctx, fmt.Sprintf("abuse penalty save failed: %s", saveErr), slog.String("policy", policy.Name))
		}
	}
	if raised.BlockedUntil.After(now) {
		return g.pickStricterDecision(decision, Decision{Verdict: VerdictBlock, Rule: penaltyRule, RetryAfter: raised.BlockedUntil.Sub(now)})
	}
	return decision
}

func (g *Guard) raisePenalty(penalty Penalty, now time.Time) Penalty {
	if !penalty.RaisedAt.IsZero() && now.Sub(penalty.RaisedAt) < penaltyRaiseInterval {
		return penalty
	}
	level := int(penalty.Level)
	if !penalty.RaisedAt.IsZero() {
		calmSince := penalty.RaisedAt
		if penalty.BlockedUntil.After(calmSince) {
			calmSince = penalty.BlockedUntil
		}
		level -= int(now.Sub(calmSince) / penaltyDecayInterval)
	}
	result := Penalty{Level: uint8(min(max(level, 0)+1, math.MaxUint8)), RaisedAt: now}
	switch {
	case result.Level >= penaltyLongLevel:
		result.BlockedUntil = now.Add(penaltyLongBlock)
	case result.Level >= penaltyShortLevel:
		result.BlockedUntil = now.Add(penaltyShortBlock)
	}
	return result
}

func (g *Guard) createPenaltyKey(policyName, ipKey string) Key {
	return Key{Namespace: penaltyNamespace + "|" + policyName, Value: ipKey}
}

func (g *Guard) pickStricterDecision(a, b Decision) Decision {
	if b.Verdict > a.Verdict {
		return b
	}
	if b.Verdict == a.Verdict && b.RetryAfter > a.RetryAfter {
		return b
	}
	return a
}

func (g *Guard) reportDecision(r *http.Request, id Identity, policy Policy, decision Decision, now time.Time) {
	if g.metrics != nil {
		g.metrics.CountDecision(policy.Name, decision.Rule, decision.Verdict, policy.DryRun)
	}
	g.log.WarnContext(
		r.Context(),
		"abuse decision",
		slog.String("policy", policy.Name),
		slog.String("rule", decision.Rule),
		slog.String("verdict", decision.Verdict.String()),
		slog.Bool("dry_run", policy.DryRun),
		slog.Bool("good_bot", id.GoodBot),
		slog.String("client_id", id.ClientID),
		slog.String("ip_hash", g.hasher.hashIP(id.IPKey, now)),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
	)
}

func (g *Guard) waitDelay(ctx context.Context, id Identity, now time.Time) bool {
	spread := maphash.String(g.seed, id.IPKey) ^ uint64(now.UnixNano())
	timer := time.NewTimer(minDelay + time.Duration(spread%uint64(delaySpread)))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (g *Guard) respondBlock(w http.ResponseWriter, r *http.Request, decision Decision) {
	retryAfter := decision.RetryAfter
	if retryAfter <= 0 {
		retryAfter = blockDefaultRetry
	}
	w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
	if g.block != nil {
		g.block.ServeHTTP(w, r)
		return
	}
	http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
}

func (g *Guard) isReadMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
}
