package routespec

import "strings"

type Option func(*Operation)

func WithID(id string) Option {
	return func(op *Operation) {
		op.ID = id
	}
}

func WithSummary(summary string) Option {
	return func(op *Operation) {
		op.Summary = summary
	}
}

func WithDescription(description string) Option {
	return func(op *Operation) {
		op.Description = description
	}
}

func WithTags(tags ...string) Option {
	return func(op *Operation) {
		op.Tags = append(op.Tags, tags...)
	}
}

func WithDeprecated() Option {
	return func(op *Operation) {
		op.Deprecated = true
	}
}

func WithRequest[T any]() Option {
	return func(op *Operation) {
		op.Request = new(T)
	}
}

func WithJSON[T any](status int, description ...string) Option {
	return func(op *Operation) {
		op.Responses = append(op.Responses, Response{
			Status:      status,
			ContentType: "application/json",
			Body:        new(T),
			Description: firstDescription(description),
		})
	}
}

func WithNoContent(status int, description ...string) Option {
	return func(op *Operation) {
		op.Responses = append(op.Responses, Response{
			Status:      status,
			Description: firstDescription(description),
		})
	}
}

func WithError(status int, description ...string) Option {
	return func(op *Operation) {
		op.Responses = append(op.Responses, Response{
			Status:      status,
			Body:        new(string),
			Description: firstDescription(description),
		})
	}
}

func WithSession(scopes ...string) Option {
	return func(op *Operation) {
		op.Security = append(op.Security, Security{Name: "session", Scopes: scopes})
	}
}

func WithBearer(scopes ...string) Option {
	return func(op *Operation) {
		op.Security = append(op.Security, Security{Name: "bearer", Scopes: scopes})
	}
}

func WithAcceptLanguage(languages ...string) Option {
	if len(languages) == 0 {
		return func(*Operation) {}
	}
	return WithHeader("Accept-Language", "Preferable response language (available: "+strings.Join(languages, ", ")+")", false)
}

func WithURLParam(name, description string, required bool) Option {
	return withParam(InPath, name, description, required)
}

func WithQueryParam(name, description string, required bool) Option {
	return withParam(InQuery, name, description, required)
}

func WithHeader(name, description string, required bool) Option {
	return withParam(InHeader, name, description, required)
}

func withParam(in ParamIn, name, description string, required bool) Option {
	return func(op *Operation) {
		op.Params = append(op.Params, Param{In: in, Name: name, Description: description, Required: required})
	}
}
