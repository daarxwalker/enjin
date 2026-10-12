package abuse

import (
	"math"
	"time"
)

func evictOldest[V touchable](entries map[Key]V, capacity int, now int64) {
	if len(entries) < capacity {
		return
	}
	var (
		oldestKey Key
		oldest    int64
		sampled   int
	)
	for key, entry := range entries {
		if entry.expiredAt(now) {
			delete(entries, key)
			return
		}
		if sampled == 0 || entry.touchedAt() < oldest {
			oldestKey, oldest = key, entry.touchedAt()
		}
		sampled++
		if sampled == memoryEvictionSampleLen {
			break
		}
	}
	delete(entries, oldestKey)
}

func removeExpiredEntries[V touchable](entries map[Key]V, now int64) {
	for key, entry := range entries {
		if entry.expiredAt(now) {
			delete(entries, key)
		}
	}
}

func timeToUnixSeconds(t time.Time) uint32 {
	seconds := t.Unix()
	if seconds <= 0 {
		return 0
	}
	if seconds >= math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(seconds)
}
