package xdb

import (
	"context"
	"fmt"
	"strings"
)

// ─────────────────────────────────────────────────────────────
// InsertBuilder
// ─────────────────────────────────────────────────────────────

// InsertBuilder is an immutable INSERT query builder.
type InsertBuilder struct {
	ec   execContext
	data insertData
}

func (b InsertBuilder) Columns(cols ...string) InsertBuilder {
	b.data = b.data.Columns(cols...)
	return b
}

func (b InsertBuilder) Values(vals ...any) InsertBuilder {
	b.data = b.data.Values(vals...)
	return b
}

func (b InsertBuilder) SetMap(m map[string]any) InsertBuilder {
	b.data = b.data.SetMap(m)
	return b
}

func (b InsertBuilder) OnConflict(clause string) InsertBuilder {
	b.data = b.data.Suffix(b.ec.conflictKeyword() + " " + clause)
	return b
}

func (b InsertBuilder) Returning(cols ...string) InsertBuilder {
	b.data = b.data.Suffix("RETURNING " + strings.Join(cols, ", "))
	return b
}

func (b InsertBuilder) Exec(ctx context.Context) (int64, error) {
	query, args, err := b.ToSQL()
	if err != nil {
		return 0, fmt.Errorf("insert.Exec build: %w", err)
	}
	b.ec.log(ctx, "insert.Exec", query, args)
	res, err := b.ec.ext.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, b.ec.wrapErr("insert.Exec", err, query, args)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (b InsertBuilder) One(ctx context.Context, dest any) error {
	if !b.ec.supportsReturning() {
		return fmt.Errorf("insert.One: driver %q does not support RETURNING", b.ec.driverName)
	}
	query, args, err := b.ToSQL()
	if err != nil {
		return fmt.Errorf("insert.One build: %w", err)
	}
	b.ec.log(ctx, "insert.One", query, args)
	if err := b.ec.ext.GetContext(ctx, dest, query, args...); err != nil {
		return b.ec.wrapErr("insert.One", err, query, args)
	}
	return nil
}

func (b InsertBuilder) ToSQL() (string, []any, error) {
	pf := questionPlaceholder
	if b.ec.pf != nil {
		pf = b.ec.pf
	}
	return b.data.ToSQL(pf)
}

// ─────────────────────────────────────────────────────────────
// UpdateBuilder
// ─────────────────────────────────────────────────────────────

type UpdateBuilder struct {
	ec   execContext
	data updateData
}

func (b UpdateBuilder) Set(col string, val any) UpdateBuilder {
	b.data = b.data.Set(col, val)
	return b
}

func (b UpdateBuilder) SetMap(m map[string]any) UpdateBuilder {
	b.data = b.data.SetMap(m)
	return b
}

func (b UpdateBuilder) SetExpr(col, expr string, args ...any) UpdateBuilder {
	b.data = b.data.SetExpr(col, expr, args...)
	return b
}

func (b UpdateBuilder) Where(pred Predicate) UpdateBuilder {
	b.data = b.data.Where(pred)
	return b
}

func (b UpdateBuilder) OrderBy(col string, dir SortDir) UpdateBuilder {
	b.data = b.data.OrderBy(col + " " + string(dir))
	return b
}

func (b UpdateBuilder) Limit(n uint64) UpdateBuilder {
	b.data = b.data.Limit(n)
	return b
}

func (b UpdateBuilder) WhereIf(cond bool, pred Predicate) UpdateBuilder {
	if cond {
		b.data = b.data.Where(pred)
	}
	return b
}

func (b UpdateBuilder) Returning(cols ...string) UpdateBuilder {
	b.data = b.data.Suffix("RETURNING " + strings.Join(cols, ", "))
	return b
}

func (b UpdateBuilder) Exec(ctx context.Context) (int64, error) {
	query, args, err := b.ToSQL()
	if err != nil {
		return 0, fmt.Errorf("update.Exec build: %w", err)
	}
	b.ec.log(ctx, "update.Exec", query, args)
	res, err := b.ec.ext.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, b.ec.wrapErr("update.Exec", err, query, args)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (b UpdateBuilder) ExecMustAffect(ctx context.Context) error {
	n, err := b.Exec(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNoRows
	}
	return nil
}

func (b UpdateBuilder) One(ctx context.Context, dest any) error {
	if !b.ec.supportsReturning() {
		return fmt.Errorf("update.One: driver %q does not support RETURNING", b.ec.driverName)
	}
	query, args, err := b.ToSQL()
	if err != nil {
		return fmt.Errorf("update.One build: %w", err)
	}
	b.ec.log(ctx, "update.One", query, args)
	if err := b.ec.ext.GetContext(ctx, dest, query, args...); err != nil {
		if isNoRows(err) {
			return ErrNotFound
		}
		return b.ec.wrapErr("update.One", err, query, args)
	}
	return nil
}

func (b UpdateBuilder) ToSQL() (string, []any, error) {
	pf := questionPlaceholder
	if b.ec.pf != nil {
		pf = b.ec.pf
	}
	return b.data.ToSQL(pf)
}

// ─────────────────────────────────────────────────────────────
// DeleteBuilder
// ─────────────────────────────────────────────────────────────

type DeleteBuilder struct {
	ec   execContext
	data deleteData
}

func (b DeleteBuilder) Where(pred Predicate) DeleteBuilder {
	b.data = b.data.Where(pred)
	return b
}

func (b DeleteBuilder) OrderBy(col string, dir SortDir) DeleteBuilder {
	b.data = b.data.OrderBy(col + " " + string(dir))
	return b
}

func (b DeleteBuilder) Limit(n uint64) DeleteBuilder {
	b.data = b.data.Limit(n)
	return b
}

func (b DeleteBuilder) WhereIf(cond bool, pred Predicate) DeleteBuilder {
	if cond {
		b.data = b.data.Where(pred)
	}
	return b
}

func (b DeleteBuilder) Returning(cols ...string) DeleteBuilder {
	b.data = b.data.Suffix("RETURNING " + strings.Join(cols, ", "))
	return b
}

func (b DeleteBuilder) Exec(ctx context.Context) (int64, error) {
	query, args, err := b.ToSQL()
	if err != nil {
		return 0, fmt.Errorf("delete.Exec build: %w", err)
	}
	b.ec.log(ctx, "delete.Exec", query, args)
	res, err := b.ec.ext.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, b.ec.wrapErr("delete.Exec", err, query, args)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (b DeleteBuilder) ExecMustAffect(ctx context.Context) error {
	n, err := b.Exec(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNoRows
	}
	return nil
}

func (b DeleteBuilder) ToSQL() (string, []any, error) {
	pf := questionPlaceholder
	if b.ec.pf != nil {
		pf = b.ec.pf
	}
	return b.data.ToSQL(pf)
}
