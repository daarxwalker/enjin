package abuse

import (
	"context"
	"hash/maphash"
	"sync"
	"sync/atomic"
	"time"
)

const (
	memoryShardCount        = 64
	memoryDefaultCounters   = 200_000
	memoryDefaultSets       = 10_000
	memoryCleanupInterval   = time.Minute
	memoryEvictionSampleLen = 8
	memoryMinMembers        = 8
)

type MemoryStore struct {
	shards      [memoryShardCount]memoryShard
	seed        maphash.Seed
	maxCounters int
	maxSets     int
	memberSalt  atomic.Uint32
}

type memoryShard struct {
	counters  map[Key]memoryCounter
	sets      map[Key]*memorySet
	penalties map[Key]memoryPenalty
	marks     map[Key]memoryMark
	mu        sync.Mutex
}

type memoryCounter struct {
	slot    int64
	window  int64
	touched int64
	current int
	prev    int
}

type memorySet struct {
	members []memoryMember
	window  int64
	touched int64
}

type memoryMember struct {
	hash uint32
	seen uint32
}

type memoryPenalty struct {
	penalty Penalty
	expires int64
	touched int64
}

type memoryMark struct {
	expires int64
	touched int64
	value   bool
}

func NewMemoryStore(ctx context.Context, options ...MemoryStoreOption) *MemoryStore {
	s := &MemoryStore{
		seed:        maphash.MakeSeed(),
		maxCounters: memoryDefaultCounters,
		maxSets:     memoryDefaultSets,
	}
	s.memberSalt.Store(createMemberSalt())
	for _, option := range options {
		option(s)
	}
	for i := range s.shards {
		s.shards[i].counters = make(map[Key]memoryCounter)
		s.shards[i].sets = make(map[Key]*memorySet)
		s.shards[i].penalties = make(map[Key]memoryPenalty)
		s.shards[i].marks = make(map[Key]memoryMark)
	}
	go s.removeExpiredPeriodically(ctx)
	return s
}

func (s *MemoryStore) HitCounter(key Key, window time.Duration, now time.Time) (int, error) {
	if window <= 0 {
		return 0, ErrInvalidWindow
	}
	shard := s.selectShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	nowNanos := now.UnixNano()
	windowNanos := int64(window)
	slot := nowNanos / windowNanos
	counter, ok := shard.counters[key]
	if !ok {
		evictOldest(shard.counters, s.calcShardCapacity(s.maxCounters), nowNanos)
		counter = memoryCounter{slot: slot, window: windowNanos}
	}
	switch counter.slot {
	case slot:
	case slot - 1:
		counter.prev, counter.current = counter.current, 0
	default:
		counter.prev, counter.current = 0, 0
	}
	counter.slot = slot
	counter.touched = nowNanos
	counter.current++
	shard.counters[key] = counter
	elapsed := float64(nowNanos-slot*windowNanos) / float64(windowNanos)
	return counter.current + int(float64(counter.prev)*(1-elapsed)), nil
}

func (s *MemoryStore) CountDistinct(key Key, member string, limit int, window time.Duration, now time.Time) (int, error) {
	if window <= 0 {
		return 0, ErrInvalidWindow
	}
	shard := s.selectShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	nowNanos := now.UnixNano()
	set, ok := shard.sets[key]
	if !ok {
		evictOldest(shard.sets, s.calcShardCapacity(s.maxSets), nowNanos)
		set = &memorySet{window: int64(window)}
		shard.sets[key] = set
	}
	set.touched = nowNanos
	seen := timeToUnixSeconds(now)
	oldest := timeToUnixSeconds(now.Add(-window))
	hash := s.hashMember(member)
	kept := set.members[:0]
	found := false
	for _, entry := range set.members {
		if entry.seen < oldest {
			continue
		}
		if entry.hash == hash {
			entry.seen = seen
			found = true
		}
		kept = append(kept, entry)
	}
	set.members = kept
	if !found && len(set.members) <= limit {
		set.members = s.growMembers(set.members, limit+1)
		set.members = append(set.members, memoryMember{hash: hash, seen: seen})
	}
	return len(set.members), nil
}

