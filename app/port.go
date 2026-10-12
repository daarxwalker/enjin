package app

import (
	"net/http"

	"github.com/daarxwalker/enjin/app/routespec"
)

type Handler interface {
	Handle(fn func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc
}

type Spec interface {
	AddOperation(method, pattern string, operation routespec.Operation) error
	MarshalJSON() ([]byte, error)
}
