package htmx

import (
	"encoding/json"
	"fmt"
	"strings"
)

func Mod(v string, mods ...string) string {
	if len(mods) == 0 {
		return v
	}
	var b strings.Builder
	modsN := len(mods)
	b.Grow(len(v) + modsN + (modsN * 8))
	b.WriteString(v)
	for _, mod := range mods {
		b.WriteString(":")
		b.WriteString(mod)
	}
	return b.String()
}

func WithMod(v string, mods ...string) string {
	return Mod(v, mods...)
}

func Modifier(key, value string) string {
	return key + ":" + value
}

func Value(base string, mods ...string) string {
	if len(mods) == 0 {
		return base
	}
	if base == "" {
		return Modifiers(mods...)
	}
	return base + " " + Modifiers(mods...)
}

func Modifiers(mods ...string) string {
	return strings.Join(mods, " ")
}

func QuoteSelector(selector string) string {
	if selector == "" {
		return selector
	}
	if strings.HasPrefix(selector, "'") && strings.HasSuffix(selector, "'") {
		return selector
	}
	if !strings.ContainsAny(selector, " ,") {
		return selector
	}
	return "'" + selector + "'"
}

func Map(value map[string]any) string {
	if len(value) == 0 {
		return "{}"
	}
	b, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Errorf("htmx: marshal map failed: %w", err))
	}
	return string(b)
}

func JSMap(value map[string]any) string {
	return "js:" + Map(value)
}

func joinComma(values []string) string {
	return strings.Join(values, ",")
}
