package abuse

import (
	"log/slog"
	"net/http"
)

type Option func(*Guard)

type MemoryStoreOption func(*MemoryStore)

type BotVerifierOption func(*BotVerifier)

func WithClock(clock Clock) Option {
	return func(g *Guard) {
		g.clock = clock
	}
}

func WithLogger(log *slog.Logger) Option {
	return func(g *Guard) {
		g.log = log
	}
}

func WithMetrics(metrics Metrics) Option {
	return func(g *Guard) {
		g.metrics = metrics
	}
}

func WithRecorder(recorder Recorder) Option {
	return func(g *Guard) {
		g.recorder = recorder
	}
}

func WithBlockHandler(handler http.Handler) Option {
	return func(g *Guard) {
		g.block = handler
	}
}

func WithBotVerifier(verifier *BotVerifier) Option {
	return func(g *Guard) {
		g.bots = verifier
	}
}

func WithGoodBotPolicy(policy Policy) Option {
	return func(g *Guard) {
		g.goodBotPolicy = policy
	}
}

func WithMemoryCapacity(counters, sets int) MemoryStoreOption {
	return func(s *MemoryStore) {
		s.maxCounters = counters
		s.maxSets = sets
	}
}

func WithResolver(resolver resolver) BotVerifierOption {
	return func(v *BotVerifier) {
		v.resolver = resolver
	}
}
