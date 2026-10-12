package app

type (
	routerCtxKeyType struct{}
	routeCtxKeyType  struct{}
)

var (
	routerCtxKey = routerCtxKeyType{}
	routeCtxKey  = routeCtxKeyType{}
)

type RouterContextValue struct {
	Router *Router
}

type RouteContextValue struct {
	Route *Route
}
