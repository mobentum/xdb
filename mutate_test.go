package xdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== INSERT TESTS ====================

func TestInsertBuilder_Columns(t *testing.T) {
	ib := InsertBuilder{data: insertData{table: "users"}}

	ib2 := ib.Columns("id", "name", "email")
	assert.Len(t, ib2.data.columns, 3)
}

func TestInsertBuilder_Values(t *testing.T) {
	ib := InsertBuilder{data: insertData{table: "users", columns: []string{"id", "name", "email"}}}

	ib2 := ib.Values("123", "Alice", "alice@example.com")
	sql, args, err := ib2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "INSERT")
	assert.Len(t, args, 3)
}

func TestInsertBuilder_SetMap(t *testing.T) {
	ib := InsertBuilder{data: insertData{table: "users"}}

	m := map[string]any{
		"id":    "123",
		"name":  "Alice",
		"email": "alice@example.com",
	}
	ib2 := ib.SetMap(m)
	sql, args, err := ib2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "INSERT")
	assert.Len(t, args, 3)
}

func TestInsertBuilder_OnConflict(t *testing.T) {
	ib := InsertBuilder{
		ec:   execContext{driverName: "postgres", pf: dollarPlaceholder},
		data: insertData{table: "users", columns: []string{"id", "name"}, values: [][]any{{"123", "Alice"}}},
	}

	ib2 := ib.OnConflict("(id) DO UPDATE SET name = EXCLUDED.name")
	sql, _, err := ib2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "ON CONFLICT")
}

func TestInsertBuilder_Returning(t *testing.T) {
	ib := InsertBuilder{
		ec:   execContext{driverName: "postgres", pf: dollarPlaceholder},
		data: insertData{table: "users", columns: []string{"id", "name"}, values: [][]any{{"123", "Alice"}}},
	}

	ib2 := ib.Returning("id", "name")
	sql, _, err := ib2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "RETURNING")
}

func TestInsertBuilder_ToSQL(t *testing.T) {
	ib := InsertBuilder{data: insertData{
		table:   "users",
		columns: []string{"id", "name", "email"},
		values:  [][]any{{"123", "Alice", "alice@example.com"}},
	}}

	sql, args, err := ib.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "INSERT INTO")
	assert.Contains(t, sql, "users")
	assert.Len(t, args, 3)
}

// ==================== UPDATE TESTS ====================

func TestUpdateBuilder_Set(t *testing.T) {
	ub := UpdateBuilder{data: updateData{table: "users"}}

	ub2 := ub.Set("name", "Bob").Set("email", "bob@example.com")
	sql, args, err := ub2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "UPDATE")
	assert.Len(t, args, 2)
}

func TestUpdateBuilder_SetMap(t *testing.T) {
	ub := UpdateBuilder{data: updateData{table: "users"}}

	m := map[string]any{
		"name":       "Bob",
		"email":      "bob@example.com",
		"updated_at": "2024-01-01",
	}
	ub2 := ub.SetMap(m)
	sql, args, err := ub2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "UPDATE")
	assert.Len(t, args, 3)
}

func TestUpdateBuilder_Where(t *testing.T) {
	ub := UpdateBuilder{data: updateData{table: "users"}}

	ub2 := ub.Set("name", "Charlie").Where(Cond.Eq("id", "123"))
	sql, args, err := ub2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "WHERE")
	assert.Len(t, args, 2)
}

func TestUpdateBuilder_WhereIf_True(t *testing.T) {
	ub := UpdateBuilder{data: updateData{table: "users"}}

	ub2 := ub.Set("status", "active").WhereIf(true, Cond.Eq("id", "123"))
	sql, args, err := ub2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "WHERE")
	assert.Len(t, args, 2)
}

func TestUpdateBuilder_WhereIf_False(t *testing.T) {
	ub := UpdateBuilder{data: updateData{table: "users"}}

	ub2 := ub.Set("status", "active").WhereIf(false, Cond.Eq("id", "123"))
	sql, args, err := ub2.ToSQL()
	require.NoError(t, err)
	assert.NotContains(t, sql, "WHERE")
	assert.Len(t, args, 1)
}

