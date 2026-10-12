package abuse

import (
	"context"
	"errors"
	"net"
	"time"
)

var ErrInvalidWindow = errors.New("abuse: window must be positive")

type Key struct {
	Namespace string
	Value     string
}

type Penalty struct {
	RaisedAt     time.Time
	BlockedUntil time.Time
	Level        uint8
}

type Store interface {
	HitCounter(key Key, window time.Duration, now time.Time) (int, error)
	CountDistinct(key Key, member string, limit int, window time.Duration, now time.Time) (int, error)
	LoadPenalty(key Key, now time.Time) (Penalty, error)
	SavePenalty(key Key, penalty Penalty, ttl time.Duration, now time.Time) error
	LoadMark(key Key, now time.Time) (bool, bool, error)
	SaveMark(key Key, value bool, ttl time.Duration, now time.Time) error
}

type Metrics interface {
	CountDecision(policy, rule string, verdict Verdict, dryRun bool)
}

type Clock interface {
	Now() time.Time
}

type resolver interface {
	LookupAddr(ctx context.Context, addr string) ([]string, error)
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

type touchable interface {
	touchedAt() int64
	expiredAt(now int64) bool
}

type systemClock struct{}

var (
	_ Store    = (*MemoryStore)(nil)
	_ Clock    = systemClock{}
	_ resolver = (*net.Resolver)(nil)
)

func (systemClock) Now() time.Time {
	return time.Now()
}
