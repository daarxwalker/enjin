package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"strconv"

	"github.com/daarxwalker/enjin/app"
	"github.com/daarxwalker/enjin/infra/cache"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	port, err := strconv.ParseUint(os.Getenv("PORT"), 10, 16)
	if err != nil {
		port = 8080
	}
	rdb, err := cache.Connect(ctx, addr)
	if err != nil {
		log.Fatal(err)
	}
	api, err := newAPI(ctx, rdb, logger, app.WithPort(port), app.WithCleanup(func(context.Context) error { return rdb.Close() }))
	if err != nil {
		log.Fatal(err)
	}
	if err := api.engine.Start(); err != nil {
		log.Fatal(err)
	}
}
