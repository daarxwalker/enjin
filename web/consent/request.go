package consent

import (
	"net/http"
	"path"

	"github.com/daarxwalker/enjin/httpx"
)

type SaveConsent struct {
	Redirect   string `form:"redirect"`
	Analytic   bool   `form:"analytic"`
	Preference bool   `form:"preference"`
	Marketing  bool   `form:"marketing"`
}

func ParseSaveConsent(r *http.Request) SaveConsent {
	action := r.FormValue("action")
	redirect := r.FormValue("redirect")
	if !httpx.IsLocalPath(redirect) {
		redirect = path.Join("/", httpx.PathPrefix(r))
	}
	switch action {
	case "all":
		return SaveConsent{
			Redirect:   redirect,
			Analytic:   true,
			Preference: true,
			Marketing:  true,
		}
	case "necessary":
		return SaveConsent{Redirect: redirect}
	case "selection":
		return SaveConsent{
			Redirect:   redirect,
			Analytic:   r.FormValue("analytic") == "true",
			Preference: r.FormValue("preference") == "true",
			Marketing:  r.FormValue("marketing") == "true",
		}
	default:
		return SaveConsent{Redirect: redirect}
	}
}
