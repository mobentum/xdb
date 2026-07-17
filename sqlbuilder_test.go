package xdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── selectData direct tests ──────────────────────────────

func TestSelectData_ToSQL_AllParts(t *testing.T) {
	sql, args, err := selectData{
		columns: []string{"id", "name"},
		from:    "users",
		joins:   []joinClause{{joinType: "JOIN", expr: "roles r ON r.id = users.role_id"}},
		wherePreds: []Predicate{
			Cond.Eq("active", true),
			Cond.Gt("age", 18),
		},
		groupBy: []string{"role_id"},
		having:  Cond.Gt("COUNT(*)", 5),
		orderBy: []string{"name ASC"},
		limit:   uint64Ptr(10),
		offset:  uint64Ptr(20),
		suffixes: []suffixClause{
			{sql: "FOR UPDATE"},
		},
	}.ToSQL(dollarPlaceholder)
	require.NoError(t, err)
	assert.Contains(t, sql, "SELECT id, name FROM users")
	assert.Contains(t, sql, "JOIN roles r ON r.id = users.role_id")
	assert.Contains(t, sql, "WHERE")
	assert.Contains(t, sql, "GROUP BY role_id")
	assert.Contains(t, sql, "HAVING")
	assert.Contains(t, sql, "ORDER BY name ASC")
	assert.Contains(t, sql, "LIMIT 10")
	assert.Contains(t, sql, "OFFSET 20")
	assert.Contains(t, sql, "FOR UPDATE")
	assert.Len(t, args, 3)
}

func TestSelectData_NoColumns_DefaultsStar(t *testing.T) {
	sql, _, err := selectData{from: "users"}.ToSQL(questionPlaceholder)
	require.NoError(t, err)
	assert.Equal(t, "SELECT * FROM users", sql)
}

func TestSelectData_NoFrom_OnlySelect(t *testing.T) {
	sql, _, err := selectData{columns: []string{"1"}}.ToSQL(questionPlaceholder)
	require.NoError(t, err)
	assert.Equal(t, "SELECT 1", sql)
}

func TestSelectData_Prefix(t *testing.T) {
	sql, args, err := selectData{
		columns: []string{"id"},
		from:    "cte1",
		prefixes: []suffixClause{
			{sql: "WITH cte1 AS (SELECT 1)", args: nil},
		},
	}.ToSQL(questionPlaceholder)
	require.NoError(t, err)
	assert.Equal(t, "WITH cte1 AS (SELECT 1) SELECT id FROM cte1", sql)
	assert.Empty(t, args)
}

func TestSelectData_Prefix_WithArgs(t *testing.T) {
	sql, args, err := selectData{
		columns: []string{"*"},
		from:    "t",
		wherePreds: []Predicate{Cond.Eq("id", 42)},
		prefixes: []suffixClause{
			{sql: "WITH cte AS (SELECT * FROM t WHERE x = ?)", args: []any{99}},
		},
	}.ToSQL(dollarPlaceholder)
	require.NoError(t, err)
	assert.Contains(t, sql, "WITH cte AS (SELECT * FROM t WHERE x = $1)")
	assert.Contains(t, sql, "WHERE id = $2")
	assert.Len(t, args, 2)
	assert.Equal(t, 99, args[0])
	assert.Equal(t, 42, args[1])
}

func TestSelectData_RemoveLimitOffset(t *testing.T) {
	d := selectData{columns: []string{"*"}, from: "t", limit: uint64Ptr(5), offset: uint64Ptr(10)}
	d2 := d.RemoveLimit().RemoveOffset()
	sql, _, err := d.ToSQL(questionPlaceholder)
	require.NoError(t, err)
	assert.Contains(t, sql, "LIMIT")
	assert.Contains(t, sql, "OFFSET")
	sql2, _, err2 := d2.ToSQL(questionPlaceholder)
	require.NoError(t, err2)
	assert.NotContains(t, sql2, "LIMIT")
	assert.NotContains(t, sql2, "OFFSET")
}

func TestSelectData_Where_Error(t *testing.T) {
	// A predicate that returns an error during ToSQL should propagate.
	_, _, err := selectData{
		columns:    []string{"*"},
		from:       "t",
		wherePreds: []Predicate{errPredicate{}},
	}.ToSQL(questionPlaceholder)
	assert.Error(t, err)
}

func TestSelectData_Having_Error(t *testing.T) {
	_, _, err := selectData{
		columns: []string{"*"},
		from:    "t",
		having:  errPredicate{},
	}.ToSQL(questionPlaceholder)
	assert.Error(t, err)
}

// ── insertData direct tests ──────────────────────────────

