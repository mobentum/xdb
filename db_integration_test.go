//go:build integration

package xdb_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mobentum/xdb"
)

func newMock(t *testing.T) (*xdb.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	return xdb.Wrap(sqlx.NewDb(sqlDB, "postgres")), mock
}

func TestIntegration_Select_One(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()
	ctx := context.Background()

	mock.ExpectQuery(`SELECT id, name FROM users WHERE id = \$1`).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("u1", "Alice"))

	var u struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	err := d.Select("id", "name").From("users").Where(xdb.Cond.Eq("id", "u1")).One(ctx, &u)
	require.NoError(t, err)
	assert.Equal(t, "u1", u.ID)
	assert.Equal(t, "Alice", u.Name)
}

func TestIntegration_Select_One_NotFound(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectQuery(`SELECT id FROM users WHERE id = \$1`).
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	var u struct{ ID string `db:"id"` }
	err := d.Select("id").From("users").Where(xdb.Cond.Eq("id", "missing")).One(context.Background(), &u)
	assert.ErrorIs(t, err, xdb.ErrNotFound)
}

func TestIntegration_Select_All(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()
	ctx := context.Background()

	mock.ExpectQuery(`SELECT id, name FROM users ORDER BY name ASC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow("u2", "Bob").
			AddRow("u1", "Alice"))

	var users []struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	err := d.Select("id", "name").From("users").OrderBy("name", xdb.ASC).All(ctx, &users)
	require.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestIntegration_Select_Count(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

	count, err := d.Select("*").From("users").Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 42, count)
}

func TestIntegration_Select_Exists(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectQuery(`SELECT EXISTS \(SELECT 1 FROM users WHERE id = \$1\)`).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	exists, err := d.Select("1").From("users").Where(xdb.Cond.Eq("id", "u1")).Exists(context.Background())
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestIntegration_Select_NotExists(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectQuery(`SELECT EXISTS \(SELECT 1 FROM users WHERE id = \$1\)`).
		WithArgs("missing").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	exists, err := d.Select("1").From("users").Where(xdb.Cond.Eq("id", "missing")).Exists(context.Background())
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestIntegration_Insert_Exec(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectExec(`INSERT INTO users \(id,name\) VALUES \(\$1,\$2\)`).
		WithArgs("u1", "Alice").
		WillReturnResult(sqlmock.NewResult(1, 1))

	n, err := d.Insert("users").Columns("id", "name").Values("u1", "Alice").Exec(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
}

func TestIntegration_Insert_One(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectQuery(`INSERT INTO users \(id,name\) VALUES \(\$1,\$2\) RETURNING id, name`).
		WithArgs("u1", "Alice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("u1", "Alice"))

	var u struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	err := d.Insert("users").Columns("id", "name").Values("u1", "Alice").Returning("id", "name").One(context.Background(), &u)
	require.NoError(t, err)
	assert.Equal(t, "u1", u.ID)
	assert.Equal(t, "Alice", u.Name)
}

func TestIntegration_Update_Exec(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectExec(`UPDATE users SET name = \$1 WHERE id = \$2`).
		WithArgs("Bob", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	n, err := d.Update("users").Set("name", "Bob").Where(xdb.Cond.Eq("id", "u1")).Exec(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
}

func TestIntegration_Update_ExecMustAffect_Success(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectExec(`UPDATE users SET name = \$1 WHERE id = \$2`).
		WithArgs("Bob", "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := d.Update("users").Set("name", "Bob").Where(xdb.Cond.Eq("id", "u1")).ExecMustAffect(context.Background())
	assert.NoError(t, err)
}

func TestIntegration_Update_ExecMustAffect_NoRows(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectExec(`UPDATE users SET name = \$1 WHERE id = \$2`).
		WithArgs("Bob", "missing").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := d.Update("users").Set("name", "Bob").Where(xdb.Cond.Eq("id", "missing")).ExecMustAffect(context.Background())
	assert.ErrorIs(t, err, xdb.ErrNoRows)
}

func TestIntegration_Update_One(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectQuery(`UPDATE users SET name = \$1 WHERE id = \$2 RETURNING id, name`).
		WithArgs("Bob", "u1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("u1", "Bob"))

	var u struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	err := d.Update("users").Set("name", "Bob").Where(xdb.Cond.Eq("id", "u1")).Returning("id", "name").One(context.Background(), &u)
	require.NoError(t, err)
	assert.Equal(t, "Bob", u.Name)
}

func TestIntegration_Delete_Exec(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectExec(`DELETE FROM users WHERE id = \$1`).
		WithArgs("u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	n, err := d.Delete("users").Where(xdb.Cond.Eq("id", "u1")).Exec(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
}

func TestIntegration_Delete_ExecMustAffect_Success(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectExec(`DELETE FROM users WHERE id = \$1`).
		WithArgs("u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := d.Delete("users").Where(xdb.Cond.Eq("id", "u1")).ExecMustAffect(context.Background())
	assert.NoError(t, err)
}

func TestIntegration_Delete_ExecMustAffect_NoRows(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectExec(`DELETE FROM users WHERE id = \$1`).
		WithArgs("missing").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := d.Delete("users").Where(xdb.Cond.Eq("id", "missing")).ExecMustAffect(context.Background())
	assert.ErrorIs(t, err, xdb.ErrNoRows)
}

func TestIntegration_RawOne(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	var n int
	err := d.RawOne(context.Background(), &n, "SELECT COUNT(*) FROM users")
	require.NoError(t, err)
	assert.Equal(t, 5, n)
}

func TestIntegration_RawOne_NotFound(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectQuery(`SELECT \* FROM users WHERE id = \$1`).
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	var u struct{}
	err := d.RawOne(context.Background(), &u, "SELECT * FROM users WHERE id = $1", "missing")
	assert.ErrorIs(t, err, xdb.ErrNotFound)
}

func TestIntegration_RawAll(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectQuery(`SELECT id, name FROM users ORDER BY id`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow("u1", "Alice").
			AddRow("u2", "Bob"))

	type User struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	var users []User
	err := d.RawAll(context.Background(), &users, "SELECT id, name FROM users ORDER BY id")
	require.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestIntegration_RawExec(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectExec(`INSERT INTO users \(id, name\) VALUES \(\$1, \$2\)`).
		WithArgs("u1", "Alice").
		WillReturnResult(sqlmock.NewResult(1, 1))

	n, err := d.RawExec(context.Background(), "INSERT INTO users (id, name) VALUES ($1, $2)", "u1", "Alice")
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
}

func TestIntegration_Paginate(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(25))

	mock.ExpectQuery(`SELECT id, name FROM users ORDER BY name ASC LIMIT 10 OFFSET 0`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow("u1", "Alice").
			AddRow("u2", "Bob"))

	result, err := xdb.Paginate[struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}](context.Background(),
		d.Select("id", "name").From("users").OrderBy("name", xdb.ASC),
		xdb.Page{Number: 1, Size: 10},
	)
	require.NoError(t, err)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, 25, result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 10, result.PageSize)
	assert.Equal(t, 3, result.TotalPages)
}

func TestIntegration_PaginateWithCount(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()
	ctx := context.Background()

	// PaginateWithCount adds COUNT(*) OVER() AS xdb_total to the SELECT
	mock.ExpectQuery(`SELECT COUNT\(\*\) OVER\(\) AS xdb_total, id, name FROM users ORDER BY name ASC LIMIT 10 OFFSET 0`).
		WillReturnRows(sqlmock.NewRows([]string{"xdb_total", "id", "name"}).
			AddRow(25, "u1", "Alice").
			AddRow(25, "u2", "Bob"))

	type User struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	result, err := xdb.PaginateWithCount[User](ctx,
		d.Select("id", "name").From("users").OrderBy("name", xdb.ASC),
		xdb.Page{Number: 1, Size: 10},
	)
	require.NoError(t, err)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, 25, result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 10, result.PageSize)
	assert.Equal(t, 3, result.TotalPages)
}

func TestIntegration_Each(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectQuery(`SELECT id, name FROM users ORDER BY id ASC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow("u1", "Alice").
			AddRow("u2", "Bob"))

	var ids []string
	err := d.Select("id", "name").From("users").OrderBy("id", xdb.ASC).
		Each(context.Background(), func(r *sqlx.Rows) error {
			var u struct {
				ID   string `db:"id"`
				Name string `db:"name"`
			}
			if err := r.StructScan(&u); err != nil {
				return err
			}
			ids = append(ids, u.ID)
			return nil
		})
	require.NoError(t, err)
	assert.Equal(t, []string{"u1", "u2"}, ids)
}

