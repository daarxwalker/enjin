package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"sync"

	"github.com/daarxwalker/enjin/app"
	"github.com/daarxwalker/enjin/app/routespec"
	"github.com/daarxwalker/enjin/httpx"
	"github.com/daarxwalker/enjin/security"
	"github.com/daarxwalker/enjin/session"
	"github.com/daarxwalker/enjin/validate"

	"github.com/go-chi/chi/v5"
)

var errProductNotFound = errors.New("product not found")

type product struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedBy string `json:"createdBy"`
	Price     int    `json:"price"`
}

type createProductRequest struct {
	Name  string `json:"name"`
	Price int    `json:"price"`
}

type productRepo struct {
	products []product
	mu       sync.Mutex
}

func newProductRepo() *productRepo {
	return &productRepo{products: []product{}}
}

func (repo *productRepo) list() []product {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	return slices.Clone(repo.products)
}

func (repo *productRepo) get(id string) (product, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	for _, p := range repo.products {
		if p.ID == id {
			return p, nil
		}
	}
	return product{}, errProductNotFound
}

func (repo *productRepo) create(name string, price int, createdBy string) product {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	p := product{ID: strconv.Itoa(len(repo.products) + 1), Name: name, Price: price, CreatedBy: createdBy}
	repo.products = append(repo.products, p)
	return p
}

type productController struct {
	repo *productRepo
}

func mountProducts(r *app.Router, repo *productRepo) {
	c := &productController{repo: repo}
	r.Group("/products", func(r *app.Router) {
		r.Get(
			"listProducts", "/", c.list,
			routespec.WithID("listProducts"),
			routespec.WithTags("products"),
			routespec.WithJSON[[]product](http.StatusOK),
		)
		r.Get(
			"getProduct", "/{id}", c.get,
			routespec.WithID("getProduct"),
			routespec.WithTags("products"),
			routespec.WithURLParam("id", "Product id", true),
			routespec.WithJSON[product](http.StatusOK),
			routespec.WithError(http.StatusNotFound, "Not found"),
		)
		r.With(session.Restrict(security.RequireSecurable(writeProducts))).Post(
			"createProduct", "/", c.create,
			routespec.WithID("createProduct"),
			routespec.WithTags("products"),
			routespec.WithBearer(),
			routespec.WithRequest[createProductRequest](),
			routespec.WithJSON[product](http.StatusCreated, "Created"),
			routespec.WithJSON[errorResponse](http.StatusUnprocessableEntity, "Validation failed"),
			routespec.WithError(http.StatusUnauthorized, "Missing or invalid token"),
		)
	})
}

func (c *productController) list(w http.ResponseWriter, _ *http.Request) error {
	return httpx.JSON(w, http.StatusOK, c.repo.list())
}

func (c *productController) get(w http.ResponseWriter, r *http.Request) error {
	found, err := c.repo.get(chi.URLParam(r, "id"))
	if errors.Is(err, errProductNotFound) {
		return httpx.StatusError(http.StatusNotFound)
	}
	return httpx.JSON(w, http.StatusOK, found)
}

func (c *productController) create(w http.ResponseWriter, r *http.Request) error {
	var req createProductRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		return httpx.Error(http.StatusBadRequest, fmt.Errorf("invalid JSON body: %w", err))
	}
	validation := make(validate.Result)
	validation.Text("name", req.Name, 2, 80)
	validation.Int("price", req.Price, 1, 100_000)
	if !validation.Zero() {
		return httpx.JSON(w, http.StatusUnprocessableEntity, errorResponse{
			Status:  http.StatusUnprocessableEntity,
			Message: "Validation failed",
			Errors:  validation,
		})
	}
	created := c.repo.create(req.Name, req.Price, session.From(r.Context()).ID)
	return httpx.JSON(w, http.StatusCreated, created)
}
