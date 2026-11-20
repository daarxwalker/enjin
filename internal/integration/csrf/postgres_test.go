package csrf_test

import (
	"context"
	"testing"

	"github.com/daarxwalker/enjin/enjintest"
	"github.com/daarxwalker/enjin/infra/db/migrations/keyvalue"
	"github.com/daarxwalker/enjin/web/csrf"
	"github.com/daarxwalker/enjin/web/csrf/csrftest"
)

func TestPostgresDriver(t *testing.T) {
	pool := enjintest.NewPostgres(context.Background(), t, enjintest.WithMigrationsFS(keyvalue.FS))
	csrftest.RunDriverSuite(t, csrf.NewPostgresDriver(pool))
}