func TestIntegration_Tx_Commit(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE accounts SET balance = balance - \$1 WHERE id = \$2`).
		WithArgs(50, "a1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE accounts SET balance = balance \+ \$1 WHERE id = \$2`).
		WithArgs(50, "a2").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := d.Tx(ctx, func(tx *xdb.TxDB) error {
		_, err := tx.Update("accounts").
			SetExpr("balance", "balance - ?", 50).
			Where(xdb.Cond.Eq("id", "a1")).
			Exec(ctx)
		require.NoError(t, err)

		_, err = tx.Update("accounts").
			SetExpr("balance", "balance + ?", 50).
			Where(xdb.Cond.Eq("id", "a2")).
			Exec(ctx)
		require.NoError(t, err)
		return nil
	})
	require.NoError(t, err)
}

func TestIntegration_Tx_Rollback(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE items SET qty = \$1 WHERE id = \$2`).
		WithArgs(20, "i1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectRollback()

	err := d.Tx(context.Background(), func(tx *xdb.TxDB) error {
		_, err := tx.Update("items").
			Set("qty", 20).
			Where(xdb.Cond.Eq("id", "i1")).
			Exec(context.Background())
		require.NoError(t, err)
		return errors.New("force rollback")
	})
	assert.Error(t, err)
}

func TestIntegration_Tx_RollbackOnPanic(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE panics SET val = \$1 WHERE id = \$2`).
		WithArgs(99, "p1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectRollback()

	func() {
		defer func() { recover() }()
		_ = d.Tx(context.Background(), func(tx *xdb.TxDB) error {
			_, _ = tx.Update("panics").
				Set("val", 99).
				Where(xdb.Cond.Eq("id", "p1")).
				Exec(context.Background())
			panic("forced panic")
		})
	}()

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIntegration_TxDB_Select_One(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, name FROM txdata WHERE id = \$1`).
		WithArgs("t1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("t1", "hello"))
	mock.ExpectCommit()

	err := d.Tx(context.Background(), func(tx *xdb.TxDB) error {
		var found struct {
			ID   string `db:"id"`
			Name string `db:"name"`
		}
		err := tx.Select("id", "name").
			From("txdata").
			Where(xdb.Cond.Eq("id", "t1")).
			One(context.Background(), &found)
		require.NoError(t, err)
		assert.Equal(t, "hello", found.Name)
		return nil
	})
	require.NoError(t, err)
}

func TestIntegration_TxDB_Insert_Exec(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO txdata \(id\) VALUES \(\$1\)`).
		WithArgs("t1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := d.Tx(context.Background(), func(tx *xdb.TxDB) error {
		n, err := tx.Insert("txdata").Columns("id").Values("t1").Exec(context.Background())
		require.NoError(t, err)
		assert.Equal(t, int64(1), n)
		return nil
	})
	require.NoError(t, err)
}

