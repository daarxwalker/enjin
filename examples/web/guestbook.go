package main

import (
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/daarxwalker/enjin/app"
	"github.com/daarxwalker/enjin/httpx"
	"github.com/daarxwalker/enjin/validate"
	"github.com/daarxwalker/enjin/web/csrf"
	"github.com/daarxwalker/enjin/web/flash"
	"github.com/daarxwalker/enjin/web/head"
	"github.com/daarxwalker/enjin/web/render"
)

const routeHome = "home"

type entry struct {
	Name    string
	Message string
}

type guestbook struct {
	entries []entry
	mu      sync.Mutex
}

func newGuestbook() *guestbook {
	return new(guestbook)
}

func (g *guestbook) list() []entry {
	g.mu.Lock()
	defer g.mu.Unlock()
	return slices.Clone(g.entries)
}

func (g *guestbook) add(e entry) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.entries = append(g.entries, e)
}

type guestbookController struct {
	book *guestbook
}

func mountGuestbook(r *app.Router, csrfManager *csrf.InputManager, book *guestbook) {
	c := &guestbookController{book: book}
	r.Get(routeHome, "/", c.render)
	r.With(csrfManager.Require).Post(routeHome, "/", c.sign)
}

func (c *guestbookController) render(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	head.Attach(r.Context(), head.WithTitle("Guestbook"), head.WithDescription("Enjin server-rendered example"))
	return render.Templ(w, r, http.StatusOK, guestbookPage(guestbookVM{ //nolint:contextcheck // templ passes ctx to Render
		Entries:    c.book.list(),
		Validation: validate.From(ctx),
		CSRF:       csrf.NewInputFrom(ctx),
	}), layout())
}

func (c *guestbookController) sign(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	signed := entry{
		Name:    strings.TrimSpace(r.FormValue("name")),
		Message: strings.TrimSpace(r.FormValue("message")),
	}
	validation := make(validate.Result)
	validation.Text("name", signed.Name, 2, 40)
	validation.Text("message", signed.Message, 1, 280)
	if !validation.Zero() {
		validate.Attach(ctx, validation)
		return httpx.Redirect(w, r, app.MustURL(ctx, routeHome))
	}
	c.book.add(signed)
	flash.Attach(r.Context(), flash.WithSuccess("Thanks for signing!"))
	return httpx.Redirect(w, r, app.MustURL(ctx, routeHome))
}
