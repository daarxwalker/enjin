package validate

import (
	"strconv"
	"strings"
)

type Verdict struct {
	Reason string `json:"reason"`
	Value  string `json:"value"`
}

var Required = Verdict{Reason: "required"}

func MinLen(v int) Verdict {
	return Verdict{Reason: "minLen", Value: strconv.Itoa(v)}
}

func MaxLen(v int) Verdict {
	return Verdict{Reason: "maxLen", Value: strconv.Itoa(v)}
}

func Min(v int) Verdict {
	return Verdict{Reason: "min", Value: strconv.Itoa(v)}
}

func Max(v int) Verdict {
	return Verdict{Reason: "max", Value: strconv.Itoa(v)}
}

func Invalid(allowed ...string) Verdict {
	return Verdict{Reason: "invalid", Value: strings.Join(allowed, "|")}
}
