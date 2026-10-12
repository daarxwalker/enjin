package validate

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"
)

type Result map[string][]Verdict

func (r Result) Zero() bool {
	return len(r) == 0
}

func (r Result) Has(key string) bool {
	_, ok := r[key]
	return ok
}

func (r Result) Get(key string) []Verdict {
	v, ok := r[key]
	if !ok {
		return []Verdict{}
	}
	return v
}

func (r Result) Text(key string, value string, minLen, maxLen int, allowed ...string) {
	r.Add(key, Text(value, minLen, maxLen, allowed...)...)
}

func (r Result) Int(key string, value int, minValue, maxValue int) {
	r.Add(key, Int(value, minValue, maxValue)...)
}

func (r Result) Float64(key string, value float64, minValue, maxValue int) {
	r.Add(key, Float64(value, minValue, maxValue)...)
}

func (r Result) UUIDv7(key string, value string) {
	r.Add(key, UUIDv7(value)...)
}

func (r Result) Email(key string, value string) {
	r.Add(key, Email(value)...)
}

func (r Result) Add(key string, verdicts ...Verdict) {
	if len(verdicts) == 0 {
		return
	}
	r[key] = append(r[key], verdicts...)
}

func Compose(verdicts ...[]Verdict) []Verdict {
	result := make([]Verdict, 0, 2*len(verdicts))
	for _, v := range verdicts {
		result = append(result, v...)
	}
	return result
}

func Text(value string, minLen, maxLen int, allowed ...string) []Verdict {
	var v []Verdict
	if value == "" {
		v = append(v, Required)
	}
	if value != "" && utf8.RuneCountInString(value) < minLen {
		v = append(v, MinLen(minLen))
	}
	if value != "" && utf8.RuneCountInString(value) > maxLen {
		v = append(v, MaxLen(maxLen))
	}
	if value != "" && len(allowed) > 0 && !slices.Contains(allowed, value) {
		v = append(v, Invalid(allowed...))
	}
	return v
}

func Int(value int, minValue, maxValue int) []Verdict {
	var v []Verdict
	if value == 0 {
		v = append(v, Required)
	}
	if value != 0 && value < minValue {
		v = append(v, Min(minValue))
	}
	if value != 0 && value > maxValue {
		v = append(v, Max(maxValue))
	}
	return v
}

func Float64(value float64, minValue, maxValue int) []Verdict {
	var v []Verdict
	if value == 0 {
		v = append(v, Required)
	}
	if value != 0 && value < float64(minValue) {
		v = append(v, Min(minValue))
	}
	if value != 0 && value > float64(maxValue) {
		v = append(v, Max(maxValue))
	}
	return v
}

func UUIDv7(value string) []Verdict {
	var v []Verdict
	if value != "" && !IsUUIDv7(value) {
		v = append(v, Invalid("uuidv7"))
	}
	return v
}

func Email(value string) []Verdict {
	var v []Verdict
	if value != "" && !emailMatcher.MatchString(value) {
		v = append(v, Invalid("email"))
	}
	return v
}

// TODO: Pretty Localized String
func (r Result) String() string {
	if len(r) == 0 {
		return ""
	}
	var builder strings.Builder
	for _, k := range slices.Sorted(maps.Keys(r)) {
		verdicts := r[k]
		builder.WriteString(k)
		builder.WriteString(":")
		for i, verdict := range verdicts {
			if i > 0 {
				builder.WriteString(",")
			}
			builder.WriteString(verdict.Reason)
			if verdict.Value != "" {
				builder.WriteString("(")
				builder.WriteString(verdict.Value)
				builder.WriteString(")")
			}
		}
		builder.WriteString("\n")
	}
	return builder.String()
}

func (r Result) Error() error {
	return errors.New(r.String())
}