func TestInsertData_ToSQL_Basic(t *testing.T) {
	sql, args, err := insertData{
		table:   "users",
		columns: []string{"id", "name"},
		values:  [][]any{{"1", "Alice"}, {"2", "Bob"}},
	}.ToSQL(dollarPlaceholder)
	require.NoError(t, err)
	assert.Equal(t, "INSERT INTO users (id,name) VALUES ($1,$2), ($3,$4)", sql)
	assert.Equal(t, []any{"1", "Alice", "2", "Bob"}, args)
}

func TestInsertData_ToSQL_SetMap(t *testing.T) {
	sql, args, err := insertData{
		table:  "users",
		setMap: map[string]any{"id": "1", "name": "Alice"},
	}.ToSQL(dollarPlaceholder)
	require.NoError(t, err)
	assert.Contains(t, sql, "INSERT INTO users")
	assert.Contains(t, sql, "id")
	assert.Contains(t, sql, "name")
	assert.Len(t, args, 2)
}

func TestInsertData_ToSQL_NoColumns(t *testing.T) {
	sql, _, err := insertData{table: "users"}.ToSQL(questionPlaceholder)
	require.NoError(t, err)
	assert.Equal(t, "INSERT INTO users", sql)
}

func TestInsertData_ToSQL_Suffix(t *testing.T) {
	sql, args, err := insertData{
		table:   "users",
		columns: []string{"id"},
		values:  [][]any{{"1"}},
		suffixes: []suffixClause{
			{sql: "RETURNING id"},
		},
	}.ToSQL(dollarPlaceholder)
	require.NoError(t, err)
	assert.Contains(t, sql, "INSERT INTO users (id) VALUES ($1) RETURNING id")
	assert.Len(t, args, 1)
}

// ── updateData direct tests ──────────────────────────────

func TestUpdateData_ToSQL_Basic(t *testing.T) {
	sql, args, err := updateData{
		table: "users",
		sets:  []setItem{{col: "name", val: "Bob"}, {col: "email", val: "b@x.com"}},
		wherePreds: []Predicate{Cond.Eq("id", "123")},
	}.ToSQL(dollarPlaceholder)
	require.NoError(t, err)
	assert.Contains(t, sql, "UPDATE users SET name = $1, email = $2 WHERE id = $3")
	assert.Equal(t, []any{"Bob", "b@x.com", "123"}, args)
}

func TestUpdateData_ToSQL_WithExpr(t *testing.T) {
	sql, args, err := updateData{
		table: "accounts",
		sets:  []setItem{{col: "balance", expr: "balance + ?", args: []any{100}}},
	}.ToSQL(dollarPlaceholder)
	require.NoError(t, err)
	assert.Contains(t, sql, "SET balance = balance + $1")
	assert.Equal(t, []any{100}, args)
}

func TestUpdateData_ToSQL_OrderByLimit(t *testing.T) {
	sql, args, err := updateData{
		table:   "users",
		sets:    []setItem{{col: "status", val: "archived"}},
		orderBy: []string{"created_at DESC"},
		limit:   uint64Ptr(10),
	}.ToSQL(dollarPlaceholder)
	require.NoError(t, err)
	assert.Contains(t, sql, "ORDER BY created_at DESC")
	assert.Contains(t, sql, "LIMIT 10")
	assert.Equal(t, []any{"archived"}, args)
}

func TestUpdateData_ToSQL_NoSets(t *testing.T) {
	sql, _, err := updateData{table: "users"}.ToSQL(questionPlaceholder)
	require.NoError(t, err)
	assert.Equal(t, "UPDATE users", sql)
}

func TestUpdateData_ToSQL_WhereError(t *testing.T) {
	_, _, err := updateData{
		table:      "t",
		wherePreds: []Predicate{errPredicate{}},
	}.ToSQL(questionPlaceholder)
	assert.Error(t, err)
}

// ── deleteData direct tests ──────────────────────────────

func TestDeleteData_ToSQL_Basic(t *testing.T) {
	sql, args, err := deleteData{
		table:      "logs",
		wherePreds: []Predicate{Cond.Eq("level", "error")},
	}.ToSQL(dollarPlaceholder)
	require.NoError(t, err)
	assert.Equal(t, "DELETE FROM logs WHERE level = $1", sql)
	assert.Equal(t, []any{"error"}, args)
}

func TestDeleteData_ToSQL_OrderByLimit(t *testing.T) {
	sql, args, err := deleteData{
		table:      "sessions",
		wherePreds: []Predicate{Cond.Lt("expires_at", "2024-01-01")},
		orderBy:    []string{"expires_at ASC"},
		limit:      uint64Ptr(100),
	}.ToSQL(dollarPlaceholder)
	require.NoError(t, err)
	assert.Contains(t, sql, "ORDER BY expires_at ASC")
	assert.Contains(t, sql, "LIMIT 100")
	assert.Len(t, args, 1)
}

