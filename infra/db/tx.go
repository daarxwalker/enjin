package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func WithTx(
	ctx context.Context,
	db Querier,
	name string,
	fn func(tx pgx.Tx) error,
) (err error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin %s tx failed: %w", name, err)
	}
	defer func() {
		if p := recover(); p != nil {
			rbErr := tx.Rollback(ctx)
			if rbErr == nil || errors.Is(rbErr, pgx.ErrTxClosed) {
				panic(p)
			}
			rbErr = fmt.Errorf("rollback %s tx failed: %w", name, rbErr)
			if pErr, ok := p.(error); ok {
				panic(errors.Join(pErr, rbErr))
			}
			panic(fmt.Errorf("%v: %w", p, rbErr))
		}
		if err == nil {
			return
		}
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			err = errors.Join(err, fmt.Errorf("rollback %s tx failed: %w", name, rbErr))
		}
	}()
	if err = fn(tx); err != nil {
		return fmt.Errorf("call func in %s tx failed: %w", name, err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit %s tx failed: %w", name, err)
	}
	return nil
}
