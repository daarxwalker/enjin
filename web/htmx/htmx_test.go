package htmx_test

import (
	"context"
	"strings"
	"testing"

	"github.com/daarxwalker/enjin/web/htmx"

	"github.com/a-h/templ"
)

func TestHTMX(t *testing.T) {
	t.Run("mod", func(t *testing.T) {
		tests := []struct {
			name string
			got  string
			want string
		}{
			{"no modifier", htmx.Target(), "hx-target"},
			{"inherited", htmx.Target(htmx.ModInherited), "hx-target:inherited"},
			{"append", htmx.Vals(htmx.ModAppend), "hx-vals:append"},
			{"chained", htmx.Vals(htmx.ModInherited, htmx.ModAppend), "hx-vals:inherited:append"},
			{"constant attribute", htmx.Mod(htmx.Trigger, htmx.ModInherited), "hx-trigger:inherited"},
			{"confirm", htmx.Confirm(htmx.ModInherited), "hx-confirm:inherited"},
			{"config", htmx.Config(), "hx-config"},
			{"disable", htmx.Disable(), "hx-disable"},
			{"on event", htmx.On(htmx.Click), "hx-on:click"},
			{"status", htmx.Status("422"), "hx-status:422"},
			{"status wildcard", htmx.Status(htmx.Status5xx), "hx-status:5xx"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.got != tt.want {
					t.Errorf("got %q, want %q", tt.got, tt.want)
				}
			})
		}
	})

	t.Run("value", func(t *testing.T) {
		tests := []struct {
			name string
			got  string
			want string
		}{
			{
				name: "bare style",
				got:  htmx.Value(htmx.SwapReplace),
				want: "outerHTML",
			},
			{
				name: "separate show keys",
				got: htmx.Value(
					htmx.SwapReplaceContent,
					htmx.Modifier(htmx.ModShow, htmx.PositionTop),
					htmx.Modifier(htmx.ModShowTarget, "#other"),
				),
				want: "innerHTML show:top showTarget:#other",
			},
			{
				name: "settle delay",
				got:  htmx.Value(htmx.SwapMorph, htmx.Modifier(htmx.ModSettle, "200ms")),
				want: "innerMorph settle:200ms",
			},
			{
				name: "modifiers only",
				got: htmx.Modifiers(
					htmx.Modifier(htmx.StatusSwap, htmx.SwapReplaceContent),
					htmx.Modifier(htmx.StatusTarget, "#errors"),
				),
				want: "swap:innerHTML target:#errors",
			},
			{
				name: "empty base does not leak a space",
				got:  htmx.Value("", htmx.Modifier(htmx.StatusPush, "false")),
				want: "push:false",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.got != tt.want {
					t.Errorf("got %q, want %q", tt.got, tt.want)
				}
			})
		}
	})

	t.Run("quote selector", func(t *testing.T) {
		tests := []struct {
			name string
			in   string
			want string
		}{
			{"simple", "#target", "#target"},
			{"space", "closest form", "'closest form'"},
			{"comma", ".a,.b", "'.a,.b'"},
			{"comma and space", ".a, .b", "'.a, .b'"},
			{"already quoted", "'closest form'", "'closest form'"},
			{"empty", "", ""},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := htmx.QuoteSelector(tt.in); got != tt.want {
					t.Errorf("QuoteSelector(%q) = %q, want %q", tt.in, got, tt.want)
				}
			})
		}
	})

	t.Run("trigger and sync", func(t *testing.T) {
		tests := []struct {
			name string
			got  string
			want string
		}{
			{"from quotes selector", htmx.From("closest form"), "from:'closest form'"},
			{"from leaves simple selector", htmx.From("body"), "from:body"},
			{"trigger for", htmx.TriggerFor(".a, .b"), "target:'.a, .b'"},
			{
				name: "trigger spec",
				got:  htmx.Value(htmx.KeyUp, htmx.Modifier(htmx.TriggerDelay, "500ms"), htmx.From("closest form")),
				want: "keyup delay:500ms from:'closest form'",
			},
			{"sync spec", htmx.SyncSpec(htmx.TargetThis, htmx.SyncQueueAll), "this:queue all"},
			{"sync spec quotes", htmx.SyncSpec("closest form", htmx.SyncDrop), "'closest form':drop"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.got != tt.want {
					t.Errorf("got %q, want %q", tt.got, tt.want)
				}
			})
		}
	})

	t.Run("config meta", func(t *testing.T) {
		var b strings.Builder
		err := htmx.ConfigMeta(map[string]any{htmx.ConfigDefaultTimeout: 0}).Render(context.Background(), &b)
		if err != nil {
			t.Fatalf("render failed: %v", err)
		}
		want := `<meta name="htmx-config" content='{"defaultTimeout":0}'/>`
		if got := b.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("config meta escapes quotes", func(t *testing.T) {
		var b strings.Builder
		err := htmx.ConfigMeta(map[string]any{htmx.ConfigPrefix: `x'/><script>`}).Render(context.Background(), &b)
		if err != nil {
			t.Fatalf("render failed: %v", err)
		}
		want := `<meta name="htmx-config" content='{"prefix":"x&#39;/\u003e\u003cscript\u003e"}'/>`
		if got := b.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("extensions", func(t *testing.T) {
		got := htmx.Extensions(htmx.SSEExt, htmx.AlpineCompatExt)
		if want := "sse, alpine-compat"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("partial", func(t *testing.T) {
		tests := []struct {
			name  string
			props htmx.PartialProps
			want  string
		}{
			{
				name:  "id shorthand",
				props: htmx.PartialProps{ID: "count"},
				want:  `<hx-partial hx-target="#count">ok</hx-partial>`,
			},
			{
				name:  "target and swap",
				props: htmx.PartialProps{Target: "closest .card", Swap: htmx.SwapAppend},
				want:  `<hx-partial hx-target="closest .card" hx-swap="append">ok</hx-partial>`,
			},
			{
				name:  "target wins over id",
				props: htmx.PartialProps{ID: "count", Target: "#total"},
				want:  `<hx-partial hx-target="#total">ok</hx-partial>`,
			},
			{
				name:  "no target",
				props: htmx.PartialProps{},
				want:  `<hx-partial>ok</hx-partial>`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var b strings.Builder
				ctx := templ.WithChildren(context.Background(), templ.Raw("ok"))
				if err := htmx.Partial(tt.props).Render(ctx, &b); err != nil {
					t.Fatalf("render failed: %v", err)
				}
				if got := b.String(); got != tt.want {
					t.Errorf("got %q, want %q", got, tt.want)
				}
			})
		}
	})
}
