package alpine

import (
	"encoding/json"
	"fmt"
	"strings"
)

var jsStringReplacer = strings.NewReplacer(
	`\`, `\\`,
	`'`, `\'`,
	"\n", `\n`,
	"\r", `\r`,
	"\u2028", `\u2028`,
	"\u2029", `\u2029`,
	"<", `\u003c`,
)

func jsString(s string) string {
	return "'" + jsStringReplacer.Replace(s) + "'"
}

func Map(value map[string]any) string {
	if len(value) == 0 {
		return "{}"
	}
	b, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Errorf("alpine: marshal map failed: %w", err))
	}
	return string(b)
}

func Array[T any](a []T) string {
	if len(a) == 0 {
		return "[]"
	}
	b, err := json.Marshal(a)
	if err != nil {
		panic(fmt.Errorf("alpine: marshal array failed: %w", err))
	}
	return string(b)
}