func TestDeleteData_ToSQL_NoWhere(t *testing.T) {
	sql, _, err := deleteData{table: "logs"}.ToSQL(questionPlaceholder)
	require.NoError(t, err)
	assert.Equal(t, "DELETE FROM logs", sql)
}

func TestDeleteData_ToSQL_Suffix(t *testing.T) {
	sql, args, err := deleteData{
		table: "t",
		suffixes: []suffixClause{
			{sql: "RETURNING id"},
		},
	}.ToSQL(dollarPlaceholder)
	require.NoError(t, err)
	assert.Equal(t, "DELETE FROM t RETURNING id", sql)
	assert.Empty(t, args)
}

func TestDeleteData_ToSQL_WhereError(t *testing.T) {
	_, _, err := deleteData{
		table:      "t",
		wherePreds: []Predicate{errPredicate{}},
	}.ToSQL(questionPlaceholder)
	assert.Error(t, err)
}

func TestSelectData_Columns(t *testing.T) {
	d := selectData{}
	d2 := d.Columns("a", "b")
	assert.Equal(t, []string{"a", "b"}, d2.columns)
	// immutability
	assert.Empty(t, d.columns)
}

func TestSelectData_Column(t *testing.T) {
	d := selectData{columns: []string{"a"}}
	d2 := d.Column("b")
	assert.Equal(t, []string{"a", "b"}, d2.columns)
	// immutability
	assert.Equal(t, []string{"a"}, d.columns)
}

func TestSelectData_PrefixMethod(t *testing.T) {
	d := selectData{columns: []string{"id"}, from: "t"}
	d2 := d.Prefix("WITH cte AS (SELECT 1)")
	sql, _, err := d2.ToSQL(questionPlaceholder)
	require.NoError(t, err)
	assert.Equal(t, "WITH cte AS (SELECT 1) SELECT id FROM t", sql)
	// immutability
	sqlOrig, _, _ := d.ToSQL(questionPlaceholder)
	assert.NotContains(t, sqlOrig, "WITH")
}

func TestSelectData_PrefixMethod_WithArgs(t *testing.T) {
	d := selectData{columns: []string{"*"}, from: "t", wherePreds: []Predicate{Cond.Eq("id", 1)}}
	d2 := d.Prefix("WITH cte AS (SELECT * FROM other WHERE x = ?)", 99)
	sql, args, err := d2.ToSQL(dollarPlaceholder)
	require.NoError(t, err)
	assert.Contains(t, sql, "$1")
	assert.Len(t, args, 2)
	assert.Equal(t, 99, args[0])
	assert.Equal(t, 1, args[1])
}

// ── UNION / INTERSECT / EXCEPT ─────────────────────────

func TestSelectData_Union(t *testing.T) {
	main := selectData{columns: []string{"id"}, from: "t1", wherePreds: []Predicate{Cond.Eq("active", true)}}
	other := selectData{columns: []string{"id"}, from: "t2", wherePreds: []Predicate{Cond.Eq("active", true)}}
	d := main.Union("UNION", false, other)
	d.orderBy = []string{"id ASC"}

	sql, args, err := d.ToSQL(dollarPlaceholder)
	require.NoError(t, err)
	assert.Contains(t, sql, "SELECT id FROM t1 WHERE active = $1")
	assert.Contains(t, sql, "UNION")
	assert.Contains(t, sql, "SELECT id FROM t2 WHERE active = $2")
	assert.Contains(t, sql, "ORDER BY id ASC")
	assert.Len(t, args, 2)
}

func TestSelectData_UnionAll(t *testing.T) {
	main := selectData{columns: []string{"id"}, from: "t1"}
	other := selectData{columns: []string{"id"}, from: "t2"}
	d := main.Union("UNION", true, other)

	sql, _, err := d.ToSQL(questionPlaceholder)
	require.NoError(t, err)
	assert.Contains(t, sql, "UNION ALL")
}

func TestSelectData_Intersect(t *testing.T) {
	main := selectData{columns: []string{"id"}, from: "t1"}
	other := selectData{columns: []string{"id"}, from: "t2"}
	d := main.Union("INTERSECT", false, other)

	sql, _, err := d.ToSQL(questionPlaceholder)
	require.NoError(t, err)
	assert.Contains(t, sql, "INTERSECT")
}

func TestSelectData_Except(t *testing.T) {
	main := selectData{columns: []string{"id"}, from: "t1"}
	other := selectData{columns: []string{"id"}, from: "t2"}
	d := main.Union("EXCEPT", false, other)

	sql, _, err := d.ToSQL(questionPlaceholder)
	require.NoError(t, err)
	assert.Contains(t, sql, "EXCEPT")
}

