package cache

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/daarxwalker/enjin/infra/db"

	"github.com/jackc/pgx/v5"
)

var _ Driver = (*PostgresDriver)(nil)

// PostgresDriver keeps the cache in the UNLOGGED table enjin_cache, created
// by package infra/db/migrations/keyvalue.
type PostgresDriver struct {
	querier db.Querier
}

func NewPostgresDriver(querier db.Querier) *PostgresDriver {
	return &PostgresDriver{querier: querier}
}

func (d *PostgresDriver) Get(ctx context.Context, key string) ([]byte, bool, error) {
	var value []byte
	err := d.querier.QueryRow(ctx, loadValueSQL, key).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

func (d *PostgresDriver) Take(ctx context.Context, key string) ([]byte, bool, error) {
	var value []byte
	var live bool
	err := d.querier.QueryRow(ctx, takeValueSQL, key).Scan(&value, &live)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !live {
		return nil, false, nil
	}
	return value, true, nil
}

func (d *PostgresDriver) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if value == nil {
		value = []byte{}
	}
	_, err := d.querier.Exec(ctx, upsertValueSQL, key, value, expiresIn(ttl))
	return err
}

// UpdateExpiration with a zero or negative TTL deletes the key, the same as
// Redis PEXPIRE.
func (d *PostgresDriver) UpdateExpiration(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	query, args := updateExpirationSQL, []any{key, expiresIn(ttl)}
	if ttl <= 0 {
		query, args = deleteLiveValueSQL, []any{key}
	}
	tag, err := d.querier.Exec(ctx, query, args...)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (d *PostgresDriver) Delete(ctx context.Context, key string) error {
	_, err := d.querier.Exec(ctx, deleteValueSQL, key)
	return err
}

func (d *PostgresDriver) Exists(ctx context.Context, key string) (bool, error) {
	var exists bool
	if err := d.querier.QueryRow(ctx, loadExistenceSQL, key).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (d *PostgresDriver) TTL(ctx context.Context, key string) (time.Duration, bool, error) {
	var micros int64
	err := d.querier.QueryRow(ctx, loadTTLSQL, key).Scan(&micros)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return time.Duration(micros) * time.Microsecond, true, nil
}

func (d *PostgresDriver) Increment(ctx context.Context, key string, delta int64, ttl time.Duration) (int64, error) {
	query := incrementValueSQL
	if delta < 0 {
		query = decrementValueSQL
	}
	var value int64
	err := d.querier.QueryRow(ctx, query, key, delta, expiresIn(ttl)).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return value, nil
}

// DeleteByPattern deletes keys matching a Redis glob. It knows `*`, `?` and
// a backslash escape; `[...]` classes match literally.
func (d *PostgresDriver) DeleteByPattern(ctx context.Context, pattern string) error {
	_, err := d.querier.Exec(ctx, deleteValuesByPatternSQL, d.createLikePattern(pattern))
	return err
}

func (d *PostgresDriver) createLikePattern(glob string) string {
	var likePattern strings.Builder
	escaped := false
	for _, char := range glob {
		switch {
		case escaped:
			d.writeEscapedLikeChar(&likePattern, char)
			escaped = false
		case char == '\\':
			escaped = true
		case char == '*':
			likePattern.WriteRune('%')
		case char == '?':
			likePattern.WriteRune('_')
		default:
			d.writeEscapedLikeChar(&likePattern, char)
		}
	}
	if escaped {
		d.writeEscapedLikeChar(&likePattern, '\\')
	}
	return likePattern.String()
}

func (d *PostgresDriver) writeEscapedLikeChar(likePattern *strings.Builder, char rune) {
	if char == '%' || char == '_' || char == '\\' {
		likePattern.WriteRune('\\')
	}
	likePattern.WriteRune(char)
}
