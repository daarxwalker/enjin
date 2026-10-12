package db

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

const maxParams = 65535

type PersistOption[T any] func(*Persister[T])

type Persister[T any] struct {
	table    string
	id       func(ent T) string
	columns  []Column[T]
	entities []T
}

func NewPersist[T any](table string, columns []Column[T], options ...PersistOption[T]) *Persister[T] {
	p := &Persister[T]{
		table:   table,
		columns: columns,
	}
	for _, option := range options {
		option(p)
	}
	return p
}

func WithPersistID[T any](fn func(ent T) string) PersistOption[T] {
	return func(p *Persister[T]) {
		p.id = fn
	}
}

func WithPersistEntities[T any](entities ...T) PersistOption[T] {
	return func(p *Persister[T]) {
		p.entities = entities
	}
}

func BulkPersist[T any](ctx context.Context, tx pgx.Tx, p *Persister[T]) error {
	if p.id == nil {
		return fmt.Errorf("persist %s without id func", p.table)
	}
	insertEntities := make([]T, 0, len(p.entities))
	updateEntities := make([]T, 0, len(p.entities))
	for _, entity := range p.entities {
		id := p.id(entity)
		if len(id) == 0 {
			insertEntities = append(insertEntities, entity)
		} else {
			updateEntities = append(updateEntities, entity)
		}
	}
	if insertErr := p.insert(ctx, tx, insertEntities); insertErr != nil {
		return insertErr
	}
	return p.update(ctx, tx, updateEntities)
}

func BulkInsert[T any](ctx context.Context, tx pgx.Tx, p *Persister[T]) error {
	return p.insert(ctx, tx, p.entities)
}

func (p *Persister[T]) insert(ctx context.Context, tx pgx.Tx, entities []T) error {
	if len(entities) == 0 {
		return nil
	}
	if len(p.columns) == 0 {
		return fmt.Errorf("insert %s without columns", p.table)
	}
	columns := make([]string, len(p.columns))
	for i, column := range p.columns {
		columns[i] = column.Name
	}
	for chunk := range slices.Chunk(entities, maxParams/len(p.columns)) {
		query := squirrel.Insert(p.table).
			Columns(columns...)
		for _, entity := range chunk {
			entityValues := make([]any, len(p.columns))
			for i, column := range p.columns {
				entityValues[i] = column.Get(entity)
			}
			query = query.Values(
				entityValues...,
			)
		}
		sql, args, createSQLErr := query.
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if createSQLErr != nil {
			return fmt.Errorf("create insert %s sql failed: %w", p.table, createSQLErr)
		}
		if _, execErr := tx.Exec(ctx, sql, args...); execErr != nil {
			return fmt.Errorf("insert %s failed: %w", p.table, execErr)
		}
	}
	return nil
}

func (p *Persister[T]) update(ctx context.Context, tx pgx.Tx, entities []T) error {
	if len(entities) == 0 {
		return nil
	}
	query := squirrel.Update(p.table)
	for _, col := range p.columns {
		query = query.Set(col.Name, squirrel.Expr("v."+col.Name))
	}
	baseSQL, _, createSQLErr := query.ToSql()
	if createSQLErr != nil {
		return fmt.Errorf("create update %s sql failed: %w", p.table, createSQLErr)
	}

	columnNames := make([]string, 0, len(p.columns)+1)
	columnNames = append(columnNames, "id")
	for _, col := range p.columns {
		columnNames = append(columnNames, col.Name)
	}

	rowParts := make([]string, 0, len(p.columns)+1)
	rowParts = append(rowParts, "?::uuid")
	for _, col := range p.columns {
		part := "?"
		if col.Cast != "" {
			part += "::" + col.Cast
		}
		rowParts = append(rowParts, part)
	}
	rowTemplate := "(" + strings.Join(rowParts, ", ") + ")"

	for chunk := range slices.Chunk(entities, maxParams/len(columnNames)) {
		placeholders := make([]string, 0, len(chunk))
		args := make([]any, 0, len(chunk)*len(columnNames))
		for _, entity := range chunk {
			placeholders = append(placeholders, rowTemplate)
			args = append(args, p.id(entity))
			for _, col := range p.columns {
				args = append(args, col.Get(entity))
			}
		}

		sql := baseSQL +
			" FROM (VALUES " + strings.Join(placeholders, ", ") + ")" +
			" AS v(" + strings.Join(columnNames, ", ") + ")" +
			" WHERE " + p.table + ".id = v.id"

		sql, rpErr := squirrel.Dollar.ReplacePlaceholders(sql)
		if rpErr != nil {
			return fmt.Errorf("replace placeholders in update %s sql failed: %w", p.table, rpErr)
		}
		if _, execErr := tx.Exec(ctx, sql, args...); execErr != nil {
			return fmt.Errorf("update %s failed: %w", p.table, execErr)
		}
	}
	return nil
}
