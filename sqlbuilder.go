package xdb

import (
	"fmt"
	"strconv"
	"strings"
)

type suffixClause struct {
	sql  string
	args []any
}

type joinClause struct {
	joinType string // "JOIN" or "LEFT JOIN"
	expr     string
}

// ─────────────────────────────────────────────────────────────
// selectData
// ─────────────────────────────────────────────────────────────

type unionItem struct {
	typ  string // UNION, INTERSECT, EXCEPT
	all  bool
	data selectData // the inner query
}

type fromSubquery struct {
	sql   string
	args  []any
	alias string
}

type selectData struct {
	columns       []string
	from          string
	fromSubquery  *fromSubquery
	unions        []unionItem
	joins         []joinClause
	wherePreds    []Predicate
	groupBy       []string
	having        Predicate
	orderBy       []string
	limit         *uint64
	offset        *uint64
	suffixes      []suffixClause
	prefixes      []suffixClause // for WITH clauses etc.
}

func (d selectData) Columns(cols ...string) selectData {
	d.columns = append(d.columns, cols...)
	return d
}

func (d selectData) Column(col string) selectData {
	d.columns = append(d.columns, col)
	return d
}

func (d selectData) From(table string) selectData {
	d.from = table
	d.fromSubquery = nil
	return d
}

func (d selectData) FromSubquery(sq selectData, alias string) selectData {
	sql, args, _ := sq.ToSQL(questionPlaceholder)
	d.from = ""
	d.fromSubquery = &fromSubquery{sql: sql, args: args, alias: alias}
	return d
}

func (d selectData) Union(typ string, all bool, other selectData) selectData {
	d.unions = append(d.unions, unionItem{typ: typ, all: all, data: other})
	return d
}

func (d selectData) Join(expr string) selectData {
	d.joins = append(d.joins, joinClause{joinType: "JOIN", expr: expr})
	return d
}

func (d selectData) LeftJoin(expr string) selectData {
	d.joins = append(d.joins, joinClause{joinType: "LEFT JOIN", expr: expr})
	return d
}

func (d selectData) Where(pred Predicate) selectData {
	d.wherePreds = append(d.wherePreds, pred)
	return d
}

func (d selectData) GroupBy(cols ...string) selectData {
	d.groupBy = append(d.groupBy, cols...)
	return d
}

func (d selectData) Having(pred Predicate) selectData {
	d.having = pred
	return d
}

func (d selectData) OrderBy(col string) selectData {
	d.orderBy = append(d.orderBy, col)
	return d
}

func (d selectData) Limit(n uint64) selectData {
	d.limit = &n
	return d
}

func (d selectData) Offset(n uint64) selectData {
	d.offset = &n
	return d
}

func (d selectData) Suffix(sql string, args ...any) selectData {
	d.suffixes = append(d.suffixes, suffixClause{sql: sql, args: args})
	return d
}

func (d selectData) Prefix(sql string, args ...any) selectData {
	d.prefixes = append(d.prefixes, suffixClause{sql: sql, args: args})
	return d
}

func (d selectData) RemoveLimit() selectData {
	d.limit = nil
	return d
}

func (d selectData) RemoveOffset() selectData {
	d.offset = nil
	return d
}

