package xdb

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Rows is an alias for sqlx.Rows, exposed so callers of Each do not
// need to import sqlx directly.
type Rows = sqlx.Rows

// SelectBuilder is an immutable SELECT query builder.
type SelectBuilder struct {
	ec        execContext
	data      selectData
	allowSort map[string]bool // optional whitelist for safe ORDER BY
	ctes      []CTE           // accumulated CTEs for WITH clauses
}

// ── Chain methods ────────────────────────────────────────────

func (b SelectBuilder) From(table string) SelectBuilder {
	b.data = b.data.From(table)
	return b
}

func (b SelectBuilder) FromSubquery(sb SelectBuilder, alias string) SelectBuilder {
	b.data = b.data.FromSubquery(sb.data, alias)
	return b
}

func (b SelectBuilder) Join(expr string) SelectBuilder {
	b.data = b.data.Join(expr)
	return b
}

func (b SelectBuilder) LeftJoin(expr string) SelectBuilder {
	b.data = b.data.LeftJoin(expr)
	return b
}

func (b SelectBuilder) Where(pred Predicate) SelectBuilder {
	b.data = b.data.Where(pred)
	return b
}

func (b SelectBuilder) WhereIf(cond bool, pred Predicate) SelectBuilder {
	if cond {
		b.data = b.data.Where(pred)
	}
	return b
}

func (b SelectBuilder) GroupBy(cols ...string) SelectBuilder {
	b.data = b.data.GroupBy(cols...)
	return b
}

func (b SelectBuilder) Having(pred Predicate) SelectBuilder {
	b.data = b.data.Having(pred)
	return b
}

func (b SelectBuilder) AllowSort(cols ...string) SelectBuilder {
	m := make(map[string]bool, len(cols))
	for _, c := range cols {
		m[c] = true
	}
	b.allowSort = m
	return b
}

func (b SelectBuilder) OrderBy(col string, dir SortDir) SelectBuilder {
	if col == "" {
		return b
	}
	if b.allowSort != nil && !b.allowSort[col] {
		return b
	}
	b.data = b.data.OrderBy(col + " " + string(dir))
	return b
}

func (b SelectBuilder) Paginate(p Page) SelectBuilder {
	b.data = b.data.Limit(uint64(p.size())).Offset(uint64(p.offset()))
	return b
}

func (b SelectBuilder) Limit(n uint64) SelectBuilder {
	b.data = b.data.Limit(n)
	return b
}

func (b SelectBuilder) Offset(n uint64) SelectBuilder {
	b.data = b.data.Offset(n)
	return b
}

func (b SelectBuilder) Suffix(sql string, args ...any) SelectBuilder {
	b.data = b.data.Suffix(sql, args...)
	return b
}

func (b SelectBuilder) WithCTE(name, sql string, args ...any) SelectBuilder {
	b.ctes = append(b.ctes, CTE{name: name, sql: sql, args: args})
	return b
}

func (b SelectBuilder) WithSelectCTE(name string, sb SelectBuilder) SelectBuilder {
	sql, args, err := sb.ToSQL()
	if err != nil {
		return b
	}
	return b.WithCTE(name, sql, args...)
}

func (b SelectBuilder) Union(other SelectBuilder) SelectBuilder {
	b.data = b.data.Union("UNION", false, other.data)
	return b
}

func (b SelectBuilder) UnionAll(other SelectBuilder) SelectBuilder {
	b.data = b.data.Union("UNION", true, other.data)
	return b
}

func (b SelectBuilder) Intersect(other SelectBuilder) SelectBuilder {
	b.data = b.data.Union("INTERSECT", false, other.data)
	return b
}

func (b SelectBuilder) IntersectAll(other SelectBuilder) SelectBuilder {
	b.data = b.data.Union("INTERSECT", true, other.data)
	return b
}

func (b SelectBuilder) Except(other SelectBuilder) SelectBuilder {
	b.data = b.data.Union("EXCEPT", false, other.data)
	return b
}

func (b SelectBuilder) ExceptAll(other SelectBuilder) SelectBuilder {
	b.data = b.data.Union("EXCEPT", true, other.data)
	return b
}

