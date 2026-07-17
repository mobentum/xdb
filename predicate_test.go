package xdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── eqPred edge cases ───────────────────────────────────

func TestEq_Nil_Produces_IS_NULL(t *testing.T) {
	sql, args, err := Cond.Eq("deleted_at", nil).ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "deleted_at IS NULL", sql)
	assert.Empty(t, args)
}

func TestNotEq_Nil_Produces_IS_NOT_NULL(t *testing.T) {
	sql, args, err := Cond.NotEq("updated_at", nil).ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "updated_at IS NOT NULL", sql)
	assert.Empty(t, args)
}

// ── In edge cases ───────────────────────────────────────

func TestIn_Empty_Produces_1eq0(t *testing.T) {
	sql, args, err := Cond.In("id").ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "1=0", sql)
	assert.Empty(t, args)
}

func TestIn_Single_Value(t *testing.T) {
	sql, args, err := Cond.In("id", 1).ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "id IN (?)", sql)
	assert.Equal(t, []any{1}, args)
}

// ── And/Or edge cases ───────────────────────────────────

func TestAnd_Single_NoParentheses(t *testing.T) {
	sql, args, err := Cond.And(Cond.Eq("a", 1)).ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "a = ?", sql)
	assert.Equal(t, []any{1}, args)
}

func TestAnd_Multiple_WrapsInParentheses(t *testing.T) {
	sql, args, err := Cond.And(Cond.Eq("a", 1), Cond.Eq("b", 2)).ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "(a = ? AND b = ?)", sql)
	assert.Equal(t, []any{1, 2}, args)
}

func TestOr_Single_NoParentheses(t *testing.T) {
	sql, args, err := Cond.Or(Cond.Eq("a", 1)).ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "a = ?", sql)
	assert.Equal(t, []any{1}, args)
}

func TestOr_Multiple_WrapsInParentheses(t *testing.T) {
	sql, args, err := Cond.Or(Cond.Eq("a", 1), Cond.Eq("b", 2)).ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "(a = ? OR b = ?)", sql)
	assert.Equal(t, []any{1, 2}, args)
}

// ── Search edge cases ───────────────────────────────────

func TestSearch_SingleColumn_NoORWrapper(t *testing.T) {
	sql, args, err := Cond.Search("term", "name").ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "(name ILIKE ?)", sql)
	assert.Len(t, args, 1)
	assert.Contains(t, args[0].(string), "term")
}

func TestSearch_MultiColumn_ORed(t *testing.T) {
	sql, args, err := Cond.Search("john", "name", "email").ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "OR")
	assert.Equal(t, "(name ILIKE ? OR email ILIKE ?)", sql)
	assert.Len(t, args, 2)
	assert.Contains(t, args[0].(string), "john")
	assert.Contains(t, args[1].(string), "john")
}

func TestSearch_TermLowercased(t *testing.T) {
	_, args, err := Cond.Search("JOHN", "name").ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "%john%", args[0])
}

// ── Between ─────────────────────────────────────────────

func TestBetween_SQL(t *testing.T) {
	sql, args, err := Cond.Between("age", 18, 65).ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "age >= ? AND age <= ?", sql)
	assert.Equal(t, []any{18, 65}, args)
}

// ── Raw ─────────────────────────────────────────────────

func TestRaw_NoArgs(t *testing.T) {
	sql, args, err := Cond.Raw("col @> ARRAY[1,2]").ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "col @> ARRAY[1,2]", sql)
	assert.Empty(t, args)
}

// ── Placeholder formatting ──────────────────────────────

func TestDollarPlaceholder_Empty(t *testing.T) {
	assert.Equal(t, "", dollarPlaceholder(""))
}

func TestDollarPlaceholder_NoPlaceholders(t *testing.T) {
	assert.Equal(t, "SELECT 1", dollarPlaceholder("SELECT 1"))
}

func TestDollarPlaceholder_ReplacesInOrder(t *testing.T) {
	result := dollarPlaceholder("a = ? AND b = ? AND c = ?")
	assert.Equal(t, "a = $1 AND b = $2 AND c = $3", result)
}

