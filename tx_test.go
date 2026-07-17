package xdb

import (
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func TestTxDB_Select_ReturnsBuilder(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: questionPlaceholder}}
	sb := tx.Select("id", "name")
	assert.Equal(t, []string{"id", "name"}, sb.data.columns)
}

func TestTxDB_Insert_ReturnsBuilder(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: questionPlaceholder}}
	ib := tx.Insert("users")
	assert.Equal(t, "users", ib.data.table)
}

func TestTxDB_Update_ReturnsBuilder(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: questionPlaceholder}}
	ub := tx.Update("users")
	assert.Equal(t, "users", ub.data.table)
}

func TestTxDB_Delete_ReturnsBuilder(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: questionPlaceholder}}
	db := tx.Delete("users")
	assert.Equal(t, "users", db.data.table)
}

// ── TxSelectBuilder via TxDB.Select ─────────────────────

func TestTxSelectBuilder_Where(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: dollarPlaceholder}}
	sb := tx.Select("*").From("users").Where(Cond.Eq("id", 1))
	sql, args, err := sb.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "WHERE")
	assert.Equal(t, []any{1}, args)
}

func TestTxSelectBuilder_WhereIf(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: dollarPlaceholder}}
	sbTrue := tx.Select("*").From("users").WhereIf(true, Cond.Eq("a", 1))
	sql, _, _ := sbTrue.ToSQL()
	assert.Contains(t, sql, "WHERE")

	sbFalse := tx.Select("*").From("users").WhereIf(false, Cond.Eq("a", 1))
	sql, _, _ = sbFalse.ToSQL()
	assert.NotContains(t, sql, "WHERE")
}

func TestTxSelectBuilder_OrderBy(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: questionPlaceholder}}
	sb := tx.Select("*").From("users").OrderBy("name", DESC)
	sql, _, err := sb.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "ORDER BY name DESC")
}

func TestTxSelectBuilder_GroupBy(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: questionPlaceholder}}
	sb := tx.Select("category", "COUNT(*)").From("orders").GroupBy("category")
	sql, _, err := sb.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "GROUP BY category")
}

func TestTxSelectBuilder_Join(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: questionPlaceholder}}
	sb := tx.Select("*").From("orders").Join("users u ON u.id = orders.user_id")
	sql, _, err := sb.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "JOIN")
}

func TestTxSelectBuilder_LeftJoin(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: questionPlaceholder}}
	sb := tx.Select("*").From("orders").LeftJoin("users u ON u.id = orders.user_id")
	sql, _, err := sb.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "LEFT JOIN")
}

func TestTxSelectBuilder_Limit(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: questionPlaceholder}}
	sb := tx.Select("*").From("users").Limit(10)
	sql, _, err := sb.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "LIMIT 10")
}

func TestTxSelectBuilder_Offset(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: questionPlaceholder}}
	sb := tx.Select("*").From("users").Offset(5)
	sql, _, err := sb.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "OFFSET 5")
}

func TestTxSelectBuilder_Suffix(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: questionPlaceholder}}
	sb := tx.Select("*").From("orders").Suffix("FOR UPDATE")
	sql, _, err := sb.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "FOR UPDATE")
}

func TestTxSelectBuilder_Lock(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: questionPlaceholder}}
	sb := tx.Select("*").From("orders").Lock(ForUpdate().SkipLocked())
	sql, _, err := sb.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "FOR UPDATE SKIP LOCKED")
}

func TestTxSelectBuilder_ToSQL(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: dollarPlaceholder}}
	sb := tx.Select("*").From("users").Where(Cond.Eq("id", 42))
	sql, args, err := sb.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "SELECT * FROM users WHERE id = $1")
	assert.Equal(t, []any{42}, args)
}

// ── TxInsertBuilder via TxDB.Insert ─────────────────────

func TestTxInsertBuilder_ColumnsValues(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: dollarPlaceholder}}
	ib := tx.Insert("users").Columns("id", "name").Values("1", "Alice")
	sql, args, err := ib.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "INSERT INTO users")
	assert.Len(t, args, 2)
}

func TestTxInsertBuilder_SetMap(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: questionPlaceholder}}
	ib := tx.Insert("users").SetMap(map[string]any{"id": "1", "name": "Alice"})
	sql, args, err := ib.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "INSERT INTO users")
	assert.Len(t, args, 2)
}

func TestTxInsertBuilder_OnConflict(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: dollarPlaceholder}}
	ib := tx.Insert("users").Columns("id").Values("1").OnConflict("(id) DO NOTHING")
	sql, _, err := ib.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "ON CONFLICT")
}

