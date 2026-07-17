package xdb

import (
	"context"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBConfig_Valid(t *testing.T) {
	cfg := DBConfig{DSN: "postgres://user:pass@localhost:5432/testdb?sslmode=disable"}
	assert.NotEmpty(t, cfg.DSN)
}

func TestWrap_CreatesDB(t *testing.T) {
	db := &sqlx.DB{}
	wrapped := Wrap(db)
	assert.NotNil(t, wrapped)
	assert.Equal(t, db, wrapped.db)
	assert.NotNil(t, wrapped.pf)
}

func TestDB_Close_MockDB(t *testing.T) {
	db := &sqlx.DB{}
	wrapped := Wrap(db)
	assert.NotNil(t, wrapped)
}

func TestDB_Underlying_Returns_RawDB(t *testing.T) {
	rawDB := &sqlx.DB{}
	wrapped := Wrap(rawDB)
	assert.Equal(t, rawDB, wrapped.Underlying())
}

func TestDB_Select_Returns_SelectBuilder(t *testing.T) {
	db := &sqlx.DB{}
	wrapped := Wrap(db)
	sb := wrapped.Select("id", "name")
	assert.NotEmpty(t, sb.data.columns)
	sql, _, err := sb.ToSQL()
	require.NoError(t, err)
	assert.NotEmpty(t, sql)
}

func TestDB_Insert_Returns_InsertBuilder(t *testing.T) {
	db := &sqlx.DB{}
	wrapped := Wrap(db)
	ib := wrapped.Insert("users")
	assert.Equal(t, "users", ib.data.table)
}

func TestDB_Update_Returns_UpdateBuilder(t *testing.T) {
	db := &sqlx.DB{}
	wrapped := Wrap(db)
	ub := wrapped.Update("users")
	assert.Equal(t, "users", ub.data.table)
}

func TestDB_Delete_Returns_DeleteBuilder(t *testing.T) {
	rawDB := &sqlx.DB{}
	wrapped := Wrap(rawDB)
	del := wrapped.Delete("users")
	assert.Equal(t, "users", del.data.table)
}

func TestDB_WithCTE_Returns_CTEBuilder(t *testing.T) {
	db := &sqlx.DB{}
	wrapped := Wrap(db)
	cte := wrapped.CTE()
	assert.NotNil(t, cte.ctes)
	assert.Empty(t, cte.ctes)
}

func TestDB_BuilderChaining(t *testing.T) {
	db := &sqlx.DB{}
	wrapped := Wrap(db)
	sb := wrapped.Select("id", "name", "email").
		From("users").
		Where(Cond.Eq("active", true)).
		OrderBy("name", ASC)
	sql, args, err := sb.ToSQL()
	require.NoError(t, err)
	assert.NotEmpty(t, sql)
	assert.Len(t, args, 1)
}

func TestDB_MultipleBuilders_Independent(t *testing.T) {
	db := &sqlx.DB{}
	wrapped := Wrap(db)
	sb1 := wrapped.Select("id").From("users")
	sb2 := wrapped.Select("name").From("products")
	sql1, _, err1 := sb1.ToSQL()
	sql2, _, err2 := sb2.ToSQL()
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, sql1, sql2)
}

func TestDB_SelectBuilder_NoColumns(t *testing.T) {
	db := &sqlx.DB{}
	wrapped := Wrap(db)
	sb := wrapped.Select()
	assert.NotNil(t, sb.data)
}

func TestDB_SelectBuilder_ManyColumns(t *testing.T) {
	db := &sqlx.DB{}
	wrapped := Wrap(db)
	sb := wrapped.Select("col1", "col2", "col3", "col4", "col5")
	sql, _, err := sb.From("table").ToSQL()
	require.NoError(t, err)
	assert.NotEmpty(t, sql)
}

func TestDB_SelectBuilder_WithExpressions(t *testing.T) {
	db := &sqlx.DB{}
	wrapped := Wrap(db)
	sb := wrapped.Select("COUNT(*) AS count", "MAX(created_at) AS latest").From("events")
	sql, _, err := sb.ToSQL()
	require.NoError(t, err)
	assert.NotEmpty(t, sql)
}

func TestIsNoRows_WithNilError(t *testing.T) {
	result := isNoRows(nil)
	assert.False(t, result)
}

func TestDB_PlaceholderFormat(t *testing.T) {
	db := &sqlx.DB{}
	wrapped := Wrap(db)
	sb := wrapped.Select("*").From("users").Where(Cond.Eq("id", 1))
	sql, _, _ := sb.ToSQL()
	assert.NotEmpty(t, sql)
	assert.NotNil(t, wrapped.pf)
}

func TestDB_wrapErr_ReturnsQueryError(t *testing.T) {
	d := &DB{pf: questionPlaceholder}
	err := d.wrapErr("test.op", assert.AnError, "SELECT 1", []any{42})
	var qe *QueryError
	assert.True(t, errors.As(err, &qe))
	assert.Equal(t, "test.op", qe.Op)
	assert.Equal(t, "SELECT 1", qe.SQL)
	assert.Equal(t, []any{42}, qe.Args)
	assert.ErrorIs(t, qe, assert.AnError)
}

func TestDB_ILike(t *testing.T) {
	d := &DB{dialect: postgresDialect{}}
	p := d.ILike("name", "%john%")
	sql, args, err := p.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "ILIKE")
	assert.Equal(t, []any{"%john%"}, args)
}

func TestExecContext_wrapErr_WithErrFn(t *testing.T) {
	var called bool
	ec := execContext{
		errfn: func(op string, err error, query string, args []any) error { called = true; return err },
	}
	err := ec.wrapErr("op", assert.AnError, "", nil)
	assert.True(t, called)
	assert.Error(t, err)
}

func TestExecContext_wrapErr_NoErrFn(t *testing.T) {
	ec := execContext{}
	err := ec.wrapErr("op", assert.AnError, "", nil)
	assert.Error(t, err)
}

func TestExecContext_supportsReturning(t *testing.T) {
	assert.True(t, (execContext{driverName: "postgres"}).supportsReturning())
	assert.True(t, (execContext{driverName: "pgx"}).supportsReturning())
	assert.False(t, (execContext{driverName: "mysql"}).supportsReturning())
	assert.False(t, (execContext{driverName: "sqlite3"}).supportsReturning())
	assert.False(t, (execContext{driverName: ""}).supportsReturning())
}

func TestExecContext_log(t *testing.T) {
	var called bool
	ec := execContext{
		logfn: func(_ context.Context, op, query string, args []any) {
			called = true
		},
	}
	ec.log(context.Background(), "op", "SELECT 1", nil)
	assert.True(t, called)
}

func TestExecContext_log_NilLogFn(t *testing.T) {
	ec := execContext{}
	ec.log(context.Background(), "op", "SELECT 1", nil) // should not panic
}

func TestExecContext_conflictKeyword(t *testing.T) {
	assert.Equal(t, "ON CONFLICT", (execContext{driverName: "postgres"}).conflictKeyword())
	assert.Equal(t, "ON DUPLICATE KEY", (execContext{driverName: "mysql"}).conflictKeyword())
	assert.Equal(t, "ON DUPLICATE KEY", (execContext{driverName: "mariadb"}).conflictKeyword())
	assert.Equal(t, "ON CONFLICT", (execContext{driverName: "sqlite3"}).conflictKeyword())
}
