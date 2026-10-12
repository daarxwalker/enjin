package filestore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeDriver struct {
	values       map[string][]byte
	contentTypes map[string]string
	buckets      []string
}

func newFakeDriver() *fakeDriver {
	return &fakeDriver{
		buckets:      make([]string, 1),
		values:       make(map[string][]byte),
		contentTypes: make(map[string]string),
	}
}

func (d *fakeDriver) HasBucket(_ context.Context, bucket string) (bool, error) {
	if slices.Contains(d.buckets, bucket) {
		return true, nil
	}
	return false, fmt.Errorf("bucket %s missing", bucket)
}

func (d *fakeDriver) List(_ context.Context, prefix string) ([]File, error) {
	var files []File
	for key := range d.values {
		if strings.HasPrefix(key, prefix) {
			files = append(files, d.file(key))
		}
	}
	return files, nil
}

func (d *fakeDriver) Stat(_ context.Context, key string) (File, bool, error) {
	if _, exists := d.values[key]; !exists {
		return File{}, false, nil
	}
	return d.file(key), true, nil
}

func (d *fakeDriver) Get(_ context.Context, key string) (*Object, bool, error) {
	value, exists := d.values[key]
	if !exists {
		return nil, false, nil
	}
	return &Object{Body: io.NopCloser(bytes.NewReader(value)), File: d.file(key)}, true, nil
}

func (d *fakeDriver) Put(_ context.Context, key string, body io.Reader, _ int64, contentType string) error {
	value, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	d.values[key] = value
	d.contentTypes[key] = contentType
	return nil
}

func (d *fakeDriver) Delete(_ context.Context, key string) error {
	delete(d.values, key)
	delete(d.contentTypes, key)
	return nil
}

func (d *fakeDriver) file(key string) File {
	return File{
		Key:         key,
		ContentType: d.contentTypes[key],
		Size:        int64(len(d.values[key])),
	}
}

func TestClient(t *testing.T) {
	ctx := context.Background()
	png := append([]byte("\x89PNG\x0D\x0A\x1A\x0A"), bytes.Repeat([]byte{0}, 1024)...)

	t.Run(
		"put detects missing content type", func(t *testing.T) {
			driver := newFakeDriver()
			client := NewClient(driver)
			require.NoError(t, client.PutBytes(ctx, "image.png", png, ""))
			require.Equal(t, "image/png", driver.contentTypes["image.png"])
			require.Equal(t, png, driver.values["image.png"])
		},
	)

	t.Run(
		"put detects content type of short body", func(t *testing.T) {
			driver := newFakeDriver()
			client := NewClient(driver)
			require.NoError(t, client.PutBytes(ctx, "note.txt", []byte("hello"), ""))
			require.Equal(t, "text/plain; charset=utf-8", driver.contentTypes["note.txt"])
			require.Equal(t, []byte("hello"), driver.values["note.txt"])
		},
	)

	t.Run(
		"put keeps given content type", func(t *testing.T) {
			driver := newFakeDriver()
			client := NewClient(driver)
			require.NoError(t, client.Put(ctx, "image.bin", bytes.NewReader(png), int64(len(png)), "application/x-custom"))
			require.Equal(t, "application/x-custom", driver.contentTypes["image.bin"])
			require.Equal(t, png, driver.values["image.bin"])
		},
	)

	t.Run(
		"get bytes", func(t *testing.T) {
			client := NewClient(newFakeDriver())
			require.NoError(t, client.PutBytes(ctx, "note.txt", []byte("hello"), "text/plain"))

			value, exists, err := client.GetBytes(ctx, "note.txt")
			require.NoError(t, err)
			require.True(t, exists)
			require.Equal(t, []byte("hello"), value)
		},
	)

	t.Run(
		"get bytes missing", func(t *testing.T) {
			client := NewClient(newFakeDriver())
			value, exists, err := client.GetBytes(ctx, "missing.txt")
			require.NoError(t, err)
			require.False(t, exists)
			require.Nil(t, value)
		},
	)
}
