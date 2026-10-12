package app

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/daarxwalker/enjin/httpx"
)

func MustURL(ctx context.Context, name string, params ...map[string]any) string {
	path, err := URL(ctx, name, params...)
	if err != nil {
		panic(err)
	}
	return path
}

func URL(ctx context.Context, name string, params ...map[string]any) (string, error) {
	req := httpx.From(ctx)
	if req == nil {
		return "", ErrNoRouter
	}
	router := RouterFrom(ctx)
	if router == nil {
		return "", ErrNoRouter
	}
	route, ok := router.routeSet[name]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrRouteNotFound, name)
	}
	path := route.pattern
	query := make(url.Values)
	if len(params) > 0 {
		oldnew := make([]string, 0, 2*len(route.paramSet))
		for param, value := range params[0] {
			if _, exists := route.paramSet[param]; !exists {
				query.Add(param, fmt.Sprint(value))
				continue
			}
			oldnew = append(oldnew, "{"+param+"}", fmt.Sprint(value))
		}
		if len(oldnew) > 0 {
			path = strings.NewReplacer(oldnew...).Replace(path)
		}
	}
	path, err := url.JoinPath("/", req.PathPrefix, path)
	if err != nil {
		return "", fmt.Errorf("join %s paths failed: %w", strings.Join([]string{req.PathPrefix, route.pattern}, ","), err)
	}
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	return path, nil
}
