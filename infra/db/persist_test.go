package db_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/daarxwalker/enjin/infra/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type execTx struct {
	pgx.Tx
	execErr    error
	execs      [][]any
	sqls       []string
	rolledBack bool
}

func (t *execTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	t.sqls = append(t.sqls, sql)
	t.execs = append(t.execs, args)
	return pgconn.CommandTag{}, t.execErr
}

func (t *execTx) Rollback(context.Context) error {
	t.rolledBack = true
	return nil
}

type item struct {
	ID   string
	Name string
}

func items(count int, withID bool) []item {
	result := make([]item, count)
	for i := range result {
		result[i].Name = strconv.Itoa(i)
		if withID {
			result[i].ID = "id-" + strconv.Itoa(i)
		}
	}
	return result
}

func itemPersister(entities []item, options ...db.PersistOption[item]) *db.Persister[item] {
	columns := []db.Column[item]{
		db.ColumnText("name", func(i item) any { return i.Name }),
	}
	options = append(options, db.WithPersistEntities(entities...))
	return db.NewPersist("items", columns, options...)
}

func withItemID() db.PersistOption[item] {
	return db.WithPersistID(func(i item) string { return i.ID })
}

func TestPersist(t *testing.T) {
	t.Run("insert nothing", func(t *testing.T) {
		tx := &execTx{}
		if err := db.BulkInsert(context.Background(), tx, itemPersister(nil)); err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if len(tx.execs) != 0 {
			t.Fatalf("execs = %d, want 0", len(tx.execs))
		}
	})

	t.Run("persist without id", func(t *testing.T) {
		tx := &execTx{}
		if err := db.BulkPersist(context.Background(), tx, itemPersister(items(1, false))); err == nil {
			t.Fatal("err = nil, want error")
		}
		if len(tx.execs) != 0 {
			t.Fatalf("execs = %d, want 0", len(tx.execs))
		}
	})

	t.Run("insert chunks", func(t *testing.T) {
		tx := &execTx{}
		if err := db.BulkInsert(context.Background(), tx, itemPersister(items(65536, false))); err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if len(tx.execs) != 2 || len(tx.execs[0]) != 65535 || len(tx.execs[1]) != 1 {
			t.Fatalf("execs = %d, want 2 with 65535 and 1 args", len(tx.execs))
		}
	})

	t.Run("update chunks", func(t *testing.T) {
		tx := &execTx{}
		if err := db.BulkPersist(context.Background(), tx, itemPersister(items(32768, true), withItemID())); err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if len(tx.execs) != 2 || len(tx.execs[0]) != 65534 || len(tx.execs[1]) != 2 {
			t.Fatalf("execs = %d, want 2 with 65534 and 2 args", len(tx.execs))
		}
		for _, sql := range tx.sqls {
			if !strings.HasPrefix(sql, "UPDATE items SET name = v.name FROM (VALUES ($1::uuid, $2::text)") {
				t.Fatalf("sql = %.80s…", sql)
			}
		}
	})

	t.Run("persist inserts and updates", func(t *testing.T) {
		tx := &execTx{}
		entities := append(items(2, false), items(3, true)...)
		if err := db.BulkPersist(context.Background(), tx, itemPersister(entities, withItemID())); err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if len(tx.sqls) != 2 || !strings.HasPrefix(tx.sqls[0], "INSERT") || !strings.HasPrefix(tx.sqls[1], "UPDATE") {
			t.Fatalf("sqls = %v, want INSERT then UPDATE", tx.sqls)
		}
		if len(tx.execs[0]) != 2 || len(tx.execs[1]) != 6 {
			t.Fatalf("args = %d and %d, want 2 and 6", len(tx.execs[0]), len(tx.execs[1]))
		}
	})

	t.Run("exec error keeps tx", func(t *testing.T) {
		execErr := errors.New("exec failed")
		for name, persist := range map[string]func(context.Context, pgx.Tx, *db.Persister[item]) error{
			"insert":  db.BulkInsert[item],
			"persist": db.BulkPersist[item],
		} {
			t.Run(name, func(t *testing.T) {
				tx := &execTx{execErr: execErr}
				err := persist(context.Background(), tx, itemPersister(items(1, name == "persist"), withItemID()))
				if !errors.Is(err, execErr) {
					t.Fatalf("err = %v, want exec error", err)
				}
				if tx.rolledBack {
					t.Fatal("tx rolled back by helper")
				}
			})
		}
	})
}
