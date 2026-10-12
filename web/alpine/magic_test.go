package alpine_test

import (
	"testing"

	"github.com/daarxwalker/enjin/web/alpine"
)

func TestMagic(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"id", alpine.Magic.ID("menu"), `$id('menu')`},
		{"id escapes", alpine.Magic.ID(`a'); alert(1); ('`), `$id('a\'); alert(1); (\'')`},
		{"watch", alpine.Magic.Watch("open", "x()"), `$watch('open', () => x())`},
		{"watch escapes", alpine.Magic.Watch(`a\'`, "x()"), `$watch('a\\\'', () => x())`},
		{"dispatch expression", alpine.Magic.Dispatch("change", "value"), `$dispatch('change', value)`},
		{"dispatch data", alpine.Magic.Dispatch("change", map[string]int{"a": 1}), `$dispatch('change', {"a":1})`},
		{"dispatch escapes", alpine.Magic.Dispatch("a'\n</script>", "{}"), `$dispatch('a\'\n\u003c/script>', {})`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}
