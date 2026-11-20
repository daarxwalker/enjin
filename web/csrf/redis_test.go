package csrf_test

import (
	"testing"

	"github.com/daarxwalker/enjin/internal/testredis"
	"github.com/daarxwalker/enjin/web/csrf"
	"github.com/daarxwalker/enjin/web/csrf/csrftest"
)

func TestRedisDriver(t *testing.T) {
	csrftest.RunDriverSuite(t, csrf.NewRedisDriver(testredis.New(t)))
}
