package csrf

const (
	// Concurrent Adds to one key would each trim before seeing the others'
	// inserts and keep more tokens than the limit. The first argument
	// ("csrf" in ASCII) keeps these locks apart from the application's own
	// advisory locks.
	lockKeySQL = `SELECT pg_advisory_xact_lock(1668510310, hashtext($1))`

	deleteExpiredEntriesSQL = `DELETE FROM enjin_csrf WHERE key = $1 AND expires_at <= $2`

	upsertEntrySQL = `INSERT INTO enjin_csrf (key, token, expires_at) VALUES ($1, $2, $3)
ON CONFLICT (key, token) DO UPDATE SET expires_at = EXCLUDED.expires_at`

	// The order matches a Redis sorted set: by expiry, then by token.
	deleteEntriesOverLimitSQL = `DELETE FROM enjin_csrf WHERE key = $1 AND token NOT IN (
    SELECT token FROM enjin_csrf WHERE key = $1 ORDER BY expires_at DESC, token DESC LIMIT $2
)`

	takeEntrySQL = `DELETE FROM enjin_csrf WHERE key = $1 AND token = $2 RETURNING expires_at`
)
