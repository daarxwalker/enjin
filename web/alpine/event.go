package alpine

import (
	"strings"
)

const (
	Prevent  = "prevent"
	Stop     = "stop"
	Outside  = "outside"
	Window   = "window"
	Document = "document"
	Once     = "once"
	Debounce = "debounce"
	Throttle = "throttle"
	Self     = "self"
	Camel    = "camel"
	Dot      = "dot"
	Passive  = "passive"
	Capture  = "capture"
)

const (
	Shift    = "shift"
	Enter    = "enter"
	Space    = "space"
	Ctrl     = "ctrl"
	Cmd      = "cmd"
	Meta     = "meta"
	Alt      = "alt"
	Up       = "up"
	Right    = "right"
	Down     = "down"
	Left     = "left"
	Escape   = "escape"
	Tab      = "tab"
	CapsLock = "caps-lock"
	Equal    = "equal"
	Period   = "period"
	Comma    = "comma"
	Slash    = "slash"
)

const (
	dot = "."
)

func Click(modifiers ...string) string {
	return CreateEvent("click", modifiers...)
}

func DragOver(modifiers ...string) string {
	return CreateEvent("dragover", modifiers...)
}

func DragLeave(modifiers ...string) string {
	return CreateEvent("dragleave", modifiers...)
}

func Drop(modifiers ...string) string {
	return CreateEvent("drop", modifiers...)
}

func Change(modifiers ...string) string {
	return CreateEvent("change", modifiers...)
}

func Blur(modifiers ...string) string {
	return CreateEvent("blur", modifiers...)
}

func Input(modifiers ...string) string {
	return CreateEvent("input", modifiers...)
}

func Submit(modifiers ...string) string {
	return CreateEvent("submit", modifiers...)
}

func Scroll(modifiers ...string) string {
	return CreateEvent("scroll", modifiers...)
}

func KeyUp(modifiers ...string) string {
	return CreateEvent("keyup", modifiers...)
}

func KeyDown(modifiers ...string) string {
	return CreateEvent("keydown", modifiers...)
}

func MouseMove(modifiers ...string) string {
	return CreateEvent("mousemove", modifiers...)
}

func MouseEnter(modifiers ...string) string {
	return CreateEvent("mouseenter", modifiers...)
}

func MouseLeave(modifiers ...string) string {
	return CreateEvent("mouseleave", modifiers...)
}

func CreateEvent(name string, modifiers ...string) string {
	var b strings.Builder
	b.WriteString("@")
	b.WriteString(name)
	if len(modifiers) > 0 {
		b.WriteString(dot)
		b.WriteString(strings.Join(modifiers, dot))
	}
	return b.String()
}