func TestDollarPlaceholder_EscapesNonPlaceholder(t *testing.T) {
	result := dollarPlaceholder("col = ? AND col2 = $$")
	assert.Equal(t, "col = $1 AND col2 = $$", result)
}

func TestQuestionPlaceholder_Preserves(t *testing.T) {
	assert.Equal(t, "a = ? AND b = ?", questionPlaceholder("a = ? AND b = ?"))
}

func TestQuestionPlaceholder_Empty(t *testing.T) {
	assert.Equal(t, "", questionPlaceholder(""))
}

// ── repeated helper ─────────────────────────────────────

func TestRepeated_Zero(t *testing.T) {
	assert.Nil(t, repeated("x", 0))
}

func TestRepeated_Negative(t *testing.T) {
	assert.Nil(t, repeated("x", -1))
}

func TestRepeated_Positive(t *testing.T) {
	r := repeated(42, 3)
	assert.Equal(t, []any{42, 42, 42}, r)
}

func TestAndPred_SingleItem_NoParens(t *testing.T) {
	sql, args, err := Cond.And(Cond.Eq("a", 1)).ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "a = ?", sql)
	assert.Equal(t, []any{1}, args)
}

func TestAndPred_Empty_Produces_1eq1(t *testing.T) {
	sql, args, err := Cond.And().ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "1=1", sql)
	assert.Empty(t, args)
}

func TestOrPred_Empty_Produces_1eq0(t *testing.T) {
	sql, args, err := Cond.Or().ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "1=0", sql)
	assert.Empty(t, args)
}

func TestAndPred_Error_Propagates(t *testing.T) {
	_, _, err := Cond.And(Cond.Eq("a", 1), errPredicate{}).ToSQL()
	assert.ErrorIs(t, err, assert.AnError)
}

func TestOrPred_Error_Propagates(t *testing.T) {
	_, _, err := Cond.Or(Cond.Eq("a", 1), errPredicate{}).ToSQL()
	assert.ErrorIs(t, err, assert.AnError)
}

// ── Predicate interface compliance ──────────────────────

func TestAllCondReturnTypesImplementPredicate(t *testing.T) {
	tests := []struct {
		name string
		p    Predicate
	}{
		{"Eq", Cond.Eq("a", 1)},
		{"NotEq", Cond.NotEq("a", 1)},
		{"Gt", Cond.Gt("a", 1)},
		{"Lt", Cond.Lt("a", 1)},
		{"GtOrEq", Cond.GtOrEq("a", 1)},
		{"LtOrEq", Cond.LtOrEq("a", 1)},
		{"Like", Cond.Like("a", "%x%")},
		{"ILike", Cond.ILike("a", "%x%")},
		{"IsNull", Cond.IsNull("a")},
		{"IsNotNull", Cond.IsNotNull("a")},
		{"In", Cond.In("a", 1, 2)},
		{"Between", Cond.Between("a", 1, 10)},
		{"Raw", Cond.Raw("a")},
		{"And", Cond.And(Cond.Eq("a", 1))},
		{"Or", Cond.Or(Cond.Eq("a", 1))},
		{"Search", Cond.Search("x", "a")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.p.ToSQL()
			assert.NoError(t, err)
			assert.NotEmpty(t, sql)
			// args may be nil for IS NULL / IS NOT NULL / raw with no args
			_ = args
		})
	}
}

// ── Immutability of Predicates ──────────────────────────

func TestCondPredicates_AreImmutable(t *testing.T) {
	// Calling ToSQL on the same predicate twice should produce the same result.
	p := Cond.And(Cond.Eq("a", 1), Cond.Or(Cond.Eq("b", 2), Cond.Eq("c", 3)))
	sql1, args1, err1 := p.ToSQL()
	require.NoError(t, err1)
	sql2, args2, err2 := p.ToSQL()
	require.NoError(t, err2)
	assert.Equal(t, sql1, sql2)
	assert.Equal(t, args1, args2)
}