func TestUpdateBuilder_Returning(t *testing.T) {
	ub := UpdateBuilder{data: updateData{table: "users"}}

	ub2 := ub.Set("name", "Dave").Returning("id", "name")
	sql, _, err := ub2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "RETURNING")
}

func TestUpdateBuilder_OrderBy(t *testing.T) {
	ub := UpdateBuilder{data: updateData{table: "users"}}

	ub2 := ub.Set("status", "archived").OrderBy("created_at", DESC)
	sql, _, err := ub2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "ORDER BY")
}

func TestUpdateBuilder_Limit(t *testing.T) {
	ub := UpdateBuilder{data: updateData{table: "users"}}

	ub2 := ub.Set("status", "reviewed").Limit(10)
	sql, _, err := ub2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "LIMIT")
}

func TestUpdateBuilder_ToSQL(t *testing.T) {
	ub := UpdateBuilder{data: updateData{
		table: "users",
		sets:  []setItem{{col: "name", val: "Eve"}, {col: "email", val: "eve@example.com"}},
	}}

	sql, args, err := ub.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "UPDATE users")
	assert.Len(t, args, 2)
}

// ==================== DELETE TESTS ====================

func TestDeleteBuilder_Where(t *testing.T) {
	db := DeleteBuilder{data: deleteData{table: "users"}}

	db2 := db.Where(Cond.Eq("id", "123"))
	sql, args, err := db2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "DELETE")
	assert.Contains(t, sql, "WHERE")
	assert.Len(t, args, 1)
}

func TestDeleteBuilder_OrderBy(t *testing.T) {
	db := DeleteBuilder{data: deleteData{table: "sessions"}}

	db2 := db.OrderBy("expires_at", ASC)
	sql, _, err := db2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "ORDER BY")
}

func TestDeleteBuilder_Limit(t *testing.T) {
	db := DeleteBuilder{data: deleteData{table: "logs"}}

	db2 := db.Limit(100)
	sql, _, err := db2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "LIMIT")
}

func TestDeleteBuilder_Returning(t *testing.T) {
	db := DeleteBuilder{data: deleteData{table: "users"}}

	db2 := db.Returning("id", "name")
	sql, _, err := db2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "RETURNING")
}

func TestDeleteBuilder_ToSQL(t *testing.T) {
	db := DeleteBuilder{data: deleteData{
		table:      "users",
		wherePreds: []Predicate{Cond.Eq("status", "inactive")},
	}}

	sql, args, err := db.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "DELETE FROM users")
	assert.Len(t, args, 1)
}

func TestDeleteBuilder_Multiple_Conditions(t *testing.T) {
	db := DeleteBuilder{data: deleteData{table: "sessions"}}

	db2 := db.Where(Cond.Lt("expires_at", "2024-01-01")).
		Where(Cond.Eq("status", "expired"))

	sql, args, err := db2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "WHERE")
	assert.Len(t, args, 2)
}

// ==================== COMPLEX TESTS ====================

func TestMutateBuilders_ComplexInsert(t *testing.T) {
	ib := InsertBuilder{data: insertData{table: "orders"}}

	ib2 := ib.Columns("id", "user_id", "amount", "status").
		Values("ord-123", "user-456", 99.99, "pending").
		Returning("id", "user_id", "amount")

	sql, args, err := ib2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "INSERT INTO orders")
	assert.Contains(t, sql, "RETURNING")
	assert.Len(t, args, 4)
}

func TestMutateBuilders_ComplexUpdate(t *testing.T) {
	ub := UpdateBuilder{data: updateData{table: "users"}}

	ub2 := ub.Set("name", "Updated Name").
		Set("email", "new@example.com").
		Where(Cond.Eq("id", "user-123")).
		Returning("id", "name", "email")

	sql, args, err := ub2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "UPDATE users")
	assert.Contains(t, sql, "WHERE")
	assert.Len(t, args, 3)
}

func TestMutateBuilders_ComplexDelete(t *testing.T) {
	db := DeleteBuilder{data: deleteData{table: "logs"}}

	db2 := db.Where(Cond.Lt("created_at", "2023-01-01")).
		Where(Cond.Eq("level", "debug")).
		Limit(1000).
		OrderBy("created_at", ASC)

	sql, args, err := db2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "DELETE FROM logs")
	assert.Contains(t, sql, "WHERE")
	assert.Len(t, args, 2)
}

