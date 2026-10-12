package redirect

import "github.com/daarxwalker/enjin/cookie"

type Option func(*Manager)

func WithConfig(options ...cookie.Option) Option {
	return func(m *Manager) {
		m.cookie.Apply(options...)
	}
}