func (s *MemoryStore) LoadPenalty(key Key, now time.Time) (Penalty, error) {
	shard := s.selectShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	entry, ok := shard.penalties[key]
	if !ok || entry.expiredAt(now.UnixNano()) {
		return Penalty{}, nil
	}
	return entry.penalty, nil
}

func (s *MemoryStore) SavePenalty(key Key, penalty Penalty, ttl time.Duration, now time.Time) error {
	shard := s.selectShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	nowNanos := now.UnixNano()
	if _, ok := shard.penalties[key]; !ok {
		evictOldest(shard.penalties, s.calcShardCapacity(s.maxCounters), nowNanos)
	}
	shard.penalties[key] = memoryPenalty{penalty: penalty, expires: now.Add(ttl).UnixNano(), touched: nowNanos}
	return nil
}

func (s *MemoryStore) LoadMark(key Key, now time.Time) (bool, bool, error) {
	shard := s.selectShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	entry, ok := shard.marks[key]
	if !ok || entry.expiredAt(now.UnixNano()) {
		return false, false, nil
	}
	return entry.value, true, nil
}

func (s *MemoryStore) SaveMark(key Key, value bool, ttl time.Duration, now time.Time) error {
	shard := s.selectShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	nowNanos := now.UnixNano()
	if _, ok := shard.marks[key]; !ok {
		evictOldest(shard.marks, s.calcShardCapacity(s.maxCounters), nowNanos)
	}
	shard.marks[key] = memoryMark{value: value, expires: now.Add(ttl).UnixNano(), touched: nowNanos}
	return nil
}

func (s *MemoryStore) growMembers(members []memoryMember, capacity int) []memoryMember {
	if len(members) < cap(members) {
		return members
	}
	grown := make([]memoryMember, len(members), min(capacity, max(memoryMinMembers, 2*cap(members))))
	copy(grown, members)
	return grown
}

func (s *MemoryStore) hashMember(member string) uint32 {
	const prime = 16777619
	hash := uint32(2166136261) ^ s.memberSalt.Load()
	for i := 0; i < len(member); i++ {
		hash ^= uint32(member[i])
		hash *= prime
	}
	return hash
}

func (s *MemoryStore) selectShard(key Key) *memoryShard {
	return &s.shards[maphash.Comparable(s.seed, key)%memoryShardCount]
}

func (s *MemoryStore) calcShardCapacity(total int) int {
	return max(1, total/memoryShardCount)
}

func (s *MemoryStore) removeExpiredPeriodically(ctx context.Context) {
	ticker := time.NewTicker(memoryCleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.removeExpired(now)
		}
	}
}

func (s *MemoryStore) removeExpired(now time.Time) {
	nowNanos := now.UnixNano()
	for i := range s.shards {
		shard := &s.shards[i]
		shard.mu.Lock()
		removeExpiredEntries(shard.counters, nowNanos)
		removeExpiredEntries(shard.sets, nowNanos)
		removeExpiredEntries(shard.penalties, nowNanos)
		removeExpiredEntries(shard.marks, nowNanos)
		shard.mu.Unlock()
	}
}

func (c memoryCounter) touchedAt() int64 {
	return c.touched
}

func (c memoryCounter) expiredAt(now int64) bool {
	return now >= (c.slot+2)*c.window
}

func (s *memorySet) touchedAt() int64 {
	return s.touched
}

func (s *memorySet) expiredAt(now int64) bool {
	return now-s.touched > s.window
}

func (p memoryPenalty) touchedAt() int64 {
	return p.touched
}

func (p memoryPenalty) expiredAt(now int64) bool {
	return now >= p.expires
}

func (m memoryMark) touchedAt() int64 {
	return m.touched
}

func (m memoryMark) expiredAt(now int64) bool {
	return now >= m.expires
}
