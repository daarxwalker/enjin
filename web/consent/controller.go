package consent

import (
	"net/http"

	"github.com/daarxwalker/enjin/app"
	"github.com/daarxwalker/enjin/httpx"
)

type controller struct{}

const (
	PathPattern = "/__consent__"
)

func Mount(r *app.Router, name string) {
	c := new(controller)
	r.With(http.NewCrossOriginProtection().Handler).Post(name, PathPattern, c.saveConsent)
}

func (c *controller) saveConsent(w http.ResponseWriter, r *http.Request) error {
	data := ParseSaveConsent(r)
	Attach(
		w, r, Cookies{
			Analytic:   data.Analytic,
			Preference: data.Preference,
			Marketing:  data.Marketing,
		},
	)
	return httpx.Redirect(w, r, data.Redirect)
}
