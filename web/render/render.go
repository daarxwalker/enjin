package render

import (
	"fmt"
	"net/http"

	"github.com/a-h/templ"
)

func Templ(w http.ResponseWriter, r *http.Request, status int, component templ.Component, wrapper ...templ.Component) error {
	if component == nil {
		return nil
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if len(wrapper) > 0 {
		if err := wrapper[0].Render(templ.WithChildren(r.Context(), component), w); err != nil {
			return fmt.Errorf("render templ component with wrapper failed: %w", err)
		}
		return nil
	}
	if err := component.Render(r.Context(), w); err != nil {
		return fmt.Errorf("render templ component failed: %w", err)
	}
	return nil
}

func TemplFragments(w http.ResponseWriter, r *http.Request, status int, component templ.Component, ids ...any) error {
	if component == nil {
		return nil
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := templ.RenderFragments(r.Context(), w, component, ids...); err != nil {
		return fmt.Errorf("render templ component failed: %w", err)
	}
	return nil
}
