package abuse

import (
	"net/netip"
	"time"
)

type Verdict uint8

const (
	VerdictAllow Verdict = iota
	VerdictDelay
	VerdictBlock
)

const RulePenalty = "penalty"

type Decision struct {
	Rule       string
	RetryAfter time.Duration
	Verdict    Verdict
}

func (v Verdict) String() string {
	switch v {
	case VerdictAllow:
		return "allow"
	case VerdictDelay:
		return "delay"
	case VerdictBlock:
		return "block"
	}
	return "unknown"
}

type Event struct {
	Time       time.Time
	Addr       netip.Addr
	IP         netip.Prefix
	Policy     string
	Rule       string
	ClientID   string
	UserID     string
	Method     string
	Path       string
	UserAgent  string
	RetryAfter time.Duration
	Verdict    Verdict
	DryRun     bool
	GoodBot    bool
}
