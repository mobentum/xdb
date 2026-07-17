package xdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCTEBuilder_WithCTE(t *testing.T) {
	b := NewCTEBuilder()

	b1 := b.WithCTE("cte1", "SELECT * FROM table1")
	assert.Len(t, b1.ctes, 1)
	assert.Equal(t, "cte1", b1.ctes[0].name)

	b2 := b1.WithCTE("cte2", "SELECT * FROM cte1")
	assert.Len(t, b2.ctes, 2)
}

func TestCTEBuilder_Select(t *testing.T) {
	b := NewCTEBuilder().
		WithCTE("temp", "SELECT 1 AS id")

	sb := b.Select("id")
	assert.Len(t, sb.ctes, 1)
}

func TestSelectBuilder_WithCTE(t *testing.T) {
	sb := SelectBuilder{
		data: selectData{columns: []string{"id"}},
		ctes: []CTE{},
	}

	sb2 := sb.WithCTE("cte1", "SELECT * FROM table1")
	assert.Len(t, sb2.ctes, 1)

	sb3 := sb2.WithCTE("cte2", "SELECT * FROM cte1")
	assert.Len(t, sb3.ctes, 2)
}

func TestSelectBuilder_ToSQL_WithCTE(t *testing.T) {
	sb := SelectBuilder{
		data: selectData{columns: []string{"id"}, from: "cte1"},
		ctes: []CTE{{name: "cte1", sql: "SELECT 1 AS id", args: nil}},
	}

	sql, _, err := sb.ToSQL()
	require.NoError(t, err)
	assert.NotEmpty(t, sql)
	assert.Contains(t, sql, "WITH")
	assert.Contains(t, sql, "cte1")
}

func TestBuildWithClause(t *testing.T) {
	ctes := []CTE{
		{name: "cte1", sql: "SELECT 1", args: []any{}},
		{name: "cte2", sql: "SELECT * FROM cte1", args: []any{}},
	}

	withClause, args := buildWithClause(ctes)

	assert.NotEmpty(t, withClause)
	assert.Contains(t, withClause, "WITH")
	assert.Contains(t, withClause, "cte1")
	assert.Contains(t, withClause, "cte2")
	assert.Empty(t, args)
}

func TestBuildWithClause_WithArgs(t *testing.T) {
	ctes := []CTE{
		{name: "cte1", sql: "SELECT * FROM table1 WHERE id = ?", args: []any{123}},
	}

	_, args := buildWithClause(ctes)

	assert.Len(t, args, 1)
	assert.Equal(t, 123, args[0])
}

func TestPrependWithClause(t *testing.T) {
	withClause := "WITH cte1 AS (SELECT 1) "
	withArgs := []any{}
	mainSQL := "SELECT * FROM cte1"
	mainArgs := []any{123}

	finalSQL, finalArgs := prependWithClause(withClause, withArgs, mainSQL, mainArgs)

	assert.Equal(t, withClause+mainSQL, finalSQL)
	assert.Len(t, finalArgs, 1)
	assert.Equal(t, 123, finalArgs[0])
}

func TestCTEBuilder_WithSelectCTE(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "users"}}
	b := NewCTEBuilder().WithSelectCTE("active_users", sb)
	assert.Len(t, b.ctes, 1)
}

func TestSelectBuilder_WithSelectCTE(t *testing.T) {
	inner := SelectBuilder{data: selectData{columns: []string{"id"}, from: "users"}}
	sb := SelectBuilder{
		data: selectData{columns: []string{"id"}, from: "active_users"},
		ctes: []CTE{},
	}
	sb2 := sb.WithSelectCTE("active_users", inner)
	sql, args, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "WITH")
	assert.Contains(t, sql, "active_users AS")
	assert.Empty(t, args)
}

func TestBuildWithClause_Empty(t *testing.T) {
	withClause, args := buildWithClause([]CTE{})
	assert.Empty(t, withClause)
	assert.Empty(t, args)
}

func TestPrependWithClause_EmptyWith(t *testing.T) {
	mainSQL := "SELECT * FROM table1"
	mainArgs := []any{456}

	finalSQL, finalArgs := prependWithClause("", []any{}, mainSQL, mainArgs)

	assert.Equal(t, mainSQL, finalSQL)
	assert.Len(t, finalArgs, 1)
	assert.Equal(t, 456, finalArgs[0])
}
