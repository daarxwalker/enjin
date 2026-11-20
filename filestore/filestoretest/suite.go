package filestoretest

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"github.com/daarxwalker/enjin/filestore"

	"github.com/stretchr/testify/require"
)

type Subject struct {
	Driver filestore.Driver
	// DerivesContentType is set for stores that ignore the content type given
	// to Put and derive it from the key or the body, as some WebDAV servers do.
	DerivesContentType bool
}

// RunDriverSuite checks that a Driver behaves like the MinIO one. Every case
// uses its own key prefix, so the store may be shared with other tests.
func RunDriverSuite(t *testing.T, subject Subject) {
	t.Run("put and get", func(t *testing.T) {
		checkPutAndGet(t, subject.Driver)
	})
	t.Run("stat", func(t *testing.T) {
		checkStat(t, subject.Driver)
	})
	t.Run("missing file", func(t *testing.T) {
		checkMissing(t, subject.Driver)
	})
	t.Run("content type is kept", func(t *testing.T) {
		checkContentType(t, subject)
	})
	t.Run("overwrite", func(t *testing.T) {
		checkOverwrite(t, subject.Driver)
	})
	t.Run("put with unknown size", func(t *testing.T) {
		checkUnknownSize(t, subject.Driver)
	})
	t.Run("large file", func(t *testing.T) {
		checkLargeFile(t, subject.Driver)
	})
	t.Run("key with spaces and diacritics", func(t *testing.T) {
		checkUnicodeKey(t, subject.Driver)
	})
	t.Run("list by prefix", func(t *testing.T) {
		checkListByPrefix(t, subject.Driver)
	})
	t.Run("list without match", func(t *testing.T) {
		checkListWithoutMatch(t, subject.Driver)
	})
	t.Run("delete", func(t *testing.T) {
		checkDelete(t, subject.Driver)
	})
}

func checkPutAndGet(t *testing.T, driver filestore.Driver) {
	key := "suite-get/hello.txt"
	value := []byte("hello filestore")
	requirePut(t, driver, key, value, "text/plain")

	object := requireGet(t, driver, key, value)
	require.Equal(t, key, object.Key)
	require.Equal(t, "text/plain", object.ContentType)
	require.Equal(t, int64(len(value)), object.Size)
	require.False(t, object.ModifiedAt.IsZero())
}

func checkStat(t *testing.T, driver filestore.Driver) {
	key := "suite-stat/data.json"
	value := []byte(`{"ok":true}`)
	requirePut(t, driver, key, value, "application/json")

	file := requireStat(t, driver, key)
	require.Equal(t, key, file.Key)
	require.Equal(t, "application/json", file.ContentType)
	require.Equal(t, int64(len(value)), file.Size)
	require.NotEmpty(t, file.ETag)
	require.False(t, file.ModifiedAt.IsZero())
}

func checkMissing(t *testing.T, driver filestore.Driver) {
	ctx := t.Context()
	object, exists, err := driver.Get(ctx, "suite-missing/file.txt")
	require.NoError(t, err)
	require.False(t, exists)
	require.Nil(t, object)

	file, exists, err := driver.Stat(ctx, "suite-missing/file.txt")
	require.NoError(t, err)
	require.False(t, exists)
	require.Zero(t, file)
}

func checkContentType(t *testing.T, subject Subject) {
	if subject.DerivesContentType {
		t.Skip("store derives the content type")
	}
	key := "suite-type/data.bin"
	requirePut(t, subject.Driver, key, []byte("custom"), "application/x-custom")

	require.Equal(t, "application/x-custom", requireStat(t, subject.Driver, key).ContentType)
	object := requireGet(t, subject.Driver, key, []byte("custom"))
	require.Equal(t, "application/x-custom", object.ContentType)
}

func checkOverwrite(t *testing.T, driver filestore.Driver) {
	key := "suite-overwrite/note.txt"
	requirePut(t, driver, key, []byte("first"), "text/plain")
	first := requireStat(t, driver, key)

	requirePut(t, driver, key, []byte("second version"), "text/plain")
	second := requireStat(t, driver, key)

	requireGet(t, driver, key, []byte("second version"))
	require.Equal(t, int64(len("second version")), second.Size)
	require.NotEqual(t, first.ETag, second.ETag)
}