func TestUpdateBuilder_SetExpr(t *testing.T) {
	ub := UpdateBuilder{data: updateData{table: "products"}}

	ub2 := ub.SetExpr("price", "price * ?", 1.10)
	sql, args, err := ub2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "UPDATE products")
	assert.Contains(t, sql, "price *")
	assert.Len(t, args, 1)
}

func TestDeleteBuilder_WhereIf_True(t *testing.T) {
	db := DeleteBuilder{data: deleteData{table: "users"}}

	db2 := db.WhereIf(true, Cond.Eq("id", "123"))
	sql, args, err := db2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "WHERE")
	assert.Len(t, args, 1)
}

func TestDeleteBuilder_WhereIf_False(t *testing.T) {
	db := DeleteBuilder{data: deleteData{table: "sessions"}}

	db2 := db.WhereIf(false, Cond.Eq("status", "expired"))
	sql, args, err := db2.ToSQL()
	require.NoError(t, err)
	assert.NotContains(t, sql, "WHERE")
	assert.Empty(t, args)
}

func TestInsertBuilder_OnConflict_MySQL(t *testing.T) {
	ib := InsertBuilder{
		ec:   execContext{driverName: "mysql", pf: questionPlaceholder},
		data: insertData{table: "users", columns: []string{"id"}, values: [][]any{{"1"}}},
	}

	ib2 := ib.OnConflict("UPDATE SET name = VALUES(name)")
	sql, _, err := ib2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "ON DUPLICATE KEY")
}

func TestInsertBuilder_NoColumnsOrValues(t *testing.T) {
	ib := InsertBuilder{data: insertData{table: "users"}}
	sql, _, err := ib.ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "INSERT INTO users", sql)
}

func TestUpdateBuilder_SetExpr_MultipleArgs(t *testing.T) {
	ub := UpdateBuilder{data: updateData{table: "accounts"}}
	ub2 := ub.SetExpr("balance", "balance + ? - ?", 100, 20)
	sql, args, err := ub2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "balance + ? - ?")
	assert.Len(t, args, 2)
	assert.Equal(t, 100, args[0])
	assert.Equal(t, 20, args[1])
}

func TestDeleteBuilder_NoWhere_DeletesAll(t *testing.T) {
	db := DeleteBuilder{data: deleteData{table: "logs"}}
	sql, _, err := db.ToSQL()
	require.NoError(t, err)
	assert.Equal(t, "DELETE FROM logs", sql)
}

func TestInsertBuilder_DollarPlaceholder(t *testing.T) {
	ib := InsertBuilder{
		ec:   execContext{pf: dollarPlaceholder},
		data: insertData{table: "users", columns: []string{"id", "name"}, values: [][]any{{"1", "Alice"}}},
	}
	sql, args, err := ib.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "$1")
	assert.Contains(t, sql, "$2")
	assert.Len(t, args, 2)
}

func TestInsertBuilder_QuestionPlaceholder(t *testing.T) {
	ib := InsertBuilder{
		ec:   execContext{pf: questionPlaceholder},
		data: insertData{table: "users", columns: []string{"id", "name"}, values: [][]any{{"1", "Alice"}}},
	}
	sql, args, err := ib.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "?")
	assert.NotContains(t, sql, "$1")
	assert.Len(t, args, 2)
}

func TestUpdateBuilder_DollarPlaceholder(t *testing.T) {
	ub := UpdateBuilder{
		ec:   execContext{pf: dollarPlaceholder},
		data: updateData{table: "users"},
	}
	ub2 := ub.Set("name", "Bob").Where(Cond.Eq("id", "123"))
	sql, args, err := ub2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "$1")
	assert.Contains(t, sql, "$2")
	assert.Len(t, args, 2)
}

func TestMutateBuilders_DeleteWithLimit(t *testing.T) {
	db := DeleteBuilder{data: deleteData{table: "logs"}}
	db2 := db.Where(Cond.Lt("created_at", "2023-01-01")).Limit(100)
	sql, args, err := db2.ToSQL()
	require.NoError(t, err)
	assert.Contains(t, sql, "DELETE FROM logs")
	assert.Contains(t, sql, "WHERE")
	assert.Contains(t, sql, "LIMIT 100")
	assert.Len(t, args, 1)
}
