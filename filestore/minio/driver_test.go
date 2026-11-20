package minio

import (
	"context"
	"testing"

	"github.com/daarxwalker/enjin/enjintest"
	"github.com/daarxwalker/enjin/filestore/filestoretest"
)

func TestDriver(t *testing.T) {
	bucket := "enjin-test"
	client := enjintest.NewSeaweedFS(context.Background(), t, bucket)
	filestoretest.RunDriverSuite(t, filestoretest.Subject{Driver: NewDriver(client, bucket)})
}
