package lock

import "time"

// expiresIn turns a TTL into microseconds for SQL, which reads 0 as NULL: a
// lock that never expires, the same as Redis SETNX without expiry. A positive
// TTL under a microsecond stays positive, so it never turns into "forever".
func expiresIn(ttl time.Duration) int64 {
	if ttl <= 0 {
		return 0
	}
	return max(ttl.Microseconds(), 1)
}
