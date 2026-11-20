package disk_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/daarxwalker/enjin/filestore"
	"github.com/daarxwalker/enjin/filestore/disk"
	"github.com/daarxwalker/enjin/filestore/filestoretest"

	"github.com/stretchr/testify/require"
)

func TestDriver(t *testing.T) {
	filestoretest.RunDriverSuite(t, filestoretest.Subject{Driver: newDriver(t, t.TempDir())})
}

func TestDriverKeys(t *testing.T) {
	driver := newDriver(t, t.TempDir())
	for _, key := range []string{
		"", ".", "/abs.txt", "../up.txt", "a/../b.txt", "a//b.txt", "a/", "./a.txt",
		`a\b.txt`, ".meta/x.json", "a/.enjin-tmp-x",
	} {
		ctx := t.Context()
		require.ErrorIs(t, driver.Put(ctx, key, strings.NewReader("x"), 1, "text/plain"), disk.ErrInvalidKey, key)
		_, _, err := driver.Get(ctx, key)
		require.ErrorIs(t, err, disk.ErrInvalidKey, key)
		_, _, err = driver.Stat(ctx, key)
		require.ErrorIs(t, err, disk.ErrInvalidKey, key)
		require.ErrorIs(t, driver.Delete(ctx, key), disk.ErrInvalidKey, key)
	}
}

func TestDriverStaysInRoot(t *testing.T) {
	ctx := t.Context()
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600))
	dir := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(dir, "link")))
	driver := newDriver(t, dir)

	_, _, err := driver.Get(ctx, "link/secret.txt")
	require.Error(t, err)
	_, _, err = driver.Stat(ctx, "link/secret.txt")
	require.Error(t, err)
	require.Error(t, driver.Put(ctx, "link/new.txt", strings.NewReader("x"), 1, "text/plain"))
	_, err = os.Stat(filepath.Join(outside, "new.txt"))
	require.ErrorIs(t, err, os.ErrNotExist)

	files, err := driver.List(ctx, "")
	require.NoError(t, err)
	require.Empty(t, files)
}

func TestDriverLayout(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	driver := newDriver(t, dir)

	t.Run("delete removes empty directories", func(t *testing.T) {
		require.NoError(t, driver.Put(ctx, "tree/deep/file.txt", strings.NewReader("x"), 1, "text/plain"))
		require.NoError(t, driver.Put(ctx, "tree/kept.txt", strings.NewReader("y"), 1, "text/plain"))

		require.NoError(t, driver.Delete(ctx, "tree/deep/file.txt"))
		require.NoDirExists(t, filepath.Join(dir, "tree", "deep"))
		require.NoDirExists(t, filepath.Join(dir, ".meta", "tree", "deep"))
		require.FileExists(t, filepath.Join(dir, "tree", "kept.txt"))

		require.NoError(t, driver.Delete(ctx, "tree/kept.txt"))
		require.NoDirExists(t, filepath.Join(dir, "tree"))
		require.DirExists(t, dir)
	})

	t.Run("file stored without the driver", func(t *testing.T) {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "foreign"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "foreign", "page.html"), []byte("<p>hi</p>"), 0o600))

		file, exists, err := driver.Stat(ctx, "foreign/page.html")
		require.NoError(t, err)
		require.True(t, exists)
		require.Equal(t, "text/html; charset=utf-8", file.ContentType)
		require.NotEmpty(t, file.ETag)
		require.Equal(t, int64(len("<p>hi</p>")), file.Size)
	})

	t.Run("file changed behind the driver", func(t *testing.T) {
		require.NoError(t, driver.Put(ctx, "changed.bin", strings.NewReader("first"), 5, "application/x-custom"))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "changed.bin"), []byte("much longer"), 0o600))

		file, exists, err := driver.Stat(ctx, "changed.bin")
		require.NoError(t, err)
		require.True(t, exists)
		require.Equal(t, "application/octet-stream", file.ContentType)
		require.NotEqual(t, sha256Hex("first"), file.ETag)
	})

	t.Run("wrong size keeps the previous content", func(t *testing.T) {
		require.NoError(t, driver.Put(ctx, "sized.txt", strings.NewReader("old"), 3, "text/plain"))
		require.ErrorIs(t, driver.Put(ctx, "sized.txt", strings.NewReader("new"), 10, "text/plain"), disk.ErrSizeMismatch)

		requireContent(ctx, t, driver, "sized.txt", "old")
		requireNoTempFiles(t, dir)
	})

	t.Run("a key cannot be a file and a directory", func(t *testing.T) {
		require.NoError(t, driver.Put(ctx, "node", strings.NewReader("x"), 1, "text/plain"))
		require.Error(t, driver.Put(ctx, "node/child", strings.NewReader("x"), 1, "text/plain"))

		require.NoError(t, driver.Put(ctx, "parent/child", strings.NewReader("x"), 1, "text/plain"))
		require.Error(t, driver.Put(ctx, "parent", strings.NewReader("x"), 1, "text/plain"))
		_, exists, err := driver.Stat(ctx, "parent")
		require.NoError(t, err)
		require.False(t, exists)
		requireNoTempFiles(t, dir)
	})

	t.Run("canceled context", func(t *testing.T) {
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		require.ErrorIs(t, driver.Put(canceled, "canceled.txt", strings.NewReader("x"), 1, "text/plain"), context.Canceled)
		_, exists, err := driver.Stat(ctx, "canceled.txt")
		require.NoError(t, err)
		require.False(t, exists)
	})
}

func TestDriverConcurrentPut(t *testing.T) {
	dir := t.TempDir()
	driver := newDriver(t, dir)
	key := "race/file.txt"

	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := range 20 {
		wg.Go(func() {
			value := strings.Repeat(fmt.Sprint(i), i+1)
			errs <- driver.Put(t.Context(), key, strings.NewReader(value), int64(len(value)), "text/plain")
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	object, exists, err := driver.Get(t.Context(), key)
	require.NoError(t, err)
	require.True(t, exists)
	body, err := io.ReadAll(object.Body)
	require.NoError(t, err)
	require.NoError(t, object.Body.Close())
	require.Equal(t, sha256Hex(string(body)), object.ETag)
	requireNoTempFiles(t, dir)
}

func newDriver(t *testing.T, dir string) *disk.Driver {
	t.Helper()
	driver, err := disk.NewDriver(dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, driver.Close())
	})
	return driver
}

func requireContent(ctx context.Context, t *testing.T, driver *disk.Driver, key, expected string) {
	t.Helper()
	value, exists, err := filestore.NewClient(driver).GetBytes(ctx, key)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, expected, string(value))
}

func requireNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, filepath.WalkDir(dir, func(name string, _ os.DirEntry, err error) error {
		require.NotContains(t, filepath.Base(name), ".enjin-tmp-")
		return err
	}))
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
