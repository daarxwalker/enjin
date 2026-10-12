package antibot

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
)

type Driver interface {
	CreateWidget(ctx context.Context) templ.Component
	Verify(r *http.Request) (success bool, err error)
}