func (b SelectBuilder) RemoveLimit() SelectBuilder {
	b.data = b.data.RemoveLimit()
	return b
}

func (b SelectBuilder) RemoveOffset() SelectBuilder {
	b.data = b.data.RemoveOffset()
	return b
}

// ToSQL returns the generated SQL string and bound arguments.
func (b SelectBuilder) ToSQL() (string, []any, error) {
	pf := questionPlaceholder
	if b.ec.pf != nil {
		pf = b.ec.pf
	}

	mainSQL, mainArgs, err := b.data.ToSQL(pf)
	if err != nil {
		return "", nil, err
	}

	if len(b.ctes) == 0 {
		return mainSQL, mainArgs, nil
	}

	withClause, withArgs := buildWithClause(b.ctes)
	finalSQL, finalArgs := prependWithClause(withClause, withArgs, mainSQL, mainArgs)
	return finalSQL, finalArgs, nil
}

// ── Execution methods ────────────────────────────────────────

// One executes the query and scans a single row into dest.
func (b SelectBuilder) One(ctx context.Context, dest any) error {
	query, args, err := b.ToSQL()
	if err != nil {
		return fmt.Errorf("select.One build: %w", err)
	}
	b.ec.log(ctx, "select.One", query, args)
	if err := b.ec.ext.GetContext(ctx, dest, query, args...); err != nil {
		if isNoRows(err) {
			return ErrNotFound
		}
		return b.ec.wrapErr("select.One", err, query, args)
	}
	return nil
}

// All executes the query and scans all rows into dest.
func (b SelectBuilder) All(ctx context.Context, dest any) error {
	query, args, err := b.ToSQL()
	if err != nil {
		return fmt.Errorf("select.All build: %w", err)
	}
	b.ec.log(ctx, "select.All", query, args)
	if err := b.ec.ext.SelectContext(ctx, dest, query, args...); err != nil {
		return b.ec.wrapErr("select.All", err, query, args)
	}
	return nil
}

// Count clones the builder, replaces columns with COUNT(*),
// removes ORDER BY, LIMIT, and OFFSET, then returns the total count.
func (b SelectBuilder) Count(ctx context.Context) (int, error) {
	countData := b.data
	countData.columns = []string{"COUNT(*)"}
	countData.orderBy = nil
	countData.limit = nil
	countData.offset = nil

	countBuilder := SelectBuilder{
		ec:        b.ec,
		data:      countData,
		ctes:      b.ctes,
		allowSort: b.allowSort,
	}

	query, args, err := countBuilder.ToSQL()
	if err != nil {
		return 0, fmt.Errorf("select.Count build: %w", err)
	}
	b.ec.log(ctx, "select.Count", query, args)
	var n int
	if err := b.ec.ext.GetContext(ctx, &n, query, args...); err != nil {
		return 0, b.ec.wrapErr("select.Count", err, query, args)
	}
	return n, nil
}

// Exists wraps the query in SELECT EXISTS (…) and returns the boolean result.
func (b SelectBuilder) Exists(ctx context.Context) (bool, error) {
	inner, args, err := b.ToSQL()
	if err != nil {
		return false, fmt.Errorf("select.Exists build: %w", err)
	}
	query := "SELECT EXISTS (" + inner + ")"
	b.ec.log(ctx, "select.Exists", query, args)
	var exists bool
	if err := b.ec.ext.GetContext(ctx, &exists, query, args...); err != nil {
		return false, b.ec.wrapErr("select.Exists", err, query, args)
	}
	return exists, nil
}

// Each executes the query and calls fn for every row.
func (b SelectBuilder) Each(ctx context.Context, fn func(*sqlx.Rows) error) error {
	query, args, err := b.ToSQL()
	if err != nil {
		return fmt.Errorf("select.Each build: %w", err)
	}
	b.ec.log(ctx, "select.Each", query, args)
	rows, err := b.ec.ext.QueryxContext(ctx, query, args...)
	if err != nil {
		return b.ec.wrapErr("select.Each", err, query, args)
	}
	defer rows.Close()
	for rows.Next() {
		if err := fn(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}
