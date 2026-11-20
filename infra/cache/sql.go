package cache

// liveRowSQL measures expiry by the database clock, so instances whose clocks
// drift apart still agree on what has expired.
const liveRowSQL = `(expires_at IS NULL OR expires_at > now())`

const (
	loadValueSQL = `SELECT value FROM enjin_cache WHERE key = $1 AND ` + liveRowSQL

	takeValueSQL = `DELETE FROM enjin_cache WHERE key = $1
RETURNING value, ` + liveRowSQL

	upsertValueSQL = `INSERT INTO enjin_cache (key, value, expires_at)
VALUES ($1, $2, now() + nullif($3::bigint, 0) * interval '1 microsecond')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, expires_at = EXCLUDED.expires_at`

	updateExpirationSQL = `UPDATE enjin_cache
SET expires_at = now() + nullif($2::bigint, 0) * interval '1 microsecond'
WHERE key = $1 AND ` + liveRowSQL

	deleteLiveValueSQL = `DELETE FROM enjin_cache WHERE key = $1 AND ` + liveRowSQL

	deleteValueSQL = `DELETE FROM enjin_cache WHERE key = $1`

	deleteValuesByPatternSQL = `DELETE FROM enjin_cache WHERE key LIKE $1`

	loadExistenceSQL = `SELECT EXISTS (SELECT 1 FROM enjin_cache WHERE key = $1 AND ` + liveRowSQL + `)`

	// A key without expiry has no row here, the same answer as a missing key.
	loadTTLSQL = `SELECT (extract(epoch FROM expires_at - now()) * 1000000)::bigint
FROM enjin_cache WHERE key = $1 AND expires_at > now()`

	// The stored value stays decimal text, so GetString reads the same as on
	// Redis. The TTL applies only to a key without one, as in the Redis script.
	incrementValueSQL = `INSERT INTO enjin_cache AS cache (key, value, expires_at)
VALUES ($1, convert_to($2::bigint::text, 'UTF8'), now() + nullif($3::bigint, 0) * interval '1 microsecond')
ON CONFLICT (key) DO UPDATE SET
    value = CASE WHEN ` + liveStoredRowSQL + `
        THEN convert_to((` + storedIntegerSQL + ` + $2::bigint)::text, 'UTF8')
        ELSE EXCLUDED.value END,
    expires_at = CASE WHEN ` + liveStoredRowSQL + `
        THEN coalesce(cache.expires_at, EXCLUDED.expires_at)
        ELSE EXCLUDED.expires_at END
RETURNING convert_from(value, 'UTF8')::bigint`

	// A negative delta never creates a key, the same as the Redis script, so it
	// is a plain UPDATE: a key deleted meanwhile cannot come back negative.
	decrementValueSQL = `UPDATE enjin_cache AS cache SET
    value = convert_to((` + storedIntegerSQL + ` + $2::bigint)::text, 'UTF8'),
    expires_at = coalesce(cache.expires_at, now() + nullif($3::bigint, 0) * interval '1 microsecond')
WHERE cache.key = $1 AND ` + liveStoredRowSQL + `
RETURNING convert_from(value, 'UTF8')::bigint`
)

const (
	liveStoredRowSQL = `(cache.expires_at IS NULL OR cache.expires_at > now())`

	// A plain ::bigint cast would also accept " 5", which Redis INCRBY
	// rejects; the fallback text makes the cast fail with a clear message.
	storedIntegerSQL = `(CASE WHEN convert_from(cache.value, 'UTF8') ~ '^-?[0-9]+$'
        THEN convert_from(cache.value, 'UTF8') ELSE 'value is not an integer' END)::bigint`
)
