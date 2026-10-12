package mail

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestRenderKeepsEscapedText(t *testing.T) {
	userText := `<a href="https://evil.example">klikni</a> & <b>`
	component := templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, `<mjml><mj-body><mj-section><mj-column><mj-text>`+
			templ.EscapeString(userText)+
			`</mj-text></mj-column></mj-section></mj-body></mjml>`)
		return err
	})

	html, err := Render(context.Background(), component)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if strings.Contains(html, "<a href=") || strings.Contains(html, "<b>") {
		t.Errorf("escaped text was turned back into markup: %s", html)
	}
	if !strings.Contains(html, "&#60;a href=&#34;https://evil.example&#34;&#62;klikni&#60;/a&#62; &amp; &#60;b&#62;") {
		t.Errorf("escaped text is missing in output: %s", html)
	}
}
