package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Client struct {
	driver Driver
}

func NewClient(driver Driver) *Client {
	return &Client{
		driver: driver,
	}
}

func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	var zero bool

	exists, err := c.driver.Exists(ctx, key)
	if err != nil {
		return zero, fmt.Errorf("check existence in cache failed: %w", err)
	}
	return exists, nil
}

func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	var zero time.Duration

	ttl, _, err := c.driver.TTL(ctx, key)
	if err != nil {
		return zero, fmt.Errorf("get ttl from cache failed: %w", err)
	}
	return ttl, nil
}

func (c *Client) GetString(ctx context.Context, key string) (string, error) {
	var zero string

	value, _, err := c.driver.Get(ctx, key)
	if err != nil {
		return zero, fmt.Errorf("get string from cache failed: %w", err)
	}
	return string(value), nil
}

func (c *Client) GetJSON(ctx context.Context, key string, target any) (bool, error) {
	var zero bool

	value, exists, err := c.driver.Get(ctx, key)
	if err != nil {
		return zero, fmt.Errorf("get serialized value from cache failed: %w", err)
	}
	if !exists {
		return false, nil
	}
	if unmarshalErr := json.Unmarshal(value, target); unmarshalErr != nil {
		return true, fmt.Errorf("deserialize cache value failed: %w", unmarshalErr)
	}
	return true, nil
}

func (c *Client) TakeString(ctx context.Context, key string) (string, error) {
	var zero string

	value, _, err := c.driver.Take(ctx, key)
	if err != nil {
		return zero, fmt.Errorf("take string from cache failed: %w", err)
	}
	return string(value), nil
}

func (c *Client) TakeJSON(ctx context.Context, key string, target any) (bool, error) {
	var zero bool

	value, exists, err := c.driver.Take(ctx, key)
	if err != nil {
		return zero, fmt.Errorf("take serialized value from cache failed: %w", err)
	}
	if !exists {
		return false, nil
	}
	if unmarshalErr := json.Unmarshal(value, target); unmarshalErr != nil {
		return true, fmt.Errorf("deserialize cache value failed: %w", unmarshalErr)
	}
	return true, nil
}

func (c *Client) SetString(ctx context.Context, key string, value string, ttl time.Duration) error {
	err := c.driver.Set(ctx, key, []byte(value), ttl)
	if err != nil {
		return fmt.Errorf("set string to cache failed: %w", err)
	}
	return nil
}

func (c *Client) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	valueBytes, marshalErr := json.Marshal(value)
	if marshalErr != nil {
		return fmt.Errorf("serialize value for cache failed: %w", marshalErr)
	}
	if err := c.driver.Set(ctx, key, valueBytes, ttl); err != nil {
		return fmt.Errorf("set serialized value to cache failed: %w", err)
	}
	return nil
}

func (c *Client) UpdateExpiration(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	var zero bool

	updated, err := c.driver.UpdateExpiration(ctx, key, ttl)
	if err != nil {
		return zero, fmt.Errorf("update expiration in cache failed: %w", err)
	}
	return updated, nil
}

func (c *Client) Increment(ctx context.Context, key string, delta int, ttl time.Duration) (int, error) {
	var zero int

	value, err := c.driver.Increment(ctx, key, int64(delta), ttl)
	if err != nil {
		return zero, fmt.Errorf("increment value in cache failed: %w", err)
	}
	return int(value), nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	err := c.driver.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("delete value in cache failed: %w", err)
	}
	return nil
}
