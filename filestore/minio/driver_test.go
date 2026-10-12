package minio

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/daarxwalker/enjin/enjintest"

	"github.com/stretchr/testify/require"
)

func TestDriver(t *testing.T) {
	ctx := context.Background()
	bucket := "enjin-test"
	client := enjintest.NewSeaweedFS(ctx, t, bucket)
	driver := NewDriver(client, bucket)
	require.NotNil(t, driver)

	t.Run(
		"put and get", func(t *testing.T) {
			key := "get/hello.txt"
			value := []byte("hello filestore")
			require.NoError(t, driver.Put(ctx, key, bytes.NewReader(value), int64(len(value)), "text/plain"))

			object, exists, err := driver.Get(ctx, key)
			require.NoError(t, err)
			require.True(t, exists)
			body, readErr := io.ReadAll(object.Body)
			require.NoError(t, readErr)
			require.NoError(t, object.Body.Close())
			require.Equal(t, value, body)
			require.Equal(t, key, object.Key)
			require.Equal(t, "text/plain", object.ContentType)
			require.Equal(t, int64(len(value)), object.Size)
		},
	)

	t.Run(
		"stat", func(t *testing.T) {
			key := "stat/data.json"
			value := []byte(`{"ok":true}`)
			require.NoError(t, driver.Put(ctx, key, bytes.NewReader(value), int64(len(value)), "application/json"))

			file, exists, err := driver.Stat(ctx, key)
			require.NoError(t, err)
			require.True(t, exists)
			require.Equal(t, key, file.Key)
			require.Equal(t, "application/json", file.ContentType)
			require.Equal(t, int64(len(value)), file.Size)
			require.NotEmpty(t, file.ETag)
			require.False(t, file.ModifiedAt.IsZero())
		},
	)

	t.Run(
		"missing", func(t *testing.T) {
			object, exists, err := driver.Get(ctx, "missing/file.txt")
			require.NoError(t, err)
			require.False(t, exists)
			require.Nil(t, object)

			_, exists, err = driver.Stat(ctx, "missing/file.txt")
			require.NoError(t, err)
			require.False(t, exists)
		},
	)

	t.Run(
		"list by prefix", func(t *testing.T) {
			for _, key := range []string{"list/a.txt", "list/nested/b.txt", "other/c.txt"} {
				value := []byte(key)
				require.NoError(t, driver.Put(ctx, key, bytes.NewReader(value), int64(len(value)), "text/plain"))
			}

			files, err := driver.List(ctx, "list/")
			require.NoError(t, err)
			keys := make([]string, 0, len(files))
			for _, file := range files {
				keys = append(keys, file.Key)
			}
			require.ElementsMatch(t, []string{"list/a.txt", "list/nested/b.txt"}, keys)
		},
	)

	t.Run(
		"delete", func(t *testing.T) {
			key := "delete/gone.txt"
			value := []byte("bye")
			require.NoError(t, driver.Put(ctx, key, bytes.NewReader(value), int64(len(value)), "text/plain"))

			require.NoError(t, driver.Delete(ctx, key))
			_, exists, err := driver.Stat(ctx, key)
			require.NoError(t, err)
			require.False(t, exists)
			require.NoError(t, driver.Delete(ctx, key))
		},
	)
}
