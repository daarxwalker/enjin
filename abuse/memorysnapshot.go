package abuse

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"
)

type memorySnapshot struct {
	Counters   []snapshotCounter
	Sets       []snapshotSet
	Penalties  []snapshotPenalty
	Marks      []snapshotMark
	MemberSalt uint32
}

type snapshotCounter struct {
	Key     Key
	Slot    int64
	Window  int64
	Touched int64
	Current int
	Prev    int
}

type snapshotSet struct {
	Key     Key
	Members []snapshotMember
	Window  int64
	Touched int64
}

type snapshotMember struct {
	Hash uint32
	Seen uint32
}

type snapshotPenalty struct {
	Key     Key
	Penalty Penalty
	Expires int64
	Touched int64
}

type snapshotMark struct {
	Key     Key
	Expires int64
	Touched int64
	Value   bool
}

func (s *MemoryStore) MarshalBinary() ([]byte, error) {
	return s.marshalAt(time.Now())
}

func (s *MemoryStore) UnmarshalBinary(data []byte) error {
	return s.unmarshalAt(data, time.Now())
}

func (s *MemoryStore) marshalAt(now time.Time) ([]byte, error) {
	nowNanos := now.UnixNano()
	snapshot := memorySnapshot{MemberSalt: s.memberSalt.Load()}
	for i := range s.shards {
		shard := &s.shards[i]
		shard.mu.Lock()
		for key, counter := range shard.counters {
			if !counter.expiredAt(nowNanos) {
				snapshot.Counters = append(snapshot.Counters, snapshotCounter{key, counter.slot, counter.window, counter.touched, counter.current, counter.prev})
			}
		}
		for key, set := range shard.sets {
			if set.expiredAt(nowNanos) {
				continue
			}
			members := make([]snapshotMember, len(set.members))
			for i, member := range set.members {
				members[i] = snapshotMember{member.hash, member.seen}
			}
			snapshot.Sets = append(snapshot.Sets, snapshotSet{key, members, set.window, set.touched})
		}
		for key, penalty := range shard.penalties {
			if !penalty.expiredAt(nowNanos) {
				snapshot.Penalties = append(snapshot.Penalties, snapshotPenalty{key, penalty.penalty, penalty.expires, penalty.touched})
			}
		}
		for key, mark := range shard.marks {
			if !mark.expiredAt(nowNanos) {
				snapshot.Marks = append(snapshot.Marks, snapshotMark{key, mark.expires, mark.touched, mark.value})
			}
		}
		shard.mu.Unlock()
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(snapshot); err != nil {
		return nil, fmt.Errorf("encode abuse snapshot failed: %w", err)
	}
	return buf.Bytes(), nil
}

func (s *MemoryStore) unmarshalAt(data []byte, now time.Time) error {
	var snapshot memorySnapshot
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&snapshot); err != nil {
		return fmt.Errorf("decode abuse snapshot failed: %w", err)
	}
	nowNanos := now.UnixNano()
	s.memberSalt.Store(snapshot.MemberSalt)
	s.restoreCounters(snapshot.Counters, nowNanos)
	s.restoreSets(snapshot.Sets, nowNanos)
	s.restorePenalties(snapshot.Penalties, nowNanos)
	s.restoreMarks(snapshot.Marks, nowNanos)
	return nil
}

func (s *MemoryStore) restoreCounters(counters []snapshotCounter, nowNanos int64) {
	for _, counter := range counters {
		entry := memoryCounter{slot: counter.Slot, window: counter.Window, touched: counter.Touched, current: counter.Current, prev: counter.Prev}
		if entry.expiredAt(nowNanos) {
			continue
		}
		s.restoreEntry(counter.Key, func(shard *memoryShard) {
			if _, ok := shard.counters[counter.Key]; !ok {
				evictOldest(shard.counters, s.calcShardCapacity(s.maxCounters), nowNanos)
			}
			shard.counters[counter.Key] = entry
		})
	}
}

func (s *MemoryStore) restoreSets(sets []snapshotSet, nowNanos int64) {
	for _, set := range sets {
		entry := &memorySet{members: make([]memoryMember, len(set.Members)), window: set.Window, touched: set.Touched}
		for i, member := range set.Members {
			entry.members[i] = memoryMember{hash: member.Hash, seen: member.Seen}
		}
		if entry.expiredAt(nowNanos) {
			continue
		}
		s.restoreEntry(set.Key, func(shard *memoryShard) {
			if _, ok := shard.sets[set.Key]; !ok {
				evictOldest(shard.sets, s.calcShardCapacity(s.maxSets), nowNanos)
			}
			shard.sets[set.Key] = entry
		})
	}
}

func (s *MemoryStore) restorePenalties(penalties []snapshotPenalty, nowNanos int64) {
	for _, penalty := range penalties {
		entry := memoryPenalty{penalty: penalty.Penalty, expires: penalty.Expires, touched: penalty.Touched}
		if entry.expiredAt(nowNanos) {
			continue
		}
		s.restoreEntry(penalty.Key, func(shard *memoryShard) {
			if _, ok := shard.penalties[penalty.Key]; !ok {
				evictOldest(shard.penalties, s.calcShardCapacity(s.maxCounters), nowNanos)
			}
			shard.penalties[penalty.Key] = entry
		})
	}
}

func (s *MemoryStore) restoreMarks(marks []snapshotMark, nowNanos int64) {
	for _, mark := range marks {
		entry := memoryMark{expires: mark.Expires, touched: mark.Touched, value: mark.Value}
		if entry.expiredAt(nowNanos) {
			continue
		}
		s.restoreEntry(mark.Key, func(shard *memoryShard) {
			if _, ok := shard.marks[mark.Key]; !ok {
				evictOldest(shard.marks, s.calcShardCapacity(s.maxCounters), nowNanos)
			}
			shard.marks[mark.Key] = entry
		})
	}
}

func (s *MemoryStore) restoreEntry(key Key, restore func(*memoryShard)) {
	shard := s.selectShard(key)
	shard.mu.Lock()
	restore(shard)
	shard.mu.Unlock()
}
