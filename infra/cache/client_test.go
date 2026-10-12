package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/infra/cache"
	"github.com/daarxwalker/enjin/internal/testredis"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type item struct {
	Name string `json:"name"`
}

func TestClient(t *testing.T) {
	ctx := context.Background()
	rdb := testredis.New(t)
	client := cache.NewClient(cache.NewRedisDriver(rdb))

	t.Run(
		"miss", func(t *testing.T) {
			exists, err := client.Exists(ctx, "missing")
			require.NoError(t, err)
			require.False(t, exists)
			value, err := client.GetString(ctx, "missing")
			require.NoError(t, err)
			require.Empty(t, value)
			ttl, err := client.TTL(ctx, "missing")
			require.NoError(t, err)
			require.Zero(t, ttl)
			target := item{Name: "untouched"}
			found, err := client.GetJSON(ctx, "missing", &target)
			require.NoError(t, err)
			require.False(t, found)
			require.Equal(t, "untouched", target.Name)
		},
	)

	t.Run(
		"string", func(t *testing.T) {
			require.NoError(t, client.SetString(ctx, "string", "value", time.Minute))
			value, err := client.GetString(ctx, "string")
			require.NoError(t, err)
			require.Equal(t, "value", value)
			exists, err := client.Exists(ctx, "string")
			require.NoError(t, err)
			require.True(t, exists)
			ttl, err := client.TTL(ctx, "string")
			require.NoError(t, err)
			require.Equal(t, time.Minute, ttl)
			require.NoError(t, client.Delete(ctx, "string"))
			exists, err = client.Exists(ctx, "string")
			require.NoError(t, err)
			require.False(t, exists)
		},
	)

	t.Run(
		"json", func(t *testing.T) {
			require.NoError(t, client.SetJSON(ctx, "json", item{Name: "šampon"}, time.Minute))
			var got item
			found, err := client.GetJSON(ctx, "json", &got)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, item{Name: "šampon"}, got)
			require.NoError(t, rdb.Set(ctx, "broken", "{", 0).Err())
			found, err = client.GetJSON(ctx, "broken", &got)
			require.Error(t, err)
			require.True(t, found)
		},
	)

	t.Run(
		"without expiry", func(t *testing.T) {
			require.NoError(t, client.SetString(ctx, "forever", "value", 0))
			ttl, err := client.TTL(ctx, "forever")
			require.NoError(t, err)
			require.Zero(t, ttl)
		},
	)

	t.Run(
		"increment keeps the first ttl", func(t *testing.T) {
			count, err := client.Increment(ctx, "counter", 1, time.Hour)
			require.NoError(t, err)
			require.Equal(t, 1, count)
			count, err = client.Increment(ctx, "counter", 2, time.Minute)
			require.NoError(t, err)
			require.Equal(t, 3, count)
			ttl, err := client.TTL(ctx, "counter")
			require.NoError(t, err)
			require.Equal(t, time.Hour, ttl)
			count, err = client.Increment(ctx, "counter", -1, time.Minute)
			require.NoError(t, err)
			require.Equal(t, 2, count)
		},
	)

	t.Run(
		"negative increment does not create a key", func(t *testing.T) {
			count, err := client.Increment(ctx, "absent", -1, time.Hour)
			require.NoError(t, err)
			require.Zero(t, count)
			exists, err := client.Exists(ctx, "absent")
			require.NoError(t, err)
			require.False(t, exists)
		},
	)

	t.Run(
		"increment without ttl keeps the key", func(t *testing.T) {
			count, err := client.Increment(ctx, "no-ttl", 1, 0)
			require.NoError(t, err)
			require.Equal(t, 1, count)
			exists, err := client.Exists(ctx, "no-ttl")
			require.NoError(t, err)
			require.True(t, exists)
		},
	)

	t.Run(
		"take", func(t *testing.T) {
			require.NoError(t, client.SetString(ctx, "take:string", "value", time.Minute))
			value, err := client.TakeString(ctx, "take:string")
			require.NoError(t, err)
			require.Equal(t, "value", value)
			value, err = client.TakeString(ctx, "take:string")
			require.NoError(t, err)
			require.Empty(t, value)

			require.NoError(t, client.SetJSON(ctx, "take:json", item{Name: "a"}, time.Minute))
			var got item
			found, err := client.TakeJSON(ctx, "take:json", &got)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, "a", got.Name)
			found, err = client.TakeJSON(ctx, "take:json", &got)
			require.NoError(t, err)
			require.False(t, found)
		},
	)

	t.Run(
		"delete by pattern", func(t *testing.T) {
			for _, key := range []string{"scan:a", "scan:b", "scan:c", "keep"} {
				require.NoError(t, client.SetString(ctx, key, "x", time.Minute))
			}
			require.NoError(t, cache.NewRedisDriver(rdb).DeleteByPattern(ctx, "scan:*", 1))
			keys, err := rdb.Keys(ctx, "*").Result()
			require.NoError(t, err)
			require.NotContains(t, keys, "scan:a")
			require.NotContains(t, keys, "scan:b")
			require.NotContains(t, keys, "scan:c")
			require.Contains(t, keys, "keep")
		},
	)
}

func TestClientOutage(t *testing.T) {
	ctx := context.Background()
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = rdb.Close() })
	client := cache.NewClient(cache.NewRedisDriver(rdb))
	server.Close()

	_, err := client.Exists(ctx, "key")
	require.Error(t, err)
	_, err = client.TTL(ctx, "key")
	require.Error(t, err)
	_, err = client.GetString(ctx, "key")
	require.Error(t, err)
	found, err := client.GetJSON(ctx, "key", new(item))
	require.Error(t, err)
	require.False(t, found)
	_, err = client.TakeString(ctx, "key")
	require.Error(t, err)
	found, err = client.TakeJSON(ctx, "key", new(item))
	require.Error(t, err)
	require.False(t, found)
	require.Error(t, client.SetString(ctx, "key", "x", time.Minute))
	require.Error(t, client.SetJSON(ctx, "key", item{}, time.Minute))
	_, err = client.Increment(ctx, "key", 1, time.Minute)
	require.Error(t, err)
	require.Error(t, client.Delete(ctx, "key"))
}
