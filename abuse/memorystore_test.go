package abuse

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMemoryStore(t *testing.T) {
	start := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	key := Key{Namespace: "policy|rule", Value: "203.0.113.7/32"}

	t.Run("sliding window weights previous window", func(t *testing.T) {
		store := NewMemoryStore(t.Context())
		for range 10 {
			_, err := store.HitCounter(key, time.Minute, start)
			require.NoError(t, err)
		}
		count, err := store.HitCounter(key, time.Minute, start.Add(time.Minute+15*time.Second))
		require.NoError(t, err)
		require.Equal(t, 1+7, count)
		count, err = store.HitCounter(key, time.Minute, start.Add(3*time.Minute))
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})

	t.Run("distinct keeps at most limit plus one members", func(t *testing.T) {
		store := NewMemoryStore(t.Context())
		var count int
		for i := range 50 {
			var err error
			count, err = store.CountDistinct(key, fmt.Sprintf("product-%d", i), 3, time.Hour, start)
			require.NoError(t, err)
		}
		require.Equal(t, 4, count)
		require.Len(t, store.selectShard(key).sets[key].members, 4)
	})

	t.Run("penalty and mark expire after ttl", func(t *testing.T) {
		store := NewMemoryStore(t.Context())
		require.NoError(t, store.SavePenalty(key, Penalty{Level: 2, RaisedAt: start}, time.Hour, start))
		require.NoError(t, store.SaveMark(key, true, time.Hour, start))
		penalty, err := store.LoadPenalty(key, start.Add(30*time.Minute))
		require.NoError(t, err)
		require.Equal(t, uint8(2), penalty.Level)
		value, found, err := store.LoadMark(key, start.Add(30*time.Minute))
		require.NoError(t, err)
		require.True(t, found)
		require.True(t, value)
		penalty, err = store.LoadPenalty(key, start.Add(2*time.Hour))
		require.NoError(t, err)
		require.Zero(t, penalty)
		_, found, err = store.LoadMark(key, start.Add(2*time.Hour))
		require.NoError(t, err)
		require.False(t, found)
	})

	t.Run("full shard evicts before adding a new key", func(t *testing.T) {
		store := NewMemoryStore(t.Context(), WithMemoryCapacity(memoryShardCount, memoryShardCount))
		for i := range 10_000 {
			_, err := store.HitCounter(Key{Namespace: "n", Value: fmt.Sprint(i)}, time.Minute, start.Add(time.Duration(i)))
			require.NoError(t, err)
		}
		for i := range store.shards {
			require.LessOrEqual(t, len(store.shards[i].counters), 1)
		}
	})

	t.Run("cleanup removes expired entries", func(t *testing.T) {
		store := NewMemoryStore(t.Context())
		_, err := store.HitCounter(key, time.Minute, start)
		require.NoError(t, err)
		_, err = store.CountDistinct(key, "a", 5, time.Minute, start)
		require.NoError(t, err)
		require.NoError(t, store.SaveMark(key, true, time.Minute, start))
		store.removeExpired(start.Add(time.Hour))
		shard := store.selectShard(key)
		require.Empty(t, shard.counters)
		require.Empty(t, shard.sets)
		require.Empty(t, shard.marks)
	})

	t.Run("concurrent hits are counted exactly", func(t *testing.T) {
		store := NewMemoryStore(t.Context())
		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() {
				for range 1000 {
					_, _ = store.HitCounter(key, time.Hour, start)
				}
			})
		}
		wg.Wait()
		count, err := store.HitCounter(key, time.Hour, start)
		require.NoError(t, err)
		require.Equal(t, 8001, count)
	})

	t.Run("two hundred thousand keys stay under memory budget", func(t *testing.T) {
		if testing.Short() {
			t.Skip("memory budget check is slow")
		}
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		store := NewMemoryStore(t.Context())
		for i := range memoryDefaultCounters {
			ip := fmt.Sprintf("198.%d.%d.%d/32", i>>16&255, i>>8&255, i&255)
			_, err := store.HitCounter(Key{Namespace: "product-page|rate-ip-60-1m0s", Value: ip}, time.Minute, start)
			require.NoError(t, err)
		}
		for i := range memoryDefaultSets {
			setKey := Key{Namespace: "product-page|distinct", Value: fmt.Sprint(i)}
			for j := range 301 {
				_, err := store.CountDistinct(setKey, fmt.Sprintf("product-%d", j), 300, time.Hour, start)
				require.NoError(t, err)
			}
		}
		runtime.GC()
		runtime.ReadMemStats(&after)
		used := after.HeapAlloc - before.HeapAlloc
		t.Logf("heap for %d counters and %d sets: %.1f MB", memoryDefaultCounters, memoryDefaultSets, float64(used)/1024/1024)
		require.Less(t, used, uint64(64*1024*1024))
		runtime.KeepAlive(store)
	})
}
