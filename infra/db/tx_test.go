package db_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/daarxwalker/enjin/infra/db"

	"github.com/jackc/pgx/v5"
)

type fakeTx struct {
	pgx.Tx
	rollbackErr error
	rolledBack  bool
	committed   bool
}

func (t *fakeTx) Rollback(context.Context) error {
	t.rolledBack = true
	return t.rollbackErr
}

func (t *fakeTx) Commit(context.Context) error {
	t.committed = true
	return nil
}

type fakeQuerier struct {
	db.Querier
	tx *fakeTx
}

func (q *fakeQuerier) Begin(context.Context) (pgx.Tx, error) {
	return q.tx, nil
}

func TestWithTx(t *testing.T) {
	t.Run("commit", func(t *testing.T) {
		tx := &fakeTx{}
		if err := db.WithTx(context.Background(), &fakeQuerier{tx: tx}, "test", func(pgx.Tx) error { return nil }); err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if !tx.committed || tx.rolledBack {
			t.Fatalf("committed = %v, rolledBack = %v", tx.committed, tx.rolledBack)
		}
	})

	t.Run("rollback error", func(t *testing.T) {
		fnErr := errors.New("fn failed")
		rbErr := errors.New("rollback failed")
		tx := &fakeTx{rollbackErr: rbErr}
		err := db.WithTx(context.Background(), &fakeQuerier{tx: tx}, "test", func(pgx.Tx) error { return fnErr })
		if !tx.rolledBack {
			t.Fatal("tx not rolled back")
		}
		if !errors.Is(err, fnErr) || !errors.Is(err, rbErr) {
			t.Fatalf("err = %v, want both fn and rollback errors", err)
		}
		if !strings.Contains(err.Error(), "rollback test tx failed: rollback failed") {
			t.Fatalf("err = %v, want rollback error message", err)
		}
	})

	t.Run("panic", func(t *testing.T) {
		tx := &fakeTx{}
		defer func() {
			if p := recover(); p != "boom" {
				t.Fatalf("recovered = %v, want boom", p)
			}
			if !tx.rolledBack || tx.committed {
				t.Fatalf("committed = %v, rolledBack = %v", tx.committed, tx.rolledBack)
			}
		}()
		_ = db.WithTx(context.Background(), &fakeQuerier{tx: tx}, "test", func(pgx.Tx) error { panic("boom") })
	})

	t.Run("panic with failed rollback keeps the panic", func(t *testing.T) {
		panicErr := errors.New("boom")
		rbErr := errors.New("rollback failed")
		tx := &fakeTx{rollbackErr: rbErr}
		defer func() {
			err, ok := recover().(error)
			if !ok || !errors.Is(err, panicErr) || !errors.Is(err, rbErr) {
				t.Fatalf("recovered = %v, want both panic and rollback errors", err)
			}
		}()
		_ = db.WithTx(context.Background(), &fakeQuerier{tx: tx}, "test", func(pgx.Tx) error { panic(panicErr) })
	})

	t.Run("non error panic with failed rollback keeps its value", func(t *testing.T) {
		rbErr := errors.New("rollback failed")
		tx := &fakeTx{rollbackErr: rbErr}
		defer func() {
			err, ok := recover().(error)
			if !ok || !errors.Is(err, rbErr) || !strings.HasPrefix(err.Error(), "boom: ") {
				t.Fatalf("recovered = %v, want boom with rollback error", err)
			}
		}()
		_ = db.WithTx(context.Background(), &fakeQuerier{tx: tx}, "test", func(pgx.Tx) error { panic("boom") })
	})
}
