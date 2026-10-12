package validate

import (
	"regexp"
)

const (
	uuidv7Len = 36
)

var uuidV7Matcher = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-` +
		`[0-9a-fA-F]{4}-` +
		`7[0-9a-fA-F]{3}-` +
		`[89abAB][0-9a-fA-F]{3}-` +
		`[0-9a-fA-F]{12}$`,
)

var emailMatcher = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func IsEmail(v string) bool {
	if v == "" {
		return false
	}
	return emailMatcher.MatchString(v)
}

func IsUUIDv7(v string) bool {
	if v == "" {
		return false
	}
	return uuidV7Matcher.MatchString(v) && len(v) == uuidv7Len
}