func checkUnknownSize(t *testing.T, driver filestore.Driver) {
	key := "suite-unknown-size/note.txt"
	value := []byte("size not known up front")
	require.NoError(t, driver.Put(t.Context(), key, bytes.NewBuffer(value), -1, "text/plain"))

	requireGet(t, driver, key, value)
	require.Equal(t, int64(len(value)), requireStat(t, driver, key).Size)
}

func checkLargeFile(t *testing.T, driver filestore.Driver) {
	key := "suite-large/random.bin"
	value := make([]byte, 3<<20)
	_, err := rand.Read(value)
	require.NoError(t, err)
	requirePut(t, driver, key, value, "application/octet-stream")

	requireGet(t, driver, key, value)
}

func checkUnicodeKey(t *testing.T, driver filestore.Driver) {
	key := "suite-unicode/žluťoučký kůň #1?.txt"
	value := []byte("úpěl ďábelské ódy")
	requirePut(t, driver, key, value, "text/plain")

	requireGet(t, driver, key, value)
	require.Equal(t, []string{key}, listKeys(t, driver, "suite-unicode/"))
}

func checkListByPrefix(t *testing.T, driver filestore.Driver) {
	for _, key := range []string{
		"suite-list/a.txt",
		"suite-list/ab/c.txt",
		"suite-list/nested/deep/d.txt",
		"suite-list/b.txt",
		"suite-listing/e.txt",
	} {
		requirePut(t, driver, key, []byte(key), "text/plain")
	}

	require.ElementsMatch(
		t,
		[]string{"suite-list/a.txt", "suite-list/ab/c.txt", "suite-list/nested/deep/d.txt", "suite-list/b.txt"},
		listKeys(t, driver, "suite-list/"),
	)
	require.ElementsMatch(t, []string{"suite-list/a.txt", "suite-list/ab/c.txt"}, listKeys(t, driver, "suite-list/a"))
	require.ElementsMatch(t, []string{"suite-list/nested/deep/d.txt"}, listKeys(t, driver, "suite-list/nested/"))
	require.ElementsMatch(
		t,
		[]string{
			"suite-list/a.txt",
			"suite-list/ab/c.txt",
			"suite-list/nested/deep/d.txt",
			"suite-list/b.txt",
			"suite-listing/e.txt",
		},
		listKeys(t, driver, "suite-list"),
	)

	files, err := driver.List(t.Context(), "suite-list/nested/")
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Equal(t, int64(len("suite-list/nested/deep/d.txt")), files[0].Size)
}

func checkListWithoutMatch(t *testing.T, driver filestore.Driver) {
	require.Empty(t, listKeys(t, driver, "suite-nothing/"))
}

func checkDelete(t *testing.T, driver filestore.Driver) {
	ctx := t.Context()
	key := "suite-delete/nested/gone.txt"
	requirePut(t, driver, key, []byte("bye"), "text/plain")
	requirePut(t, driver, "suite-delete/kept.txt", []byte("stay"), "text/plain")

	require.NoError(t, driver.Delete(ctx, key))
	_, exists, err := driver.Stat(ctx, key)
	require.NoError(t, err)
	require.False(t, exists)
	require.Equal(t, []string{"suite-delete/kept.txt"}, listKeys(t, driver, "suite-delete/"))

	require.NoError(t, driver.Delete(ctx, key))
}

func requirePut(t *testing.T, driver filestore.Driver, key string, value []byte, contentType string) {
	t.Helper()
	require.NoError(t, driver.Put(t.Context(), key, bytes.NewReader(value), int64(len(value)), contentType))
}

func requireGet(t *testing.T, driver filestore.Driver, key string, expected []byte) *filestore.Object {
	t.Helper()
	object, exists, err := driver.Get(t.Context(), key)
	require.NoError(t, err)
	require.True(t, exists, key)
	body, readErr := io.ReadAll(object.Body)
	require.NoError(t, readErr)
	require.NoError(t, object.Body.Close())
	require.Equal(t, expected, body, key)
	return object
}

func requireStat(t *testing.T, driver filestore.Driver, key string) filestore.File {
	t.Helper()
	file, exists, err := driver.Stat(t.Context(), key)
	require.NoError(t, err)
	require.True(t, exists, key)
	return file
}

func listKeys(t *testing.T, driver filestore.Driver, prefix string) []string {
	t.Helper()
	files, err := driver.List(t.Context(), prefix)
	require.NoError(t, err)
	keys := make([]string, 0, len(files))
	for _, file := range files {
		keys = append(keys, file.Key)
	}
	return keys
}
