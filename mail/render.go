package mail

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/a-h/templ"
	"github.com/preslavrachev/gomjml/mjml"
)

// gomjml decodes &lt; and &gt; inside mj-text, which would turn text escaped by templ
// back into markup. Numeric references survive the round trip.
var specialCharProtector = strings.NewReplacer("&lt;", "&#60;", "&gt;", "&#62;")

func Render(ctx context.Context, component templ.Component) (string, error) {
	var buf bytes.Buffer
	if renderErr := component.Render(ctx, &buf); renderErr != nil {
		return "", fmt.Errorf("render mail templ failed: %w", renderErr)
	}
	content, mjmlErr := mjml.Render(specialCharProtector.Replace(buf.String()))
	if mjmlErr != nil {
		return "", fmt.Errorf("render mail mjml failed: %w", mjmlErr)
	}
	return content, nil
}
