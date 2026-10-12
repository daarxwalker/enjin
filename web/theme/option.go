package theme

import "github.com/daarxwalker/enjin/cookie"

type Option func(*Manager)

func WithDefaultTheme(theme Value) Option {
	return func(m *Manager) {
		if value, ok := parseValue(string(theme)); ok {
			m.defaultTheme = value
		}
	}
}

func WithConfig(options ...cookie.Option) Option {
	return func(m *Manager) {
		m.cookie.Apply(options...)
	}
}
