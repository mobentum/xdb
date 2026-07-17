//go:build realdb

package xdb_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mobentum/xdb"

	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func testDB(tb testing.TB) *xdb.DB {
	tb.Helper()

	driver := os.Getenv("XDB_TEST_DRIVER")
	if driver == "" {
		driver = "sqlite3"
	}

	dsn := os.Getenv("XDB_TEST_DSN")
	if dsn == "" {
		switch driver {
		case "postgres":
			dsn = pgDefaultDSN()
		default:
			dsn = ":memory:"
		}
	}

	db, err := xdb.New(xdb.DBConfig{
		Driver:     driver,
		DSN:        dsn,
		LogQueries: false,
	})
	require.NoError(tb, err)
	tb.Cleanup(func() { db.Close() })

	resetTestSchema(tb, db)
	return db
}

func pgDefaultDSN() string {
	host := lookupEnv("PGHOST", "localhost")
	port := lookupEnv("PGPORT", "5432")
	user := lookupEnv("PGUSER", "xdb")
	password := lookupEnv("PGPASSWORD", "xdb")
	dbname := lookupEnv("PGDATABASE", "xdb_test")
	return "postgres://" + user + ":" + password + "@" + host + ":" + port + "/" + dbname + "?sslmode=disable"
}

func lookupEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func resetTestSchema(tb testing.TB, db *xdb.DB) {
	tb.Helper()
	ctx := context.Background()

	db.RawExec(ctx, `DROP TABLE IF EXISTS users`)
	_, err := db.RawExec(ctx, `CREATE TABLE users (
		id   TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT
	)`)
	require.NoError(tb, err)
}

// ── CRUD ────────────────────────────────────────────────

func TestRealDB_InsertAndSelectOne(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	n, err := db.Insert("users").Columns("id", "name", "email").
		Values("u1", "Alice", "alice@example.com").
		Exec(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	var user struct {
		ID    string `db:"id"`
		Name  string `db:"name"`
		Email string `db:"email"`
	}
	err = db.Select("id", "name", "email").From("users").
		Where(xdb.Cond.Eq("id", "u1")).
		One(ctx, &user)
	require.NoError(t, err)
	assert.Equal(t, "Alice", user.Name)
	assert.Equal(t, "alice@example.com", user.Email)
}

func TestRealDB_SelectOne_NotFound(t *testing.T) {
	db := testDB(t)

	var u struct{ ID string `db:"id"` }
	err := db.Select("id").From("users").
		Where(xdb.Cond.Eq("id", "missing")).
		One(context.Background(), &u)
	assert.ErrorIs(t, err, xdb.ErrNotFound)
}

func TestRealDB_SelectAll(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	db.Insert("users").Columns("id", "name").Values("u1", "Alice").Exec(ctx)
	db.Insert("users").Columns("id", "name").Values("u2", "Bob").Exec(ctx)

	var users []struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	err := db.Select("id", "name").From("users").OrderBy("name", xdb.ASC).All(ctx, &users)
	require.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "Alice", users[0].Name)
	assert.Equal(t, "Bob", users[1].Name)
}

