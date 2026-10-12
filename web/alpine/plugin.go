package alpine

import (
	"strconv"
	"strings"
)

// --------------------------------------------------------

func Mask(dynamic ...bool) string {
	attr := "x-mask"
	if len(dynamic) > 0 {
		attr += ":dynamic"
	}
	return attr
}

// Intersect
const (
	IntersectEnter = "enter"
	IntersectLeave = "leave"
)

const (
	IntersectOnce      = "once"
	IntersectHalf      = "half"
	IntersectFull      = "full"
	IntersectThreshold = "threshold"
	IntersectMargin    = "margin"
)

// --------------------------------------------------------

func Intersect(modifiers ...string) string {
	var b strings.Builder
	b.WriteString("x-intersect")
	for _, m := range modifiers {
		if m == IntersectEnter || m == IntersectLeave {
			b.WriteString(":")
		} else {
			b.WriteString(".")
		}
		b.WriteString(m)
	}
	return b.String()
}

// --------------------------------------------------------

const (
	ResizeDocument = "document"
	ResizeWindow   = "window"
)

func Resize(modifiers ...string) string {
	var b strings.Builder
	b.WriteString("x-resize")
	for _, m := range modifiers {
		b.WriteString(".")
		b.WriteString(m)
	}
	return b.String()
}

// --------------------------------------------------------

func Persist(modifiers ...string) string {
	var b strings.Builder
	b.WriteString("$persist")
	for _, m := range modifiers {
		b.WriteString(".")
		b.WriteString(m)
	}
	return b.String()
}

func PersistAs(value string) string {
	return "as(" + value + ")"
}

func PersistUsing(value string) string {
	return "using(" + value + ")"
}

// --------------------------------------------------------

const (
	TrapInert       = "inert"
	TrapNoScroll    = "noscroll"
	TrapNoReturn    = "noreturn"
	TrapNoAutoFocus = "noautofocus"
)

func Trap(modifiers ...string) string {
	var b strings.Builder
	b.WriteString("x-trap")
	for _, m := range modifiers {
		b.WriteString(".")
		b.WriteString(m)
	}
	return b.String()
}

type FocusMagic string

var Focus = FocusMagic("$focus")

func (f FocusMagic) Focus(el string) FocusMagic {
	f += FocusMagic(".focus(" + el + ")")
	return f
}

func (f FocusMagic) Focusable(el string) FocusMagic {
	f += FocusMagic(".focusable(" + el + ")")
	return f
}

func (f FocusMagic) Focusables() FocusMagic {
	f += FocusMagic(".focusables()")
	return f
}

func (f FocusMagic) LastFocused() FocusMagic {
	f += FocusMagic(".lastFocused()")
	return f
}

func (f FocusMagic) Within(el string) FocusMagic {
	f += FocusMagic(".within(" + el + ")")
	return f
}

func (f FocusMagic) First() FocusMagic {
	f += FocusMagic(".first()")
	return f
}

func (f FocusMagic) Last() FocusMagic {
	f += FocusMagic(".last()")
	return f
}

func (f FocusMagic) Next() FocusMagic {
	f += FocusMagic(".next()")
	return f
}

func (f FocusMagic) Previous() FocusMagic {
	f += FocusMagic(".previous()")
	return f
}

func (f FocusMagic) NoScroll() FocusMagic {
	f += FocusMagic(".noscroll()")
	return f
}

func (f FocusMagic) Wrap() FocusMagic {
	f += FocusMagic(".wrap()")
	return f
}

func (f FocusMagic) GetFirst() FocusMagic {
	f += FocusMagic(".getFirst()")
	return f
}

func (f FocusMagic) GetLast() FocusMagic {
	f += FocusMagic(".getLast()")
	return f
}

func (f FocusMagic) GetNext() FocusMagic {
	f += FocusMagic(".getNext()")
	return f
}

func (f FocusMagic) GetPrevious() FocusMagic {
	f += FocusMagic(".getPrevious()")
	return f
}

// --------------------------------------------------------

const (
	CollapseMin      = "min"
	CollapseDuration = "duration"
)

func Collapse(modifiers ...string) string {
	var b strings.Builder
	b.WriteString("x-collapse")
	for _, m := range modifiers {
		b.WriteString(".")
		b.WriteString(m)
	}
	return b.String()
}

// --------------------------------------------------------

const (
	AnchorBottom      = "bottom"
	AnchorBottomStart = "bottom-start"
	AnchorBottomEnd   = "bottom-end"
	AnchorTop         = "top"
	AnchorTopStart    = "top-start"
	AnchorTopEnd      = "top-end"
	AnchorLeft        = "left"
	AnchorLeftStart   = "left-start"
	AnchorLeftEnd     = "left-end"
	AnchorRight       = "right"
	AnchorRightStart  = "right-start"
	AnchorRightEnd    = "right-end"
)

const (
	AnchorFixed   = "fixed"
	AnchorNoFlip  = "noflip"
	AnchorNoStyle = "no-style"
)

func Anchor(modifiers ...string) string {
	var b strings.Builder
	b.WriteString("x-anchor")
	for _, m := range modifiers {
		b.WriteString(".")
		b.WriteString(m)
	}
	return b.String()
}

func AnchorOffset(offset int) string {
	return "offset." + strconv.Itoa(offset)
}

// --------------------------------------------------------

const (
	Sort       = "x-sort"
	SortGroup  = "x-sort:group"
	SortItem   = "x-sort:item"
	SortHandle = "x-sort:handle"
	SortIgnore = "x-sort:ignore"
	SortGhost  = "x-sort.ghost"
	SortConfig = "x-sort:config"
)
