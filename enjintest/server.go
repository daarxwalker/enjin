package enjintest

import (
	"net/http/httptest"

	"github.com/daarxwalker/enjin/app"
)

func NewServer(r *app.Router) *httptest.Server {
	return httptest.NewServer(r.Raw())
}
