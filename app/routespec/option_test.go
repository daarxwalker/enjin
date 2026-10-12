package routespec_test

import (
	"net/http"
	"testing"

	"github.com/daarxwalker/enjin/app/routespec"

	"github.com/stretchr/testify/require"
)

type request struct{ Name string }

type response struct{ ID string }

func TestOptions(t *testing.T) {
	t.Run(
		"build operation", func(t *testing.T) {
			op := apply(
				routespec.WithID("createProduct"),
				routespec.WithSummary("Create product"),
				routespec.WithDescription("Creates a product."),
				routespec.WithTags("products"),
				routespec.WithTags("admin"),
				routespec.WithDeprecated(),
				routespec.WithRequest[request](),
				routespec.WithJSON[response](http.StatusCreated, "Created"),
				routespec.WithNoContent(http.StatusNoContent),
				routespec.WithError(http.StatusUnprocessableEntity, "Invalid input"),
				routespec.WithSession("admin"),
				routespec.WithBearer("products.write"),
			)
			require.Equal(t, "createProduct", op.ID)
			require.Equal(t, "Create product", op.Summary)
			require.Equal(t, "Creates a product.", op.Description)
			require.Equal(t, []string{"products", "admin"}, op.Tags)
			require.True(t, op.Deprecated)
			require.IsType(t, new(request), op.Request)
			require.Equal(
				t, []routespec.Security{
					{Name: "session", Scopes: []string{"admin"}},
					{Name: "bearer", Scopes: []string{"products.write"}},
				}, op.Security,
			)
			require.Len(t, op.Responses, 3)
			require.Equal(t, http.StatusCreated, op.Responses[0].Status)
			require.Equal(t, "application/json", op.Responses[0].ContentType)
			require.IsType(t, new(response), op.Responses[0].Body)
			require.Equal(t, "Created", op.Responses[0].Description)
			require.Equal(t, routespec.Response{Status: http.StatusNoContent}, op.Responses[1])
			require.IsType(t, new(string), op.Responses[2].Body)
			require.Equal(t, "Invalid input", op.Responses[2].Description)
		},
	)

	t.Run(
		"params are appended", func(t *testing.T) {
			op := apply(
				routespec.WithURLParam("id", "Product ID", true),
				routespec.WithQueryParam("q", "Search", false),
				routespec.WithHeader("X-Request-ID", "Trace", false),
				routespec.WithAcceptLanguage("cs", "en"),
				routespec.WithAcceptLanguage(),
			)
			require.Equal(
				t, []routespec.Param{
					{In: routespec.InPath, Name: "id", Description: "Product ID", Required: true},
					{In: routespec.InQuery, Name: "q", Description: "Search"},
					{In: routespec.InHeader, Name: "X-Request-ID", Description: "Trace"},
					{In: routespec.InHeader, Name: "Accept-Language", Description: "Preferable response language (available: cs, en)"},
				}, op.Params,
			)
		},
	)
}

func apply(options ...routespec.Option) routespec.Operation {
	var op routespec.Operation
	for _, option := range options {
		option(&op)
	}
	return op
}
