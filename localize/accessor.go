package localize

import (
	"context"
)

func TranslateFrom(ctx context.Context) Translate {
	val := ctx.Value(localizerKey)
	if val == nil {
		return nil
	}
	t, ok := val.(Translate)
	if !ok {
		return nil
	}
	return t
}

func LangFrom(ctx context.Context) string {
	val := ctx.Value(langKey)
	if val == nil {
		return ""
	}
	lang, ok := val.(string)
	if !ok {
		return ""
	}
	return lang
}