func TestTxInsertBuilder_Returning(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: dollarPlaceholder}}
	ib := tx.Insert("users").Columns("id").Values("1").Returning("id")
	sql, _, err := ib.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "RETURNING id")
}

// ── TxUpdateBuilder via TxDB.Update ─────────────────────

func TestTxUpdateBuilder_SetWhere(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: dollarPlaceholder}}
	ub := tx.Update("users").Set("name", "Bob").Where(Cond.Eq("id", "123"))
	sql, args, err := ub.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "UPDATE users SET name = $1 WHERE id = $2")
	assert.Equal(t, []any{"Bob", "123"}, args)
}

func TestTxUpdateBuilder_SetMap(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: questionPlaceholder}}
	ub := tx.Update("users").SetMap(map[string]any{"name": "Bob"})
	sql, _, err := ub.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "SET name = ?")
}

func TestTxUpdateBuilder_SetExpr(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: dollarPlaceholder}}
	ub := tx.Update("accounts").SetExpr("balance", "balance + ?", 50)
	sql, args, err := ub.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "SET balance = balance + $1")
	assert.Equal(t, []any{50}, args)
}

func TestTxUpdateBuilder_WhereIf(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: questionPlaceholder}}
	ubTrue := tx.Update("users").Set("a", 1).WhereIf(true, Cond.Eq("b", 2))
	sql, _, _ := ubTrue.ToSQL()
	assert.Contains(t, sql, "WHERE")

	ubFalse := tx.Update("users").Set("a", 1).WhereIf(false, Cond.Eq("b", 2))
	sql, _, _ = ubFalse.ToSQL()
	assert.NotContains(t, sql, "WHERE")
}

func TestTxUpdateBuilder_Returning(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: dollarPlaceholder}}
	ub := tx.Update("users").Set("name", "Bob").Returning("id")
	sql, _, err := ub.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "RETURNING id")
}

// ── TxDeleteBuilder via TxDB.Delete ─────────────────────

func TestTxDeleteBuilder_Where(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: dollarPlaceholder}}
	db := tx.Delete("sessions").Where(Cond.Lt("expires_at", "2024-01-01"))
	sql, args, err := db.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "DELETE FROM sessions")
	assert.Len(t, args, 1)
}

func TestTxDeleteBuilder_WhereIf(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: questionPlaceholder}}
	dbTrue := tx.Delete("logs").WhereIf(true, Cond.Eq("level", "error"))
	sql, _, _ := dbTrue.ToSQL()
	assert.Contains(t, sql, "WHERE")

	dbFalse := tx.Delete("logs").WhereIf(false, Cond.Eq("level", "error"))
	sql, _, _ = dbFalse.ToSQL()
	assert.NotContains(t, sql, "WHERE")
}

func TestTxDeleteBuilder_Returning(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: dollarPlaceholder}}
	db := tx.Delete("users").Where(Cond.Eq("id", "1")).Returning("id")
	sql, _, err := db.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "DELETE FROM users")
	assert.Contains(t, sql, "RETURNING id")
}

// ── TxDB error wrapping ────────────────────────────────

func TestTxDB_wrapErr_WithErrFn(t *testing.T) {
	expected := errors.New("wrapped")
	tx := &TxDB{
		tx: &sqlx.Tx{},
		ec: execContext{
			errfn: func(op string, err error, query string, args []any) error { return expected },
		},
	}
	err := tx.ec.wrapErr("op", assert.AnError, "", nil)
	assert.Equal(t, expected, err)
}

func TestTxDB_wrapErr_NoErrFn(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{}}
	err := tx.ec.wrapErr("op", assert.AnError, "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "op")
}

// ── Immutability ───────────────────────────────────────

func TestTxDB_Immutability(t *testing.T) {
	tx := &TxDB{tx: &sqlx.Tx{}, ec: execContext{pf: questionPlaceholder}}
	base := tx.Select("*").From("users")

	_ = base.Where(Cond.Eq("a", 1))
	_ = base.OrderBy("a", ASC)
	_ = base.Limit(5)
	_ = base.Suffix("FOR UPDATE")

	sql, _, _ := base.ToSQL()
	assert.NotContains(t, sql, "WHERE")
	assert.NotContains(t, sql, "ORDER BY")
	assert.NotContains(t, sql, "LIMIT")
	assert.NotContains(t, sql, "FOR UPDATE")
}
