package app

import "errors"

var (
	ErrNoRouter      = errors.New("app: no request or router in context")
	ErrRouteNotFound = errors.New("app: route not found")
)
