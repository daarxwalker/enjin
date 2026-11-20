package csrf

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/daarxwalker/enjin/infra/db"

	"github.com/jackc/pgx/v5"
)

var _ Driver = (*PostgresDriver)(nil)

// PostgresDriver keeps the tokens in the UNLOGGED table enjin_csrf, created
// by package infra/db/migrations/keyvalue.
type PostgresDriver struct {
	querier db.Querier
}

func NewPostgresDriver(querier db.Querier) *PostgresDriver {
	return &PostgresDriver{querier: querier}
}

// Add removes expired tokens by the application clock, the same clock that
// computes expiresAt and the now of Take, as the Redis driver does.
func (d *PostgresDriver) Add(ctx context.Context, key, token string, expiresAt time.Time, limit int) error {
	var batch pgx.Batch
	batch.Queue(lockKeySQL, key)
	batch.Queue(deleteExpiredEntriesSQL, key, time.Now())
	batch.Queue(upsertEntrySQL, key, token, expiresAt)
	batch.Queue(deleteEntriesOverLimitSQL, key, max(limit, 1))
	err := db.WithTx(ctx, d.querier, "add csrf token", func(tx pgx.Tx) error {
		if sendErr := tx.SendBatch(ctx, &batch).Close(); sendErr != nil {
			return fmt.Errorf("send csrf token batch failed: %w", sendErr)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("add csrf token for %q failed: %w", key, err)
	}
	return nil
}

func (d *PostgresDriver) Take(ctx context.Context, key, token string, now time.Time) (bool, error) {
	var expiresAt time.Time
	err := d.querier.QueryRow(ctx, takeEntrySQL, key, token).Scan(&expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("take csrf token for %q failed: %w", key, err)
	}
	return expiresAt.After(now), nil
}
