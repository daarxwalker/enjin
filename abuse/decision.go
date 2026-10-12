package abuse

import "time"

type Verdict uint8

const (
	VerdictAllow Verdict = iota
	VerdictDelay
	VerdictBlock
)

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
