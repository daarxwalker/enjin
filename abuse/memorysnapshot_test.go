package abuse

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMemorySnapshot(t *testing.T) {
	start := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	key := Key{Namespace: "policy|rule", Value: "203.0.113.7/32"}

	t.Run("restores state into a new store", func(t *testing.T) {
		source := NewMemoryStore(t.Context())
		for range 5 {
			_, err := source.HitCounter(key, time.Minute, start)
			require.NoError(t, err)
		}
		for _, member := range []string{"a", "b", "c"} {
			_, err := source.CountDistinct(key, member, 10, time.Hour, start)
			require.NoError(t, err)
		}
		penalty := Penalty{RaisedAt: start, BlockedUntil: start.Add(time.Hour), Level: 2}
		require.NoError(t, source.SavePenalty(key, penalty, time.Hour, start))
		require.NoError(t, source.SaveMark(key, true, time.Hour, start))

		data, err := source.marshalAt(start)
		require.NoError(t, err)
		restored := NewMemoryStore(t.Context())
		require.NoError(t, restored.unmarshalAt(data, start))

		now := start.Add(time.Second)
		count, err := restored.HitCounter(key, time.Minute, now)
		require.NoError(t, err)
		require.Equal(t, 6, count)
		distinct, err := restored.CountDistinct(key, "a", 10, time.Hour, now)
		require.NoError(t, err)
		require.Equal(t, 3, distinct)
		distinct, err = restored.CountDistinct(key, "d", 10, time.Hour, now)
		require.NoError(t, err)
		require.Equal(t, 4, distinct)
		loaded, err := restored.LoadPenalty(key, now)
		require.NoError(t, err)
		require.Equal(t, penalty.Level, loaded.Level)
		require.True(t, penalty.BlockedUntil.Equal(loaded.BlockedUntil))
		value, found, err := restored.LoadMark(key, now)
		require.NoError(t, err)
		require.True(t, found)
		require.True(t, value)
	})

	t.Run("skips expired entries", func(t *testing.T) {
		source := NewMemoryStore(t.Context())
		_, err := source.HitCounter(key, time.Minute, start)
		require.NoError(t, err)
		require.NoError(t, source.SavePenalty(key, Penalty{Level: 1}, time.Minute, start))
		require.NoError(t, source.SaveMark(key, true, time.Minute, start))

		later := start.Add(time.Hour)
		data, err := source.marshalAt(later)
		require.NoError(t, err)
		restored := NewMemoryStore(t.Context())
		require.NoError(t, restored.unmarshalAt(data, later))

		count, err := restored.HitCounter(key, time.Minute, later)
		require.NoError(t, err)
		require.Equal(t, 1, count)
		loaded, err := restored.LoadPenalty(key, later)
		require.NoError(t, err)
		require.Zero(t, loaded.Level)
		_, found, err := restored.LoadMark(key, later)
		require.NoError(t, err)
		require.False(t, found)
	})

	t.Run("restore respects store capacity", func(t *testing.T) {
		source := NewMemoryStore(t.Context())
		for i := range 1000 {
			_, err := source.HitCounter(Key{Namespace: "policy|rule", Value: strconv.Itoa(i)}, time.Minute, start)
			require.NoError(t, err)
		}
		data, err := source.marshalAt(start)
		require.NoError(t, err)
		restored := NewMemoryStore(t.Context(), WithMemoryCapacity(memoryShardCount, memoryShardCount))
		require.NoError(t, restored.unmarshalAt(data, start))
		var counters int
		for i := range restored.shards {
			counters += len(restored.shards[i].counters)
		}
		require.LessOrEqual(t, counters, memoryShardCount)
	})

	t.Run("restore runs concurrently with live traffic", func(t *testing.T) {
		source := NewMemoryStore(t.Context())
		_, err := source.CountDistinct(key, "a", 10, time.Hour, start)
		require.NoError(t, err)
		data, err := source.marshalAt(start)
		require.NoError(t, err)
		restored := NewMemoryStore(t.Context())
		var wg sync.WaitGroup
		wg.Go(func() {
			for range 100 {
				_, _ = restored.CountDistinct(key, "b", 10, time.Hour, start)
			}
		})
		require.NoError(t, restored.unmarshalAt(data, start))
		wg.Wait()
	})

	t.Run("rejects corrupted data", func(t *testing.T) {
		require.Error(t, NewMemoryStore(t.Context()).UnmarshalBinary([]byte("not gob")))
	})
}
