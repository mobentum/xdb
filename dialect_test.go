package xdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveDialect_Postgres(t *testing.T) {
	d := resolveDialect("postgres")
	assert.Equal(t, "postgres", d.DriverName())
	assert.True(t, d.SupportsReturning())
	assert.True(t, d.SupportsOnConflict())
	assert.Equal(t, `"users"`, d.QuoteIdent("users"))
}

func TestResolveDialect_Pgx(t *testing.T) {
	d := resolveDialect("pgx")
	assert.Equal(t, "postgres", d.DriverName())
}

func TestResolveDialect_MySQL(t *testing.T) {
	d := resolveDialect("mysql")
	assert.Equal(t, "mysql", d.DriverName())
	assert.False(t, d.SupportsReturning())
	assert.True(t, d.SupportsOnConflict())
	assert.Equal(t, "`users`", d.QuoteIdent("users"))
}

func TestResolveDialect_MariaDB(t *testing.T) {
	d := resolveDialect("mariadb")
	assert.Equal(t, "mysql", d.DriverName())
}

func TestResolveDialect_SQLite(t *testing.T) {
	d := resolveDialect("sqlite3")
	assert.Equal(t, "sqlite3", d.DriverName())
	assert.False(t, d.SupportsReturning())
	assert.False(t, d.SupportsOnConflict())
	assert.Equal(t, `"users"`, d.QuoteIdent("users"))
}

func TestResolveDialect_Unknown_DefaultsToPostgres(t *testing.T) {
	d := resolveDialect("unknown")
	assert.Equal(t, "postgres", d.DriverName())
}

func TestPostgresDialect_ILike(t *testing.T) {
	d := postgresDialect{}
	p := d.ILike("name", "%john%")
	sql, args, err := p.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "ILIKE")
	assert.Len(t, args, 1)
	assert.Equal(t, "%john%", args[0])
}

func TestMySQLDialect_ILike(t *testing.T) {
	d := mysqlDialect{}
	p := d.ILike("name", "%John%")
	sql, args, err := p.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "LOWER(name)")
	assert.Contains(t, sql, "LIKE")
	assert.Len(t, args, 1)
	assert.Equal(t, "%john%", args[0])
}

func TestSQLiteDialect_ILike(t *testing.T) {
	d := sqliteDialect{}
	p := d.ILike("email", "%GMAIL%")
	sql, args, err := p.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "ILIKE")
	assert.Len(t, args, 1)
	assert.Equal(t, "%GMAIL%", args[0])
}

func TestSQLiteDialect_Placeholder(t *testing.T) {
	pf := sqliteDialect{}.Placeholder()
	result := pf("WHERE id = ? AND name = ?")
	assert.Equal(t, "WHERE id = ? AND name = ?", result)
}

func TestPostgresDialect_Placeholder(t *testing.T) {
	pf := postgresDialect{}.Placeholder()
	result := pf("WHERE id = ? AND name = ?")
	assert.Equal(t, "WHERE id = $1 AND name = $2", result)
}

func TestMySQLDialect_Placeholder(t *testing.T) {
	pf := mysqlDialect{}.Placeholder()
	result := pf("WHERE id = ? AND name = ?")
	assert.Equal(t, "WHERE id = ? AND name = ?", result)
}

func TestPostgresDialect_QuoteIdent(t *testing.T) {
	assert.Equal(t, `"users"`, postgresDialect{}.QuoteIdent("users"))
	assert.Equal(t, `"user id"`, postgresDialect{}.QuoteIdent("user id"))
}

func TestMySQLDialect_QuoteIdent(t *testing.T) {
	assert.Equal(t, "`users`", mysqlDialect{}.QuoteIdent("users"))
	assert.Equal(t, "`user id`", mysqlDialect{}.QuoteIdent("user id"))
}

func TestDB_Dialect(t *testing.T) {
	db := &DB{dialect: postgresDialect{}}
	assert.Equal(t, "postgres", db.Dialect().DriverName())

	db2 := &DB{dialect: mysqlDialect{}}
	assert.Equal(t, "mysql", db2.Dialect().DriverName())
}
