package openapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/daarxwalker/enjin/app"
	"github.com/daarxwalker/enjin/app/openapi"
	"github.com/daarxwalker/enjin/app/routespec"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type itemRequest struct {
	Name string `json:"name"`
}

type itemResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func noContent(w http.ResponseWriter, _ *http.Request) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func TestSpec(t *testing.T) {
	spec := openapi.New(
		openapi.WithTitle("Items API"),
		openapi.WithVersion("1.0.0"),
		openapi.WithURL("https://api.example.com"),
		openapi.WithCookieSession("session", "Session cookie"),
		openapi.WithBearer("Session token"),
	)
	router := app.NewRouter(chi.NewRouter(), app.NewDefaultHandler(), app.WithRouterSpec(spec), app.WithRouterLangs("cs", "en"))
	router.Group("/api", func(r *app.Router) {
		r.Get("item", "/items/{id}", noContent,
			routespec.WithID("getItem"),
			routespec.WithSummary("Get item"),
			routespec.WithTags("items"),
			routespec.WithURLParam("id", "Item ID", true),
			routespec.WithJSON[itemResponse](http.StatusOK, "Item"),
			routespec.WithError(http.StatusNotFound, "Missing"),
			routespec.WithSession(),
		)
		r.Post("item.create", "/items", noContent,
			routespec.WithRequest[itemRequest](),
			routespec.WithNoContent(http.StatusCreated),
			routespec.WithQueryParam("dry", "Dry run", false),
			routespec.WithDeprecated(),
			routespec.WithBearer(),
		)
	})
	require.NoError(t, router.Err())

	raw, err := spec.MarshalJSON()
	require.NoError(t, err)
	var doc struct {
		OpenAPI string `json:"openapi"`
		Info    struct {
			Title   string `json:"title"`
			Version string `json:"version"`
		} `json:"info"`
		Servers    []struct{ URL string } `json:"servers"`
		Components struct {
			SecuritySchemes map[string]struct {
				Type        string `json:"type"`
				Scheme      string `json:"scheme"`
				Description string `json:"description"`
				In          string `json:"in"`
				Name        string `json:"name"`
			} `json:"securitySchemes"`
		} `json:"components"`
		Paths map[string]map[string]struct {
			OperationID string   `json:"operationId"`
			Summary     string   `json:"summary"`
			Tags        []string `json:"tags"`
			Deprecated  bool     `json:"deprecated"`
			Parameters  []struct {
				Name     string `json:"name"`
				In       string `json:"in"`
				Required bool   `json:"required"`
			} `json:"parameters"`
			RequestBody map[string]any            `json:"requestBody"`
			Responses   map[string]map[string]any `json:"responses"`
			Security    []map[string][]string     `json:"security"`
		} `json:"paths"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))

	require.Equal(t, "3.1.0", doc.OpenAPI)
	require.Equal(t, "Items API", doc.Info.Title)
	require.Equal(t, "1.0.0", doc.Info.Version)
	require.Equal(t, "https://api.example.com", doc.Servers[0].URL)
	require.Equal(t, "cookie", doc.Components.SecuritySchemes["session"].In)
	require.Equal(t, "http", doc.Components.SecuritySchemes["bearer"].Type)
	require.Equal(t, "bearer", doc.Components.SecuritySchemes["bearer"].Scheme)
	require.Equal(t, "Session token", doc.Components.SecuritySchemes["bearer"].Description)

	get := doc.Paths["/api/items/{id}"]["get"]
	require.Equal(t, "getItem", get.OperationID)
	require.Equal(t, "Get item", get.Summary)
	require.Equal(t, []string{"items"}, get.Tags)
	require.Len(t, get.Parameters, 2)
	require.Equal(t, "id", get.Parameters[0].Name)
	require.Equal(t, "path", get.Parameters[0].In)
	require.True(t, get.Parameters[0].Required)
	require.Equal(t, "Accept-Language", get.Parameters[1].Name)
	require.Equal(t, "header", get.Parameters[1].In)
	require.Equal(t, "Item", get.Responses["200"]["description"])
	require.Contains(t, get.Responses["200"]["content"], "application/json")
	require.Equal(t, "Missing", get.Responses["404"]["description"])
	require.Equal(t, []map[string][]string{{"session": {}}}, get.Security)

	post := doc.Paths["/api/items"]["post"]
	require.True(t, post.Deprecated)
	require.NotEmpty(t, post.RequestBody)
	require.Contains(t, post.Responses, "201")
	require.Equal(t, "dry", post.Parameters[0].Name)
	require.Equal(t, "query", post.Parameters[0].In)
	require.Equal(t, []map[string][]string{{"bearer": {}}}, post.Security)
}
