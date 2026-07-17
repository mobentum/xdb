package xdb

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCond_Eq(t *testing.T) {
	pred := Cond.Eq("id", 123)
	sql, args, err := pred.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "id")
	assert.Len(t, args, 1)
	assert.Equal(t, 123, args[0])
}

func TestCond_NotEq(t *testing.T) {
	pred := Cond.NotEq("status", "inactive")
	sql, args, err := pred.ToSQL()
	require.NoError(t, err)
	assert.True(t, strings.Contains(sql, "!=") || strings.Contains(sql, "<>"))
	assert.Len(t, args, 1)
	assert.Equal(t, "inactive", args[0])
}

func TestCond_Gt(t *testing.T) {
	sql, args, err := Cond.Gt("age", 18).ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, ">")
	assert.Equal(t, []any{18}, args)
}

func TestCond_Lt(t *testing.T) {
	sql, args, err := Cond.Lt("price", 100).ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "<")
	assert.Equal(t, []any{100}, args)
}

func TestCond_GtOrEq(t *testing.T) {
	sql, args, err := Cond.GtOrEq("score", 50).ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, ">=")
	assert.Equal(t, []any{50}, args)
}

func TestCond_LtOrEq(t *testing.T) {
	sql, args, err := Cond.LtOrEq("value", 75).ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "<=")
	assert.Equal(t, []any{75}, args)
}

func TestCond_Like(t *testing.T) {
	sql, args, err := Cond.Like("name", "%John%").ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "LIKE")
	assert.Equal(t, []any{"%John%"}, args)
}

func TestCond_ILike(t *testing.T) {
	sql, args, err := Cond.ILike("email", "%gmail%").ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "ILIKE")
	assert.Equal(t, []any{"%gmail%"}, args)
}

func TestCond_IsNull(t *testing.T) {
	sql, args, err := Cond.IsNull("deleted_at").ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "deleted_at IS NULL", sql)
	assert.Empty(t, args)
}

func TestCond_IsNotNull(t *testing.T) {
	sql, args, err := Cond.IsNotNull("updated_at").ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "updated_at IS NOT NULL", sql)
	assert.Empty(t, args)
}

func TestCond_In(t *testing.T) {
	sql, args, err := Cond.In("status", "active", "pending", "completed").ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "IN")
	assert.Len(t, args, 3)
}

func TestCond_In_Empty(t *testing.T) {
	sql, args, err := Cond.In("id").ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "1=0", sql)
	assert.Empty(t, args)
}

func TestCond_Between(t *testing.T) {
	sql, args, err := Cond.Between("age", 18, 65).ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, ">=")
	assert.Contains(t, sql, "<=")
	assert.Equal(t, []any{18, 65}, args)
}

func TestCond_And(t *testing.T) {
	pred := Cond.And(Cond.Eq("status", "active"), Cond.Gt("age", 18))
	sql, args, err := pred.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "AND")
	assert.Len(t, args, 2)
}

func TestCond_And_Empty(t *testing.T) {
	sql, args, err := Cond.And().ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "1=1", sql)
	assert.Empty(t, args)
}

func TestCond_Or(t *testing.T) {
	pred := Cond.Or(Cond.Eq("role", "admin"), Cond.Eq("role", "moderator"))
	sql, args, err := pred.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "OR")
	assert.Len(t, args, 2)
}

func TestCond_Or_Empty(t *testing.T) {
	sql, args, err := Cond.Or().ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "1=0", sql)
	assert.Empty(t, args)
}

func TestCond_Search(t *testing.T) {
	pred := Cond.Search("john", "name", "email")
	sql, args, err := pred.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "ILIKE")
	assert.Len(t, args, 2)
	assert.Contains(t, args[0].(string), "john")
}

func TestCond_Search_NoColumns(t *testing.T) {
	sql, args, err := Cond.Search("term").ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "1=0", sql)
	assert.Empty(t, args)
}

func TestCond_Raw(t *testing.T) {
	sql, args, err := Cond.Raw("col @> ?", `{"tag"}`).ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "@>")
	assert.Len(t, args, 1)
}

func TestCond_Nested_And_Or(t *testing.T) {
	pred := Cond.And(
		Cond.Eq("status", "active"),
		Cond.Or(Cond.Eq("role", "admin"), Cond.Eq("role", "moderator")),
	)
	sql, args, err := pred.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "AND")
	assert.Contains(t, sql, "OR")
	assert.Len(t, args, 3)
}

// ── JSONB / Array ──────────────────────────────────────

func TestCond_JsonContains(t *testing.T) {
	sql, args, err := Cond.JsonContains("metadata", `{"status":"active"}`).ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "metadata @> ?", sql)
	assert.Equal(t, []any{`{"status":"active"}`}, args)
}

func TestCond_JsonHasKey(t *testing.T) {
	sql, args, err := Cond.JsonHasKey("metadata", "description").ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "jsonb_exists(metadata, ?)", sql)
	assert.Equal(t, []any{"description"}, args)
}

func TestCond_ArrayOverlaps(t *testing.T) {
	sql, args, err := Cond.ArrayOverlaps("roles", "admin", "editor").ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "roles && ARRAY[?,?]", sql)
	assert.Equal(t, []any{"admin", "editor"}, args)
}

func TestCond_ArrayContains(t *testing.T) {
	sql, args, err := Cond.ArrayContains("tags", "go", "database").ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "tags @> ARRAY[?,?]", sql)
	assert.Equal(t, []any{"go", "database"}, args)
}
