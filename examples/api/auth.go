package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/daarxwalker/enjin/app"
	"github.com/daarxwalker/enjin/app/routespec"
	"github.com/daarxwalker/enjin/httpx"
	"github.com/daarxwalker/enjin/session"
	"github.com/daarxwalker/enjin/validate"
)

type loginRequest struct {
	Name string `json:"name"`
}

type loginResponse struct {
	Token string `json:"token"`
}

type authController struct {
	store *session.Store
}

func mountAuth(r *app.Router, store *session.Store) {
	c := &authController{store: store}
	r.Post(
		"login", "/login", c.login,
		routespec.WithID("login"),
		routespec.WithTags("auth"),
		routespec.WithRequest[loginRequest](),
		routespec.WithJSON[loginResponse](http.StatusOK),
		routespec.WithJSON[errorResponse](http.StatusUnprocessableEntity, "Validation failed"),
	)
}

func (c *authController) login(w http.ResponseWriter, r *http.Request) error {
	var req loginRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		return httpx.Error(http.StatusBadRequest, fmt.Errorf("invalid JSON body: %w", err))
	}
	validation := make(validate.Result)
	validation.Text("name", req.Name, 2, 40)
	if !validation.Zero() {
		return httpx.JSON(w, http.StatusUnprocessableEntity, errorResponse{
			Status:  http.StatusUnprocessableEntity,
			Message: "Validation failed",
			Errors:  validation,
		})
	}
	token, err := c.store.Seed(r.Context(), session.WithID(req.Name), session.WithRole(editor))
	if err != nil {
		return httpx.Error(http.StatusInternalServerError, fmt.Errorf("seed session failed: %w", err))
	}
	return httpx.JSON(w, http.StatusOK, loginResponse{Token: token})
}
