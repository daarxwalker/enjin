package localize

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

const pluralCountKey = "Count"

type Localizer struct {
	defaultLang language.Tag
	fsys        fs.FS
	bundle      *i18n.Bundle
	unmarshaler i18n.UnmarshalFunc
	translates  map[string]Translate
	matcher     language.Matcher
	supported   []language.Tag
	subpath     string
	extension   string
	langs       []language.Tag
}

type Translate func(message *i18n.Message, data ...map[string]any) string

func NewLocalizer(options ...Option) (*Localizer, error) {
	l := new(Localizer)
	for _, option := range options {
		option(l)
	}
	l.translates = make(map[string]Translate, len(l.langs))
	l.bundle = i18n.NewBundle(l.defaultLang)
	l.bundle.RegisterUnmarshalFunc(l.extension, l.unmarshaler)
	for _, lang := range l.langs {
		var path string
		if len(l.subpath) == 0 {
			path = fmt.Sprintf("active.%s.%s", lang, l.extension)
		} else {
			path = fmt.Sprintf("%s/active.%s.%s", l.subpath, lang, l.extension)
		}
		if _, loadErr := l.bundle.LoadMessageFileFS(
			l.fsys, path,
		); loadErr != nil {
			return nil, fmt.Errorf("load %s %s failed: %w", lang, l.subpath, loadErr)
		}
	}
	l.supported = make([]language.Tag, 0, len(l.langs)+1)
	l.supported = append(l.supported, l.defaultLang)
	for _, lang := range l.langs {
		localizer := i18n.NewLocalizer(l.bundle, lang.String())
		l.translates[lang.String()] = l.createTranslate(localizer)
		if lang != l.defaultLang {
			l.supported = append(l.supported, lang)
		}
	}
	l.matcher = language.NewMatcher(l.supported)
	return l, nil
}

func (l *Localizer) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			langCode := l.getLang(r)
			ctx := r.Context()
			ctx = context.WithValue(ctx, langKey, langCode)
			ctx = l.WithTranslate(ctx, l.GetTranslate(langCode))
			next.ServeHTTP(w, r.WithContext(ctx))
		},
	)
}

func (l *Localizer) GetTranslate(lang string) Translate {
	t, ok := l.translates[lang]
	if !ok {
		return l.translates[l.defaultLang.String()]
	}
	return t
}

func (l *Localizer) WithTranslate(ctx context.Context, translate Translate) context.Context {
	return context.WithValue(ctx, localizerKey, translate)
}

func (l *Localizer) getLang(r *http.Request) string {
	_, index := language.MatchStrings(l.matcher, r.URL.Query().Get("lang"), r.Header.Get("Accept-Language"))
	return l.supported[index].String()
}

func (l *Localizer) createTranslate(localizer *i18n.Localizer) Translate {
	return func(message *i18n.Message, data ...map[string]any) string {
		var templateData map[string]any
		if len(data) > 0 {
			templateData = data[0]
		}
		result, err := localizer.Localize(
			&i18n.LocalizeConfig{
				DefaultMessage: message,
				TemplateData:   templateData,
				PluralCount:    l.findPluralCount(message, templateData),
			},
		)
		if err != nil {
			return "<" + message.ID + ">"
		}
		return result
	}
}

func (l *Localizer) findPluralCount(message *i18n.Message, data map[string]any) any {
	if message.Zero == "" && message.One == "" && message.Two == "" && message.Few == "" && message.Many == "" {
		return nil
	}
	return data[pluralCountKey]
}

type Option func(*Localizer)

func WithFS(fsys fs.FS) Option {
	return func(t *Localizer) {
		t.fsys = fsys
	}
}

func WithDefaultLang(defaultLang language.Tag) Option {
	return func(t *Localizer) {
		t.defaultLang = defaultLang
	}
}

func WithLangs(langs ...language.Tag) Option {
	return func(t *Localizer) {
		t.langs = langs
	}
}

func WithSubpath(subpath string) Option {
	return func(t *Localizer) {
		t.subpath = subpath
	}
}

func WithExtension(extension string, unmarshaler i18n.UnmarshalFunc) Option {
	return func(t *Localizer) {
		t.extension = extension
		t.unmarshaler = unmarshaler
	}
}