func (d selectData) ToSQL(pf placeholderFormat) (string, []any, error) {
	if len(d.unions) > 0 {
		return d.toSQLUnion(pf)
	}

	var buf strings.Builder
	var args []any

	// Prefix (WITH clauses etc.)
	for _, p := range d.prefixes {
		buf.WriteString(p.sql)
		buf.WriteString(" ")
		args = append(args, p.args...)
	}

	buf.WriteString("SELECT ")

	if len(d.columns) == 0 {
		buf.WriteString("*")
	} else {
		buf.WriteString(strings.Join(d.columns, ", "))
	}

	if d.fromSubquery != nil {
		buf.WriteString(" FROM (")
		buf.WriteString(d.fromSubquery.sql)
		buf.WriteString(") AS ")
		buf.WriteString(d.fromSubquery.alias)
		args = append(args, d.fromSubquery.args...)
	} else if d.from != "" {
		buf.WriteString(" FROM ")
		buf.WriteString(d.from)
	}

	for _, j := range d.joins {
		buf.WriteString(" ")
		buf.WriteString(j.joinType)
		buf.WriteString(" ")
		buf.WriteString(j.expr)
	}

	if len(d.wherePreds) > 0 {
		buf.WriteString(" WHERE ")
		for i, pred := range d.wherePreds {
			if i > 0 {
				buf.WriteString(" AND ")
			}
			sql, predArgs, err := pred.ToSQL()
			if err != nil {
				return "", nil, fmt.Errorf("selectData.Where: %w", err)
			}
			buf.WriteString(sql)
			args = append(args, predArgs...)
		}
	}

	if len(d.groupBy) > 0 {
		buf.WriteString(" GROUP BY ")
		buf.WriteString(strings.Join(d.groupBy, ", "))
	}

	if d.having != nil {
		sql, havingArgs, err := d.having.ToSQL()
		if err != nil {
			return "", nil, fmt.Errorf("selectData.Having: %w", err)
		}
		buf.WriteString(" HAVING ")
		buf.WriteString(sql)
		args = append(args, havingArgs...)
	}

	if len(d.orderBy) > 0 {
		buf.WriteString(" ORDER BY ")
		buf.WriteString(strings.Join(d.orderBy, ", "))
	}

	if d.limit != nil {
		buf.WriteString(" LIMIT ")
		buf.WriteString(strconv.FormatUint(*d.limit, 10))
	}

	if d.offset != nil {
		buf.WriteString(" OFFSET ")
		buf.WriteString(strconv.FormatUint(*d.offset, 10))
	}

	for _, s := range d.suffixes {
		buf.WriteString(" ")
		buf.WriteString(s.sql)
		args = append(args, s.args...)
	}

	sql := pf(buf.String())
	return sql, args, nil
}

func (d selectData) toSQLUnion(pf placeholderFormat) (string, []any, error) {
	var buf strings.Builder
	var args []any

	// Render the main SELECT (without ORDER BY/LIMIT/OFFSET/SUFFIX — those belong to the union result)
	mainSQL, mainArgs, err := d.renderCore(pf)
	if err != nil {
		return "", nil, err
	}
	buf.WriteString(mainSQL)

	for _, u := range d.unions {
		childSQL, childArgs, err := u.data.renderCore(pf)
		if err != nil {
			return "", nil, err
		}
		buf.WriteString("\n")
		buf.WriteString(u.typ)
		if u.all {
			buf.WriteString(" ALL")
		}
		buf.WriteString("\n")
		buf.WriteString(childSQL)
		args = append(args, childArgs...)
	}

	// ORDER BY at union level
	if len(d.orderBy) > 0 {
		buf.WriteString("\nORDER BY ")
		buf.WriteString(strings.Join(d.orderBy, ", "))
	}

	if d.limit != nil {
		buf.WriteString("\nLIMIT ")
		buf.WriteString(strconv.FormatUint(*d.limit, 10))
	}

	if d.offset != nil {
		buf.WriteString("\nOFFSET ")
		buf.WriteString(strconv.FormatUint(*d.offset, 10))
	}

	for _, s := range d.suffixes {
		buf.WriteString(" ")
		buf.WriteString(s.sql)
		args = append(args, s.args...)
	}

	sql := pf(buf.String())
	return sql, append(mainArgs, args...), nil
}

