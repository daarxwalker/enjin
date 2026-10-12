package head

type Option func(p *ContextValue)

func WithTitle(title string) Option {
	return func(p *ContextValue) {
		p.Title = title
	}
}

func WithDescription(content string) Option {
	return WithMeta("description", content)
}

func WithMeta(name, content string) Option {
	return func(p *ContextValue) {
		p.Metas = append(
			p.Metas, Meta{
				Name:    name,
				Content: content,
			},
		)
	}
}

func WithLink(link Link) Option {
	return func(p *ContextValue) {
		p.Links = append(
			p.Links, link,
		)
	}
}

func WithScript(script Script) Option {
	return func(p *ContextValue) {
		p.Scripts = append(
			p.Scripts, script,
		)
	}
}
