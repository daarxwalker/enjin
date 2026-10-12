package app

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/daarxwalker/enjin/app/routespec"

	"github.com/go-chi/chi/v5"
)

type specOperation struct {
	method    string
	pattern   string
	operation routespec.Operation
}

type fakeSpec struct {
	operations []specOperation
	err        error
}

func (s *fakeSpec) AddOperation(method, pattern string, operation routespec.Operation) error {
	s.operations = append(s.operations, specOperation{method: method, pattern: pattern, operation: operation})
	return s.err
}

func (s *fakeSpec) MarshalJSON() ([]byte, error) {
	return []byte(`{"openapi":"3.1.0"}`), nil
}

type itemResponse struct {
	ID string `json:"id"`
}

func TestSpec(t *testing.T) {
	t.Run("router passes operations", func(t *testing.T) {
		spec := &fakeSpec{}
		r := NewRouter(chi.NewRouter(), NewDefaultHandler(), WithRouterSpec(spec), WithRouterLangs("cs", "en"))
		r.Group("/api", func(r *Router) {
			r.Get("item", "/items/{id}", noContent,
				routespec.WithID("getItem"),
				routespec.WithSummary("Get item"),
				routespec.WithTags("items"),
				routespec.WithURLParam("id", "Item ID", true),
				routespec.WithJSON[itemResponse](http.StatusOK, "Item"),
				routespec.WithError(http.StatusNotFound),
				routespec.WithSession(),
			)
		})
		if err := r.Err(); err != nil {
			t.Fatal(err)
		}
		want := []specOperation{{
			method:  http.MethodGet,
			pattern: "/api/items/{id}",
			operation: routespec.Operation{
				ID:      "getItem",
				Summary: "Get item",
				Tags:    []string{"items"},
				Params: []routespec.Param{
					{In: routespec.InPath, Name: "id", Description: "Item ID", Required: true},
					{In: routespec.InHeader, Name: "Accept-Language", Description: "Preferable response language (available: cs, en)"},
				},
				Responses: []routespec.Response{
					{Status: http.StatusOK, ContentType: "application/json", Body: new(itemResponse), Description: "Item"},
					{Status: http.StatusNotFound, Body: new(string)},
				},
				Security: []routespec.Security{{Name: "session"}},
			},
		}}
		if !reflect.DeepEqual(spec.operations, want) {
			t.Fatalf("operations = %+v, want %+v", spec.operations, want)
		}
	})

	t.Run("router records spec error", func(t *testing.T) {
		spec := &fakeSpec{err: errors.New("broken")}
		r := NewRouter(chi.NewRouter(), NewDefaultHandler(), WithRouterSpec(spec))
		r.Get("home", "/", noContent)
		if err := r.Err(); err == nil {
			t.Fatal("Err() = nil, want spec error")
		}
	})

	t.Run("router without spec", func(t *testing.T) {
		r := NewRouter(chi.NewRouter(), NewDefaultHandler())
		r.Get("home", "/", noContent, routespec.WithSummary("Home"))
		if err := r.Err(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("engine serves spec", func(t *testing.T) {
		spec := &fakeSpec{}
		e := newTestEngine(t,
			WithSpec(spec),
			WithRouter(NewRouter(chi.NewRouter(), NewDefaultHandler())),
		)
		if err := e.createSpecRoute(); err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		e.Root.Raw().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/spec.json", nil))
		if w.Code != http.StatusOK || w.Body.String() != `{"openapi":"3.1.0"}` {
			t.Fatalf("GET /spec.json = %d %q", w.Code, w.Body.String())
		}
	})
}