// renderCore renders SELECT columns FROM table JOIN … WHERE … GROUP BY … HAVING …
// without ORDER BY, LIMIT, OFFSET, or SUFFIX (used for union members).
func (d selectData) renderCore(pf placeholderFormat) (string, []any, error) {
	var buf strings.Builder
	var args []any

	buf.WriteString("SELECT ")
	if len(d.columns) == 0 {
		buf.WriteString("*")
	} else {
		buf.WriteString(strings.Join(d.columns, ", "))
	}

	if d.fromSubquery != nil {
		buf.WriteString(" FROM (")
		buf.WriteString(d.fromSubquery.sql)
		buf.WriteString(") AS ")
		buf.WriteString(d.fromSubquery.alias)
		args = append(args, d.fromSubquery.args...)
	} else if d.from != "" {
		buf.WriteString(" FROM ")
		buf.WriteString(d.from)
	}

	for _, j := range d.joins {
		buf.WriteString(" ")
		buf.WriteString(j.joinType)
		buf.WriteString(" ")
		buf.WriteString(j.expr)
	}

	if len(d.wherePreds) > 0 {
		buf.WriteString(" WHERE ")
		for i, pred := range d.wherePreds {
			if i > 0 {
				buf.WriteString(" AND ")
			}
			sql, predArgs, err := pred.ToSQL()
			if err != nil {
				return "", nil, fmt.Errorf("selectData.renderCore.Where: %w", err)
			}
			buf.WriteString(sql)
			args = append(args, predArgs...)
		}
	}

	if len(d.groupBy) > 0 {
		buf.WriteString(" GROUP BY ")
		buf.WriteString(strings.Join(d.groupBy, ", "))
	}

	if d.having != nil {
		sql, havingArgs, err := d.having.ToSQL()
		if err != nil {
			return "", nil, fmt.Errorf("selectData.renderCore.Having: %w", err)
		}
		buf.WriteString(" HAVING ")
		buf.WriteString(sql)
		args = append(args, havingArgs...)
	}

	return buf.String(), args, nil
}

// ─────────────────────────────────────────────────────────────
// insertData
// ─────────────────────────────────────────────────────────────

type insertData struct {
	table    string
	columns  []string
	values   [][]any
	setMap   map[string]any
	suffixes []suffixClause
}

func (d insertData) Columns(cols ...string) insertData {
	d.columns = append(d.columns, cols...)
	return d
}

func (d insertData) Values(vals ...any) insertData {
	d.values = append(d.values, vals)
	return d
}

func (d insertData) SetMap(m map[string]any) insertData {
	d.setMap = m
	return d
}

func (d insertData) Suffix(sql string, args ...any) insertData {
	d.suffixes = append(d.suffixes, suffixClause{sql: sql, args: args})
	return d
}

func (d insertData) ToSQL(pf placeholderFormat) (string, []any, error) {
	var cols []string
	var allVals [][]any

	if d.setMap != nil {
		cols = make([]string, 0, len(d.setMap))
		var row []any
		for col, val := range d.setMap {
			cols = append(cols, col)
			if row == nil {
				row = make([]any, 0, len(d.setMap))
			}
			row = append(row, val)
		}
		allVals = [][]any{row}
	} else {
		cols = d.columns
		allVals = d.values
	}

	var buf strings.Builder
	var args []any

	buf.WriteString("INSERT INTO ")
	buf.WriteString(d.table)

	if len(cols) > 0 {
	buf.WriteString(" (")
	buf.WriteString(strings.Join(cols, ","))
	buf.WriteString(")")
	}

	if len(allVals) > 0 {
		buf.WriteString(" VALUES ")
		for vi, row := range allVals {
			if vi > 0 {
				buf.WriteString(", ")
			}
			placeholders := make([]string, len(row))
			for i := range row {
				placeholders[i] = "?"
			}
			buf.WriteString("(" + strings.Join(placeholders, ",") + ")")
			args = append(args, row...)
		}
	}

	for _, s := range d.suffixes {
		buf.WriteString(" ")
		buf.WriteString(s.sql)
		args = append(args, s.args...)
	}

	sql := pf(buf.String())
	return sql, args, nil
}

// ─────────────────────────────────────────────────────────────
// updateData
// ─────────────────────────────────────────────────────────────

type setItem struct {
	col  string
	val  any    // for plain SET: the value
	expr string // for expr SET: the SQL template with ? placeholders
	args []any  // for expr SET: args matching the ? placeholders
}

type updateData struct {
	table      string
	sets       []setItem
	wherePreds []Predicate
	orderBy    []string
	limit      *uint64
	suffixes   []suffixClause
}

func (d updateData) Set(col string, val any) updateData {
	d.sets = append(d.sets, setItem{col: col, val: val})
	return d
}

