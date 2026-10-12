package abuse

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

var automatedUserAgentMarkers = []string{
	"curl",
	"wget",
	"python-requests",
	"python-urllib",
	"aiohttp",
	"httpx",
	"go-http-client",
	"headlesschrome",
	"phantomjs",
	"puppeteer",
	"playwright",
	"selenium",
	"axios",
	"node-fetch",
	"undici",
	"scrapy",
	"httpclient",
	"okhttp",
	"java/",
	"libwww-perl",
}

type Rule interface {
	Name() string
	CheckRequest(r *http.Request, id Identity, store Store, namespace string, now time.Time) (Decision, error)
}

type rateLimitRule struct {
	name      string
	countedBy CountedBy
	limit     int
	per       time.Duration
	then      Verdict
}

type distinctURLParamRule struct {
	urlParam  string
	name      string
	countedBy CountedBy
	limit     int
	per       time.Duration
	then      Verdict
}

type userAgentRule struct {
	name string
	then Verdict
}

var (
	_ Rule = rateLimitRule{}
	_ Rule = distinctURLParamRule{}
	_ Rule = userAgentRule{}
)

func CreateRateLimitRule(countedBy CountedBy, limit int, per time.Duration, then Verdict) Rule {
	return rateLimitRule{
		countedBy: countedBy,
		name:      fmt.Sprintf("rate-%s-%d-%s", countedBy.name, limit, per),
		limit:     limit,
		per:       per,
		then:      then,
	}
}

func CreateDistinctURLParamRule(countedBy CountedBy, urlParam string, limit int, per time.Duration, then Verdict) Rule {
	return distinctURLParamRule{
		countedBy: countedBy,
		urlParam:  urlParam,
		name:      fmt.Sprintf("distinct-%s-%s-%d-%s", urlParam, countedBy.name, limit, per),
		limit:     limit,
		per:       per,
		then:      then,
	}
}

func CreateUserAgentRule(then Verdict) Rule {
	return userAgentRule{name: "user-agent", then: then}
}

func (r rateLimitRule) Name() string {
	return r.name
}

func (r rateLimitRule) CheckRequest(req *http.Request, id Identity, store Store, namespace string, now time.Time) (Decision, error) {
	value := r.countedBy.readValue(req, id)
	if value == "" {
		return Decision{}, nil
	}
	count, err := store.HitCounter(Key{Namespace: namespace, Value: value}, r.per, now)
	if err != nil {
		return Decision{}, fmt.Errorf("hit %s failed: %w", r.name, err)
	}
	if count <= r.limit {
		return Decision{}, nil
	}
	return Decision{Verdict: r.then, Rule: r.name, RetryAfter: r.per}, nil
}

func (r distinctURLParamRule) Name() string {
	return r.name
}

func (r distinctURLParamRule) CheckRequest(req *http.Request, id Identity, store Store, namespace string, now time.Time) (Decision, error) {
	value := r.countedBy.readValue(req, id)
	if value == "" {
		return Decision{}, nil
	}
	member := chi.URLParam(req, r.urlParam)
	if member == "" {
		return Decision{}, nil
	}
	count, err := store.CountDistinct(Key{Namespace: namespace, Value: value}, member, r.limit, r.per, now)
	if err != nil {
		return Decision{}, fmt.Errorf("distinct %s failed: %w", r.name, err)
	}
	if count <= r.limit {
		return Decision{}, nil
	}
	return Decision{Verdict: r.then, Rule: r.name, RetryAfter: r.per}, nil
}

func (r userAgentRule) Name() string {
	return r.name
}

func (r userAgentRule) CheckRequest(req *http.Request, _ Identity, _ Store, _ string, _ time.Time) (Decision, error) {
	if !r.isAutomated(req.UserAgent()) {
		return Decision{}, nil
	}
	return Decision{Verdict: r.then, Rule: r.name}, nil
}

func (r userAgentRule) isAutomated(userAgent string) bool {
	userAgent = strings.ToLower(strings.TrimSpace(userAgent))
	if userAgent == "" {
		return true
	}
	return slices.ContainsFunc(automatedUserAgentMarkers, func(marker string) bool {
		return strings.Contains(userAgent, marker)
	})
}
