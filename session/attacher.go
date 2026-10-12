package session

import (
	"maps"
	"net/http"

	"github.com/daarxwalker/enjin/security"
	"github.com/daarxwalker/enjin/token"
)

type AttachOption func(*Profile)

func Attach(w http.ResponseWriter, r *http.Request, options ...AttachOption) *Profile {
	if len(options) == 0 {
		return nil
	}
	ctx := r.Context()
	cv := from(ctx)
	if cv == nil {
		return nil
	}
	next := &Profile{Token: token.New()}
	if cv.bearer && cv.profile != nil {
		next.Token = cv.profile.Token
	}
	for _, option := range options {
		option(next)
	}
	if current := cv.profile; current != nil && (next.ID == "" || next.ID == current.ID) {
		kept := *current
		kept.Token = next.Token
		kept.Values = maps.Clone(current.Values)
		for _, option := range options {
			option(&kept)
		}
		next = &kept
	}
	cv.replaceProfile(ctx, w, next)
	return cv.profile
}

func Destroy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cv := from(ctx)
	if cv == nil || cv.token == "" {
		return
	}
	cv.destroyProfile(ctx, w)
}

func WithID(id string) AttachOption {
	return func(p *Profile) {
		p.ID = id
	}
}

func WithKey(key string) AttachOption {
	return func(p *Profile) {
		p.Key = key
	}
}

func WithRole(role *security.Role) AttachOption {
	return func(p *Profile) {
		p.Role = role
	}
}
