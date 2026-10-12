package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestAPI(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	api, err := newAPI(context.Background(), rdb, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	h := api.engine.Root.Raw()

	res := call(t, h, http.MethodGet, "/products/", "", "")
	require.Equal(t, http.StatusOK, res.Code)
	require.JSONEq(t, `[]`, res.Body.String())

	res = call(t, h, http.MethodPost, "/products/", "", `{"name":"Shampoo","price":120}`)
	require.Equal(t, http.StatusUnauthorized, res.Code)

	res = call(t, h, http.MethodPost, "/login", "", `{"name":"x"}`)
	require.Equal(t, http.StatusUnprocessableEntity, res.Code)
	require.Contains(t, res.Body.String(), `"minLen"`)

	res = call(t, h, http.MethodPost, "/login", "", `{"name":"alice"}`)
	require.Equal(t, http.StatusOK, res.Code)
	var login loginResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &login))
	require.NotEmpty(t, login.Token)
	require.Empty(t, res.Header().Values("Set-Cookie"))

	res = call(t, h, http.MethodPost, "/products/", login.Token, `{"name":"Shampoo","price":0}`)
	require.Equal(t, http.StatusUnprocessableEntity, res.Code)
	require.Contains(t, res.Body.String(), `"price"`)

	res = call(t, h, http.MethodPost, "/products/", login.Token, `{"name":`)
	require.Equal(t, http.StatusBadRequest, res.Code)

	res = call(t, h, http.MethodPost, "/products/", login.Token, `{"name":"Shampoo","price":120}`)
	require.Equal(t, http.StatusCreated, res.Code)
	require.JSONEq(t, `{"id":"1","name":"Shampoo","price":120,"createdBy":"alice"}`, res.Body.String())

	res = call(t, h, http.MethodGet, "/products/1", "", "")
	require.Equal(t, http.StatusOK, res.Code)
	require.Contains(t, res.Body.String(), `"Shampoo"`)

	res = call(t, h, http.MethodGet, "/products/2", "", "")
	require.Equal(t, http.StatusNotFound, res.Code)
	require.JSONEq(t, `{"status":404,"message":"Not Found"}`, res.Body.String())

	preflight := httptest.NewRequest(http.MethodOptions, "/products/", nil)
	preflight.Header.Set("Origin", "http://localhost:5173")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodPost)
	res = httptest.NewRecorder()
	h.ServeHTTP(res, preflight)
	require.Equal(t, "http://localhost:5173", res.Header().Get("Access-Control-Allow-Origin"))

	spec, err := api.spec.MarshalJSON()
	require.NoError(t, err)
	require.Contains(t, string(spec), `"/products/{id}"`)
	require.Contains(t, string(spec), `"bearer"`)
}

func call(t *testing.T, h http.Handler, method, target, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader = http.NoBody
	if body != "" {
		reader = bytes.NewBufferString(body)
	}
	r := httptest.NewRequest(method, target, reader)
	if strings.TrimSpace(body) != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
