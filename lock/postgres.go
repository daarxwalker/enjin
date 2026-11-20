package lock

import (
	"context"
	"time"

	"github.com/daarxwalker/enjin/infra/db"
)

var _ Driver = (*PostgresDriver)(nil)

// PostgresDriver keeps the locks in the UNLOGGED table enjin_lock, created by
// package infra/db/migrations/keyvalue.
type PostgresDriver struct {
	querier db.Querier
}

func NewPostgresDriver(querier db.Querier) *PostgresDriver {
	return &PostgresDriver{querier: querier}
}

// TryLock with a zero or negative TTL holds the lock until Unlock, the same as
// Redis SETNX without expiry.
func (d *PostgresDriver) TryLock(ctx context.Context, key, token string, ttl time.Duration) (bool, error) {
	tag, err := d.querier.Exec(ctx, insertLockSQL, key, token, expiresIn(ttl))
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (d *PostgresDriver) Unlock(ctx context.Context, key, token string) (bool, error) {
	tag, err := d.querier.Exec(ctx, deleteLockSQL, key, token)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