func TestIntegration_TxDB_Delete_Exec(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM txdata WHERE id = \$1`).
		WithArgs("t1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := d.Tx(context.Background(), func(tx *xdb.TxDB) error {
		n, err := tx.Delete("txdata").Where(xdb.Cond.Eq("id", "t1")).Exec(context.Background())
		require.NoError(t, err)
		assert.Equal(t, int64(1), n)
		return nil
	})
	require.NoError(t, err)
}

func TestIntegration_TxDB_Exists(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT EXISTS \(SELECT 1 FROM txdata WHERE id = \$1\)`).
		WithArgs("t1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectCommit()

	err := d.Tx(context.Background(), func(tx *xdb.TxDB) error {
		exists, err := tx.Select("1").From("txdata").Where(xdb.Cond.Eq("id", "t1")).Exists(context.Background())
		require.NoError(t, err)
		assert.True(t, exists)
		return nil
	})
	require.NoError(t, err)
}

func TestIntegration_TxDB_RawOne(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM txdata`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectCommit()

	err := d.Tx(context.Background(), func(tx *xdb.TxDB) error {
		var n int
		err := tx.RawOne(context.Background(), &n, "SELECT COUNT(*) FROM txdata")
		require.NoError(t, err)
		assert.Equal(t, 3, n)
		return nil
	})
	require.NoError(t, err)
}

func TestIntegration_TxDB_RawAll(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id FROM txdata ORDER BY id`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("a").AddRow("b"))
	mock.ExpectCommit()

	err := d.Tx(context.Background(), func(tx *xdb.TxDB) error {
		var all []struct{ ID string `db:"id"` }
		err := tx.RawAll(context.Background(), &all, "SELECT id FROM txdata ORDER BY id")
		require.NoError(t, err)
		assert.Len(t, all, 2)
		return nil
	})
	require.NoError(t, err)
}

func TestIntegration_TxDB_RawExec(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO txdata \(id\) VALUES \(\$1\)`).
		WithArgs("x1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := d.Tx(context.Background(), func(tx *xdb.TxDB) error {
		n, err := tx.RawExec(context.Background(), "INSERT INTO txdata (id) VALUES ($1)", "x1")
		require.NoError(t, err)
		assert.Equal(t, int64(1), n)
		return nil
	})
	require.NoError(t, err)
}

func TestIntegration_CTE(t *testing.T) {
	d, mock := newMock(t)
	defer d.Close()

	mock.ExpectQuery(`WITH regional AS \(SELECT region, SUM\(amount\) AS total FROM cte_orders GROUP BY region\) SELECT region, total FROM regional ORDER BY total DESC`).
		WillReturnRows(sqlmock.NewRows([]string{"region", "total"}).
			AddRow("US", 300).
			AddRow("EU", 300))

	type Result struct {
		Region string `db:"region"`
		Total  int    `db:"total"`
	}
	var results []Result
	err := d.CTE().
		WithCTE("regional", "SELECT region, SUM(amount) AS total FROM cte_orders GROUP BY region").
		Select("region", "total").
		From("regional").
		OrderBy("total", xdb.DESC).
		All(context.Background(), &results)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}
