package backlink

import (
	"net/http"
	"net/url"

	"github.com/daarxwalker/enjin/httpx"
)

func Query(r *http.Request, path string) (url.Values, bool) {
	referer, err := url.Parse(r.Referer())
	if err != nil || referer.Host != httpx.HostFrom(r) || referer.Path != path {
		return nil, false
	}
	return referer.Query(), true
}
