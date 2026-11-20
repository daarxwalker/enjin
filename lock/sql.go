package lock

const (
	// An expired lock is taken over by the new owner; expiry is measured by
	// the database clock, so instances whose clocks drift apart still agree.
	insertLockSQL = `INSERT INTO enjin_lock AS held (key, token, expires_at)
VALUES ($1, $2, now() + nullif($3::bigint, 0) * interval '1 microsecond')
ON CONFLICT (key) DO UPDATE SET token = EXCLUDED.token, expires_at = EXCLUDED.expires_at
WHERE held.expires_at IS NOT NULL AND held.expires_at <= now()`

	deleteLockSQL = `DELETE FROM enjin_lock
WHERE key = $1 AND token = $2 AND (expires_at IS NULL OR expires_at > now())`
)