func TestRealDB_Update(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	db.Insert("users").Columns("id", "name").Values("u1", "Alice").Exec(ctx)

	n, err := db.Update("users").Set("name", "Alice Updated").
		Where(xdb.Cond.Eq("id", "u1")).
		Exec(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	var name string
	err = db.Select("name").From("users").
		Where(xdb.Cond.Eq("id", "u1")).
		One(ctx, &name)
	require.NoError(t, err)
	assert.Equal(t, "Alice Updated", name)
}

func TestRealDB_Update_ExecMustAffect(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	err := db.Update("users").Set("name", "x").
		Where(xdb.Cond.Eq("id", "nonexistent")).
		ExecMustAffect(ctx)
	assert.ErrorIs(t, err, xdb.ErrNoRows)
}

func TestRealDB_Delete(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	db.Insert("users").Columns("id", "name").Values("u1", "Alice").Exec(ctx)

	n, err := db.Delete("users").Where(xdb.Cond.Eq("id", "u1")).Exec(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	err = db.Select("id").From("users").Where(xdb.Cond.Eq("id", "u1")).
		One(ctx, &struct{ ID string `db:"id"` }{})
	assert.ErrorIs(t, err, xdb.ErrNotFound)
}

// ── Query features ──────────────────────────────────────

func TestRealDB_Exists(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	db.Insert("users").Columns("id", "name").Values("u1", "Alice").Exec(ctx)

	exists, err := db.Select("1").From("users").Where(xdb.Cond.Eq("id", "u1")).Exists(ctx)
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = db.Select("1").From("users").Where(xdb.Cond.Eq("id", "missing")).Exists(ctx)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRealDB_Count(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	db.Insert("users").Columns("id", "name").Values("u1", "Alice").Exec(ctx)
	db.Insert("users").Columns("id", "name").Values("u2", "Bob").Exec(ctx)

	count, err := db.Select("*").From("users").Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

// ── Transactions ─────────────────────────────────────────

func TestRealDB_Transaction_Commit(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	err := db.Tx(ctx, func(tx *xdb.TxDB) error {
		_, err := tx.Insert("users").Columns("id", "name").Values("t1", "TxUser").Exec(ctx)
		require.NoError(t, err)
		return nil
	})
	require.NoError(t, err)

	var name string
	err = db.Select("name").From("users").
		Where(xdb.Cond.Eq("id", "t1")).
		One(ctx, &name)
	require.NoError(t, err)
	assert.Equal(t, "TxUser", name)
}

func TestRealDB_Transaction_Rollback(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	err := db.Tx(ctx, func(tx *xdb.TxDB) error {
		tx.Insert("users").Columns("id", "name").Values("rb1", "RollbackUser").Exec(ctx)
		return assert.AnError
	})
	assert.Error(t, err)

	count, err := db.Select("1").From("users").Where(xdb.Cond.Eq("id", "rb1")).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// ── Pagination ───────────────────────────────────────────

func TestRealDB_Paginate(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	for i := range 5 {
		id := rune('0') + rune(i)
		db.Insert("users").Columns("id", "name").Values(string([]rune{'p', id}), "User "+string([]rune{'p', id})).Exec(ctx)
	}

	result, err := xdb.Paginate[struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}](ctx,
		db.Select("id", "name").From("users").OrderBy("id", xdb.ASC),
		xdb.Page{Number: 1, Size: 3},
	)
	require.NoError(t, err)
	assert.Len(t, result.Items, 3)
	assert.Equal(t, 5, result.Total)
	assert.Equal(t, 2, result.TotalPages)
}

// ── Streaming ────────────────────────────────────────────

func TestRealDB_Each(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	db.Insert("users").Columns("id", "name").Values("u1", "Alice").Exec(ctx)
	db.Insert("users").Columns("id", "name").Values("u2", "Bob").Exec(ctx)

	var names []string
	err := db.Select("id", "name").From("users").OrderBy("id", xdb.ASC).
		Each(ctx, func(r *sqlx.Rows) error {
			var u struct {
				ID   string `db:"id"`
				Name string `db:"name"`
			}
			if err := r.StructScan(&u); err != nil {
				return err
			}
			names = append(names, u.Name)
			return nil
		})
	require.NoError(t, err)
	assert.Equal(t, []string{"Alice", "Bob"}, names)
}

// ── Raw SQL helpers ──────────────────────────────────────

func TestRealDB_RawOperations(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	db.Insert("users").Columns("id", "name").Values("r1", "Raw").Exec(ctx)

	var n int
	err := db.RawOne(ctx, &n, "SELECT COUNT(*) FROM users")
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	var all []struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	err = db.RawAll(ctx, &all, "SELECT id, name FROM users ORDER BY id")
	require.NoError(t, err)
	assert.Len(t, all, 1)

	affected, err := db.RawExec(ctx, "UPDATE users SET name = 'RawUpdated' WHERE id = 'r1'")
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)
}

// ── WHERE conditions ─────────────────────────────────────

// ── Migrations ──────────────────────────────────────────

func newSQLiteFileDB(tb testing.TB) (*xdb.DB, string) {
	tb.Helper()
	f := filepath.Join(tb.TempDir(), "xdb_test.db")
	db, err := xdb.New(xdb.DBConfig{
		Driver: "sqlite3",
		DSN:    f,
	})
	require.NoError(tb, err)
	tb.Cleanup(func() { db.Close() })
	return db, f
}

func TestRealDB_MigrateUpAndDown(t *testing.T) {
	db, _ := newSQLiteFileDB(t)
	ctx := context.Background()

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "000001_create_test.up.sql"), []byte("CREATE TABLE migration_test (id TEXT PRIMARY KEY);"), 0644)
	os.WriteFile(filepath.Join(dir, "000001_create_test.down.sql"), []byte("DROP TABLE IF EXISTS migration_test;"), 0644)

	err := db.MigrateUp(os.DirFS(dir), ".")
	require.NoError(t, err)

	var n int
	err = db.RawOne(ctx, &n, "SELECT COUNT(*) FROM migration_test")
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	err = db.MigrateDown(os.DirFS(dir), ".")
	require.NoError(t, err)

	err = db.RawOne(ctx, &n, "SELECT COUNT(*) FROM migration_test")
	assert.Error(t, err)
}

func TestRealDB_MigrateUp_Idempotent(t *testing.T) {
	db, _ := newSQLiteFileDB(t)

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "000001_create.up.sql"), []byte("CREATE TABLE mig2 (id TEXT PRIMARY KEY);"), 0644)
	os.WriteFile(filepath.Join(dir, "000001_create.down.sql"), []byte("DROP TABLE IF EXISTS mig2;"), 0644)

	err := db.MigrateUp(os.DirFS(dir), ".")
	require.NoError(t, err)

	err = db.MigrateUp(os.DirFS(dir), ".")
	require.NoError(t, err)
}

func TestRealDB_Migrate_NoDSN(t *testing.T) {
	db := xdb.Wrap(nil)
	err := db.MigrateUp(nil, "")
	assert.ErrorContains(t, err, "DSN not available")
}

func TestRealDB_WhereConditions(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	db.Insert("users").Columns("id", "name", "email").Values("w1", "Alice", "alice@x.com").Exec(ctx)
	db.Insert("users").Columns("id", "name", "email").Values("w2", "Bob", "bob@x.com").Exec(ctx)
	db.Insert("users").Columns("id", "name", "email").Values("w3", "Charlie", nil).Exec(ctx)

	t.Run("IsNull", func(t *testing.T) {
		count, err := db.Select("1").From("users").
			Where(xdb.Cond.IsNull("email")).
			Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("And", func(t *testing.T) {
		var name string
		err := db.Select("name").From("users").
			Where(xdb.Cond.And(
				xdb.Cond.Eq("email", "bob@x.com"),
				xdb.Cond.Eq("name", "Bob"),
			)).
			One(ctx, &name)
		require.NoError(t, err)
		assert.Equal(t, "Bob", name)
	})
}
