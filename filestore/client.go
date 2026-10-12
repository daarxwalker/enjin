package filestore

import (
	"bytes"
	"context"
	"fmt"
	"io"
)

const sniffLen = 512

var _ Commander = (*Client)(nil)

type Client struct {
	driver Driver
}

func NewClient(driver Driver) *Client {
	return &Client{
		driver: driver,
	}
}

func (c *Client) List(ctx context.Context, prefix string) ([]File, error) {
	files, err := c.driver.List(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("list files in filestore failed: %w", err)
	}
	return files, nil
}

func (c *Client) Stat(ctx context.Context, key string) (File, bool, error) {
	var zero File
	file, exists, err := c.driver.Stat(ctx, key)
	if err != nil {
		return zero, false, fmt.Errorf("stat file in filestore failed: %w", err)
	}
	return file, exists, nil
}

func (c *Client) Get(ctx context.Context, key string) (*Object, bool, error) {
	object, exists, err := c.driver.Get(ctx, key)
	if err != nil {
		return nil, false, fmt.Errorf("get file from filestore failed: %w", err)
	}
	return object, exists, nil
}

func (c *Client) GetBytes(ctx context.Context, key string) ([]byte, bool, error) {
	object, exists, err := c.Get(ctx, key)
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}
	value, readErr := io.ReadAll(object.Body)
	closeErr := object.Body.Close()
	if readErr != nil {
		return nil, false, fmt.Errorf("read file from filestore failed: %w", readErr)
	}
	if closeErr != nil {
		return nil, false, fmt.Errorf("close file from filestore failed: %w", closeErr)
	}
	return value, true, nil
}

func (c *Client) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	if contentType == "" {
		ct, sniffed, err := detectContentType(body)
		if err != nil {
			return fmt.Errorf("detect content type failed: %w", err)
		}
		contentType = ct
		body = sniffed
	}
	if err := c.driver.Put(ctx, key, body, size, contentType); err != nil {
		return fmt.Errorf("put file to filestore failed: %w", err)
	}
	return nil
}

func (c *Client) PutBytes(ctx context.Context, key string, value []byte, contentType string) error {
	return c.Put(ctx, key, bytes.NewReader(value), int64(len(value)), contentType)
}

func (c *Client) Delete(ctx context.Context, key string) error {
	if err := c.driver.Delete(ctx, key); err != nil {
		return fmt.Errorf("delete file in filestore failed: %w", err)
	}
	return nil
}