func (d updateData) SetMap(m map[string]any) updateData {
	for col, val := range m {
		d.sets = append(d.sets, setItem{col: col, val: val})
	}
	return d
}

func (d updateData) SetExpr(col string, expr string, args ...any) updateData {
	d.sets = append(d.sets, setItem{col: col, expr: expr, args: args})
	return d
}

func (d updateData) Where(pred Predicate) updateData {
	d.wherePreds = append(d.wherePreds, pred)
	return d
}

func (d updateData) OrderBy(col string) updateData {
	d.orderBy = append(d.orderBy, col)
	return d
}

func (d updateData) Limit(n uint64) updateData {
	d.limit = &n
	return d
}

func (d updateData) Suffix(sql string, args ...any) updateData {
	d.suffixes = append(d.suffixes, suffixClause{sql: sql, args: args})
	return d
}

func (d updateData) ToSQL(pf placeholderFormat) (string, []any, error) {
	var buf strings.Builder
	var args []any

	buf.WriteString("UPDATE ")
	buf.WriteString(d.table)

	if len(d.sets) > 0 {
		buf.WriteString(" SET ")
		for i, s := range d.sets {
			if i > 0 {
				buf.WriteString(", ")
			}
			if s.expr != "" {
				buf.WriteString(s.col)
				buf.WriteString(" = ")
				buf.WriteString(s.expr)
				args = append(args, s.args...)
			} else {
				buf.WriteString(s.col)
				buf.WriteString(" = ?")
				args = append(args, s.val)
			}
		}
	}

	if len(d.wherePreds) > 0 {
		buf.WriteString(" WHERE ")
		for i, pred := range d.wherePreds {
			if i > 0 {
				buf.WriteString(" AND ")
			}
			sql, predArgs, err := pred.ToSQL()
			if err != nil {
				return "", nil, fmt.Errorf("updateData.Where: %w", err)
			}
			buf.WriteString(sql)
			args = append(args, predArgs...)
		}
	}

	if len(d.orderBy) > 0 {
		buf.WriteString(" ORDER BY ")
		buf.WriteString(strings.Join(d.orderBy, ", "))
	}

	if d.limit != nil {
		buf.WriteString(" LIMIT ")
		buf.WriteString(strconv.FormatUint(*d.limit, 10))
	}

	for _, s := range d.suffixes {
		buf.WriteString(" ")
		buf.WriteString(s.sql)
		args = append(args, s.args...)
	}

	sql := pf(buf.String())
	return sql, args, nil
}

// ─────────────────────────────────────────────────────────────
// Window function helpers
// ─────────────────────────────────────────────────────────────

// Window builds a window function expression for use as a SELECT column.
//
//	RowNumber().Over().PartitionBy("dept").OrderBy("salary", DESC).As("rank")
//	// → ROW_NUMBER() OVER (PARTITION BY dept ORDER BY salary DESC) AS rank
//
// Supported function names: ROW_NUMBER, RANK, DENSE_RANK, NTILE,
// LAG, LEAD, FIRST_VALUE, LAST_VALUE, SUM, AVG, COUNT, MIN, MAX.
func Window(funcName string) WindowBuilder {
	return WindowBuilder{fn: funcName}
}

// RowNumber is a shortcut for Window("ROW_NUMBER").
func RowNumber() WindowBuilder { return WindowBuilder{fn: "ROW_NUMBER"} }

// Rank is a shortcut for Window("RANK").
func Rank() WindowBuilder { return WindowBuilder{fn: "RANK"} }

// DenseRank is a shortcut for Window("DENSE_RANK").
func DenseRank() WindowBuilder { return WindowBuilder{fn: "DENSE_RANK"} }

// Ntile is a shortcut for Window("NTILE").
func Ntile() WindowBuilder { return WindowBuilder{fn: "NTILE"} }

// Lag is a shortcut for Window("LAG").
func Lag() WindowBuilder { return WindowBuilder{fn: "LAG"} }

// Lead is a shortcut for Window("LEAD").
func Lead() WindowBuilder { return WindowBuilder{fn: "LEAD"} }

