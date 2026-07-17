package xdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectBuilder_From(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"id", "name"}}}

	sb2 := sb.From("users")
	assert.NotEmpty(t, sb2.data.from)
}

func TestSelectBuilder_Join(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "orders"}}

	sb2 := sb.Join("users u ON u.id = orders.user_id")
	sql, _, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "JOIN")
}

func TestSelectBuilder_LeftJoin(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "orders"}}

	sb2 := sb.LeftJoin("users u ON u.id = orders.user_id")
	sql, _, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "LEFT JOIN")
}

func TestSelectBuilder_Where(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "users"}}

	sb2 := sb.Where(Cond.Eq("active", true))
	sql, args, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "WHERE")
	assert.Len(t, args, 1)
}

func TestSelectBuilder_WhereIf_True(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "users"}}

	sb2 := sb.WhereIf(true, Cond.Eq("role", "admin"))
	sql, args, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "WHERE")
	assert.Len(t, args, 1)
}

func TestSelectBuilder_WhereIf_False(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "users"}}

	sb2 := sb.WhereIf(false, Cond.Eq("role", "admin"))
	sql, args, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.NotContains(t, sql, "WHERE")
	assert.Empty(t, args)
}

func TestSelectBuilder_GroupBy(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"category", "COUNT(*)"}, from: "orders"}}

	sb2 := sb.GroupBy("category")
	sql, _, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "GROUP BY")
}

func TestSelectBuilder_GroupBy_Multiple(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"region", "product", "SUM(amount)"}, from: "orders"}}

	sb2 := sb.GroupBy("region", "product")
	sql, _, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "GROUP BY")
}

func TestSelectBuilder_Having(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"category", "COUNT(*)"}, from: "orders"}}

	sb2 := sb.GroupBy("category").Having(Cond.Gt("COUNT(*)", 100))
	sql, _, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "HAVING")
}

func TestSelectBuilder_OrderBy_Asc(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "users"}}

	sb2 := sb.OrderBy("name", ASC)
	sql, _, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "ORDER BY")
	assert.Contains(t, sql, "ASC")
}

func TestSelectBuilder_OrderBy_Desc(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "users"}}

	sb2 := sb.OrderBy("created_at", DESC)
	sql, _, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "ORDER BY")
	assert.Contains(t, sql, "DESC")
}

func TestSelectBuilder_OrderBy_EmptyColumn(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "users"}}

	sb2 := sb.OrderBy("", DESC)
	sql, _, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.NotContains(t, sql, "ORDER BY")
}

func TestSelectBuilder_AllowSort_Whitelist(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "users"}}

	sb2 := sb.AllowSort("name", "email", "created_at")
	sb3 := sb2.OrderBy("name", ASC)
	sql, _, err := sb3.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "ORDER BY")
}

func TestSelectBuilder_AllowSort_Reject_Unlisted(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "users"}}

	sb2 := sb.AllowSort("name", "email")
	sb3 := sb2.OrderBy("password", ASC)
	sql, _, err := sb3.ToSQL()
	require.NoError(t, err)
	assert.NotContains(t, sql, "ORDER BY")
}

func TestSelectBuilder_Paginate(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "users"}}

	sb2 := sb.Paginate(Page{Number: 2, Size: 20})
	sql, _, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "LIMIT")
	assert.Contains(t, sql, "OFFSET")
}

func TestSelectBuilder_Suffix(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "users"}}

	sb2 := sb.Suffix("FOR UPDATE")
	sql, _, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "FOR UPDATE")
}

func TestSelectBuilder_ToSQL_Simple(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"id", "name"}, from: "users"}}

	sql, args, err := sb.ToSQL()
	require.NoError(t, err)
	assert.NotEmpty(t, sql)
	assert.Contains(t, sql, "SELECT")
	assert.Contains(t, sql, "FROM")
	assert.Empty(t, args)
}

func TestSelectBuilder_Chain_Multiple_Conditions(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "users"}}

	sb2 := sb.Where(Cond.Eq("status", "active")).
		Where(Cond.Gt("age", 18)).
		OrderBy("name", ASC)

	sql, args, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "WHERE")
	assert.Len(t, args, 2)
}