// ── FromSubquery ────────────────────────────────────────

func TestSelectData_FromSubquery(t *testing.T) {
	inner := selectData{columns: []string{"id", "name"}, from: "users", wherePreds: []Predicate{Cond.Eq("active", true)}}
	d := selectData{columns: []string{"*"}}
	d2 := d.FromSubquery(inner, "sub")

	sql, args, err := d2.ToSQL(dollarPlaceholder)
	require.NoError(t, err)
	assert.Contains(t, sql, "SELECT * FROM (SELECT id, name FROM users WHERE active = $1) AS sub")
	assert.Equal(t, []any{true}, args)
}

func TestSelectData_FromSubquery_ReplacesFrom(t *testing.T) {
	d := selectData{columns: []string{"*"}, from: "users"}
	inner := selectData{columns: []string{"id"}, from: "other"}
	d2 := d.FromSubquery(inner, "sub")

	sql, _, err := d2.ToSQL(questionPlaceholder)
	require.NoError(t, err)
	assert.Contains(t, sql, "FROM (SELECT id FROM other) AS sub")
	assert.NotContains(t, sql, "FROM users")
}

// ── WindowBuilder ───────────────────────────────────────

func TestWindowBuilder_RowNumber(t *testing.T) {
	w := RowNumber().Over().PartitionBy("dept").OrderBy("salary", DESC).As("rank")
	assert.Equal(t, "ROW_NUMBER() OVER (PARTITION BY dept ORDER BY salary DESC) AS rank", w)
}

func TestWindowBuilder_Rank(t *testing.T) {
	w := Rank().Over().OrderBy("total", DESC).As("rank")
	assert.Equal(t, "RANK() OVER (ORDER BY total DESC) AS rank", w)
}

func TestWindowBuilder_Lag(t *testing.T) {
	w := Lag().Args("amount", 1).Over().OrderBy("date", ASC).As("prev")
	assert.Equal(t, `LAG(amount, 1) OVER (ORDER BY date ASC) AS prev`, w)
}

func TestWindowBuilder_String_NoAlias(t *testing.T) {
	w := RowNumber().Over().PartitionBy("dept").OrderBy("salary", DESC).String()
	assert.Equal(t, "ROW_NUMBER() OVER (PARTITION BY dept ORDER BY salary DESC)", w)
}

func TestWindowBuilder_NoPartitionBy(t *testing.T) {
	w := RowNumber().Over().OrderBy("id", ASC).As("rn")
	assert.Equal(t, "ROW_NUMBER() OVER (ORDER BY id ASC) AS rn", w)
}

func TestWindowBuilder_Empty(t *testing.T) {
	w := RowNumber().Over().String()
	assert.Equal(t, "ROW_NUMBER() OVER ()", w)
}

func TestWindowBuilder_Shortcuts(t *testing.T) {
	assert.Contains(t, Window("SUM").Over().OrderBy("x", ASC).As("total"), "SUM()")
	assert.Contains(t, Ntile().Over().PartitionBy("g").As("tile"), "NTILE()")
	assert.Contains(t, Lead().Args("col", 1).Over().OrderBy("id", ASC).As("next"), "LEAD(col, 1)")
	assert.Contains(t, FirstValue().Over().OrderBy("id", ASC).As("first"), "FIRST_VALUE()")
	assert.Contains(t, LastValue().Over().OrderBy("id", ASC).As("last"), "LAST_VALUE()")
}

func TestWindowBuilder_DenseRank(t *testing.T) {
	w := DenseRank().Over().PartitionBy("region").OrderBy("sales", DESC).As("dense_rank")
	assert.Contains(t, w, "DENSE_RANK() OVER")
	assert.Contains(t, w, "PARTITION BY region")
}

func TestSelectBuilder_IntersectAll(t *testing.T) {
	a := SelectBuilder{data: selectData{columns: []string{"id"}, from: "t1"}}
	b := SelectBuilder{data: selectData{columns: []string{"id"}, from: "t2"}}
	sql, _, err := a.IntersectAll(b).ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "INTERSECT ALL")
}

func TestSelectBuilder_ExceptAll(t *testing.T) {
	a := SelectBuilder{data: selectData{columns: []string{"id"}, from: "t1"}}
	b := SelectBuilder{data: selectData{columns: []string{"id"}, from: "t2"}}
	sql, _, err := a.ExceptAll(b).ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "EXCEPT ALL")
}

// ── Helpers ──────────────────────────────────────────────

func uint64Ptr(n uint64) *uint64 { return &n }

// errPredicate is a Predicate that always returns an error.
type errPredicate struct{}

func (errPredicate) ToSQL() (string, []any, error) {
	return "", nil, assert.AnError
}