// FirstValue is a shortcut for Window("FIRST_VALUE").
func FirstValue() WindowBuilder { return WindowBuilder{fn: "FIRST_VALUE"} }

// LastValue is a shortcut for Window("LAST_VALUE").
func LastValue() WindowBuilder { return WindowBuilder{fn: "LAST_VALUE"} }

// WindowBuilder builds a window function expression string.
type WindowBuilder struct {
	fn          string
	args        []any
	partitionBy []string
	orderBy     []string
}

// Args sets positional arguments for the window function
// (e.g., LAG(col, 1, 0) would use Args(col, 1, 0)).
func (w WindowBuilder) Args(args ...any) WindowBuilder {
	w.args = args
	return w
}

// Over starts the OVER clause.
func (w WindowBuilder) Over() WindowBuilder { return w }

// PartitionBy adds PARTITION BY columns.
func (w WindowBuilder) PartitionBy(cols ...string) WindowBuilder {
	w.partitionBy = append(w.partitionBy, cols...)
	return w
}

// OrderBy adds an ORDER BY column within the OVER clause.
func (w WindowBuilder) OrderBy(col string, dir SortDir) WindowBuilder {
	w.orderBy = append(w.orderBy, col+" "+string(dir))
	return w
}

// As wraps the expression with an alias for use as a select column.
func (w WindowBuilder) As(alias string) string {
	s := w.String()
	return s + " AS " + alias
}

// String returns the window function expression (without alias).
func (w WindowBuilder) String() string {
	var b strings.Builder
	b.WriteString(w.fn)
	b.WriteString("(")
	for i, arg := range w.args {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprint(&b, arg)
	}
	b.WriteString(")")
	b.WriteString(" OVER (")
	written := false
	if len(w.partitionBy) > 0 {
		b.WriteString("PARTITION BY ")
		b.WriteString(strings.Join(w.partitionBy, ", "))
		written = true
	}
	if len(w.orderBy) > 0 {
		if written {
			b.WriteString(" ")
		}
		b.WriteString("ORDER BY ")
		b.WriteString(strings.Join(w.orderBy, ", "))
	}
	b.WriteString(")")
	return b.String()
}

// ─────────────────────────────────────────────────────────────
// deleteData
// ─────────────────────────────────────────────────────────────

type deleteData struct {
	table      string
	wherePreds []Predicate
	orderBy    []string
	limit      *uint64
	suffixes   []suffixClause
}

func (d deleteData) Where(pred Predicate) deleteData {
	d.wherePreds = append(d.wherePreds, pred)
	return d
}

func (d deleteData) OrderBy(col string) deleteData {
	d.orderBy = append(d.orderBy, col)
	return d
}

func (d deleteData) Limit(n uint64) deleteData {
	d.limit = &n
	return d
}

func (d deleteData) Suffix(sql string, args ...any) deleteData {
	d.suffixes = append(d.suffixes, suffixClause{sql: sql, args: args})
	return d
}

func (d deleteData) ToSQL(pf placeholderFormat) (string, []any, error) {
	var buf strings.Builder
	var args []any

	buf.WriteString("DELETE FROM ")
	buf.WriteString(d.table)

	if len(d.wherePreds) > 0 {
		buf.WriteString(" WHERE ")
		for i, pred := range d.wherePreds {
			if i > 0 {
				buf.WriteString(" AND ")
			}
			sql, predArgs, err := pred.ToSQL()
			if err != nil {
				return "", nil, fmt.Errorf("deleteData.Where: %w", err)
			}
			buf.WriteString(sql)
			args = append(args, predArgs...)
		}
	}

	if len(d.orderBy) > 0 {
		buf.WriteString(" ORDER BY ")
		buf.WriteString(strings.Join(d.orderBy, ", "))
	}

	if d.limit != nil {
		buf.WriteString(" LIMIT ")
		buf.WriteString(strconv.FormatUint(*d.limit, 10))
	}

	for _, s := range d.suffixes {
		buf.WriteString(" ")
		buf.WriteString(s.sql)
		args = append(args, s.args...)
	}

	sql := pf(buf.String())
	return sql, args, nil
}
