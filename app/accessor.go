package app

import "context"

func IsRouteActive(ctx context.Context, name string) bool {
	route := RouteFrom(ctx)
	return route != nil && route.name == name
}

func RouterFrom(ctx context.Context) *Router {
	v := ctx.Value(routerCtxKey)
	if v == nil {
		return nil
	}
	rcv, ok := v.(*RouterContextValue)
	if !ok {
		return nil
	}
	return rcv.Router
}

func RouteFrom(ctx context.Context) *Route {
	v := ctx.Value(routeCtxKey)
	if v == nil {
		return nil
	}
	rcv, ok := v.(*RouteContextValue)
	if !ok {
		return nil
	}
	return rcv.Route
}