func TestSelectBuilder_Immutability(t *testing.T) {
	original := SelectBuilder{data: selectData{columns: []string{"*"}, from: "users"}}

	modified := original.Where(Cond.Eq("id", 123))

	origSQL, origArgs, _ := original.ToSQL()
	assert.NotContains(t, origSQL, "WHERE")
	assert.Empty(t, origArgs)

	modSQL, modArgs, _ := modified.ToSQL()
	assert.Contains(t, modSQL, "WHERE")
	assert.Len(t, modArgs, 1)
}

func TestSelectBuilder_Multiple_OrderBy(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "users"}}

	sb2 := sb.OrderBy("region", ASC).OrderBy("name", ASC)
	sql, _, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "ORDER BY")
}

func TestSelectBuilder_NoColumns_DefaultsToStar(t *testing.T) {
	sb := SelectBuilder{data: selectData{from: "users"}}
	sql, _, err := sb.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "SELECT *")
}

func TestSelectBuilder_Limit(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "users"}}
	sb2 := sb.Limit(10)
	sql, _, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "LIMIT 10")
}

func TestSelectBuilder_Offset(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "users"}}
	sb2 := sb.Offset(20)
	sql, _, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "OFFSET 20")
}

func TestSelectBuilder_RemoveLimit(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "users"}}
	sb2 := sb.Limit(10).RemoveLimit()
	sql, _, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.NotContains(t, sql, "LIMIT")
}

func TestSelectBuilder_RemoveOffset(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "users"}}
	sb2 := sb.Offset(20).RemoveOffset()
	sql, _, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.NotContains(t, sql, "OFFSET")
}

func TestSelectBuilder_SuffixWithArgs(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "users"}}
	sb2 := sb.Suffix("FOR UPDATE OF ?", "orders")
	sql, args, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "FOR UPDATE OF")
	assert.Len(t, args, 1)
	assert.Equal(t, "orders", args[0])
}

func TestSelectBuilder_DollarPlaceholder(t *testing.T) {
	sb := SelectBuilder{
		ec:   execContext{pf: dollarPlaceholder},
		data: selectData{columns: []string{"*"}, from: "users"},
	}
	sb2 := sb.Where(Cond.Eq("id", 1))
	sql, args, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "$1")
	assert.Len(t, args, 1)
}

func TestSelectBuilder_QuestionPlaceholder(t *testing.T) {
	sb := SelectBuilder{
		ec:   execContext{pf: questionPlaceholder},
		data: selectData{columns: []string{"*"}, from: "users"},
	}
	sb2 := sb.Where(Cond.Eq("id", 1))
	sql, args, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "?")
	assert.NotContains(t, sql, "$1")
	assert.Len(t, args, 1)
}

func TestSelectBuilder_ToSQL_WithNoDB_UsesQuestionPlaceholder(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "users"}}
	sb2 := sb.Where(Cond.Eq("id", 1))
	sql, args, err := sb2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "?")
	assert.Len(t, args, 1)
}

func TestSelectBuilder_AllMethods_ReturnCopy(t *testing.T) {
	base := SelectBuilder{
		ec:   execContext{pf: dollarPlaceholder},
		data: selectData{columns: []string{"*"}, from: "users"},
	}
	_ = base.From("orders")
	_ = base.Join("x ON x.id = y.id")
	_ = base.Where(Cond.Eq("a", 1))
	_ = base.GroupBy("a")
	_ = base.OrderBy("a", ASC)
	_ = base.Limit(5)
	_ = base.Offset(10)
	_ = base.Suffix("FOR UPDATE")
	_ = base.Paginate(Page{Number: 2, Size: 10})

	sql, _, _ := base.ToSQL()
	assert.NotContains(t, sql, "orders")
	assert.NotContains(t, sql, "JOIN")
	assert.NotContains(t, sql, "WHERE")
	assert.NotContains(t, sql, "GROUP BY")
	assert.NotContains(t, sql, "ORDER BY")
	assert.NotContains(t, sql, "LIMIT")
	assert.NotContains(t, sql, "OFFSET")
	assert.NotContains(t, sql, "FOR UPDATE")
}
