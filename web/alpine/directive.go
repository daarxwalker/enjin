package alpine

import "strings"

const (
	Data      = "x-data"
	Init      = "x-init"
	Show      = "x-show"
	Text      = "x-text"
	HTML      = "x-html"
	Model     = "x-model"
	Modelable = "x-modelable"
	For       = "x-for"
	Effect    = "x-effect"
	Ignore    = "x-ignore"
	Ref       = "x-ref"
	Cloak     = "x-cloak"
	Teleport  = "x-teleport"
	If        = "x-if"
	ID        = "x-id"
)

func On(event string) string {
	return "x-on:" + event
}

func Bind(attr string) string {
	return ":" + attr
}

func Transition(modifiers ...string) string {
	var b strings.Builder
	b.WriteString("x-transition")
	n := len(modifiers)
	for i, m := range modifiers {
		if i == 0 {
			b.WriteString(":")
		}
		b.WriteString(m)
		if i < n-1 {
			b.WriteString(".")
		}
	}
	return b.String()
}
