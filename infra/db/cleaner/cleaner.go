package cleaner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/daarxwalker/enjin/infra/db"

	"github.com/jackc/pgx/v5"
)

type Cleaner struct {
	querier   db.Querier
	log       *slog.Logger
	tables    []string
	batchSize int
	interval  time.Duration
}

type Option func(*Cleaner)

// NewCleaner deletes expired rows from tables, each of which needs an
// expires_at timestamptz column. A table name may carry its schema
// ("app.sessions").
func NewCleaner(querier db.Querier, tables []string, options ...Option) *Cleaner {
	c := &Cleaner{
		querier:   querier,
		log:       slog.Default(),
		tables:    slices.Clone(tables),
		batchSize: 1000,
		interval:  time.Minute,
	}
	for _, option := range options {
		option(c)
	}
	return c
}

func WithLogger(log *slog.Logger) Option {
	return func(c *Cleaner) {
		c.log = log
	}
}

// WithBatchSize sets how many rows one DELETE removes, so a cleanup never
// holds many row locks at once.
func WithBatchSize(batchSize int) Option {
	return func(c *Cleaner) {
		if batchSize > 0 {
			c.batchSize = batchSize
		}
	}
}

// WithInterval sets how often RunCleanups deletes expired rows; the default
// is one minute. A zero or negative interval is ignored.
func WithInterval(interval time.Duration) Option {
	return func(c *Cleaner) {
		if interval > 0 {
			c.interval = interval
		}
	}
}

// DeleteExpiredRows deletes the expired rows of every table and returns how
// many. A failing table does not stop the others; their errors are joined.
func (c *Cleaner) DeleteExpiredRows(ctx context.Context) (int64, error) {
	var (
		total int64
		errs  []error
	)
	for _, table := range c.tables {
		deleted, err := c.deleteExpiredTableRows(ctx, table)
		total += deleted
		if err != nil {
			errs = append(errs, fmt.Errorf("delete expired rows of %s failed: %w", table, err))
		}
	}
	return total, errors.Join(errs...)
}

// RunCleanups deletes expired rows every interval (WithInterval) until ctx
// ends. A failed cleanup is logged and the next one runs on time.
func (c *Cleaner) RunCleanups(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := c.DeleteExpiredRows(ctx); err != nil {
				c.log.ErrorContext(ctx, fmt.Sprintf("cleaner: expired rows were not deleted: %v", err))
			}
		}
	}
}

func (c *Cleaner) deleteExpiredTableRows(ctx context.Context, table string) (int64, error) {
	identifier := pgx.Identifier(strings.Split(table, ".")).Sanitize()
	query := `DELETE FROM ` + identifier + ` WHERE ctid IN (
    SELECT ctid FROM ` + identifier + ` WHERE expires_at <= now() LIMIT $1
)`
	var total int64
	for {
		tag, err := c.querier.Exec(ctx, query, c.batchSize)
		if err != nil {
			return total, fmt.Errorf("delete batch of expired rows failed: %w", err)
		}
		total += tag.RowsAffected()
		if tag.RowsAffected() < int64(c.batchSize) {
			return total, nil
		}
	}
}
