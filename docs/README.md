# xdb

**xdb** is a fluent, immutable SQL query builder for Go. It wraps `sqlx` for execution and struct scanning, with zero reflection in SQL construction and first-class support for PostgreSQL, MySQL, and SQLite.

## Quick start

```go
import "github.com/mobentum/xdb"

db, err := xdb.New(xdb.DBConfig{
    DSN: "postgres://user:pass@localhost:5432/mydb?sslmode=disable",
})
defer db.Close()
```

All examples are runnable as Example functions in `example_test.go`:

```
go test -run=Example -v
```

## Features

- **Fluent immutable builders** — every chain method returns a new copy; fork freely
- **Multi-dialect** — PostgreSQL (`$N`), MySQL/SQLite (`?`)
- **Observability** — configurable structured logging via `slog`, errors include SQL and args
- **Locking** — `ForUpdate().SkipLocked()`, `ForShare().NoWait()`, `Of(tables...)`
- **Pagination** — generic `Paginate[T]` helper with COUNT + data query
- **Transactions** — `DB.Tx()` with auto-rollback on error or panic
- **CTE** — `WithCTE()` / `WithSelectCTE()` for common table expressions
- **Migrations** — `MigrateUp`, `MigrateDown`, `MigrateTo`, `MigrateStep` via `golang-migrate`
- **Conditional Where** — `WhereIf(cond, pred)` for dynamic filters
- **Sort whitelist** — `AllowSort(cols...)` prevents ORDER BY injection
- **Streaming** — `Each()` avoids loading entire result sets
- **Raw SQL** — `RawOne` / `RawAll` / `RawExec` escape hatches

## Documentation

| File | Description |
|---|---|
| `docs/ARCHITECTURE.md` | Design philosophy, layers, immutability, dependencies |
| `docs/TESTS.md` | Test tiers, coverage, how to run |
| `example_test.go` | 18 runnable examples covering every API surface |

## Dependencies

- `github.com/jmoiron/sqlx` — execution and struct scanning
- `github.com/golang-migrate/migrate/v4` — migration engine (optional)

No ORM, no reflection-based cloning, no implicit database driver imports.

## Design notes

- **Same builders in transactions** — `TxDB.Select()` / `Insert()` / `Update()` / `Delete()` return the exact same builder types as `DB`. No separate `Tx*Builder` types to learn.
- **Immutability** — every chain method returns a new copy; fork freely.
- **Dialect-aware** — PostgreSQL (`$N`), MySQL/SQLite (`?`), `ON CONFLICT` vs `ON DUPLICATE KEY`.
- **Observability** — errors include SQL and args via `QueryError`; optional `slog` logging.

## Quick reference

```go
// SELECT
db.Select("id", "name").From("users").Where(Cond.Eq("id", id)).One(ctx, &user)
db.Select("*").From("users").OrderBy("name", ASC).All(ctx, &users)
db.Select("*").From("users").Count(ctx)
db.Select("1").From("users").Where(Cond.Eq("email", e)).Exists(ctx)

// INSERT
db.Insert("users").Columns("id", "name").Values(id, name).Exec(ctx)
db.Insert("users").Columns("id", "name").Values(id, name).Returning("id").One(ctx, &u)
db.Insert("users").SetMap(m).OnConflict("(id) DO UPDATE SET name = EXCLUDED.name").Exec(ctx)

// UPDATE
db.Update("users").Set("name", "Bob").Where(Cond.Eq("id", id)).Exec(ctx)
db.Update("users").Set("name", "Bob").ExecMustAffect(ctx)
db.Update("products").SetExpr("price", "price * ?", 1.10).Exec(ctx)

// DELETE
db.Delete("users").Where(Cond.Eq("id", id)).Exec(ctx)
db.Delete("sessions").Where(Cond.Lt("expires_at", t)).Returning("id").Exec(ctx)

// Transactions
db.Tx(ctx, func(tx *xdb.TxDB) error {
    tx.Update("accounts").SetExpr("balance", "balance - ?", 100).Where(Cond.Eq("id", a)).ExecMustAffect(ctx)
    tx.Update("accounts").SetExpr("balance", "balance + ?", 100).Where(Cond.Eq("id", b)).ExecMustAffect(ctx)
    return nil
})

// Pagination
result, _ := xdb.Paginate[Post](ctx, db.Select("id", "title").From("posts"), Page{Number: 1, Size: 20})

// Locking
db.Select("*").From("orders").Lock(ForUpdate().SkipLocked()).All(ctx, &orders)

// CTE
db.WithCTE().WithCTE("recent", `SELECT * FROM orders WHERE created_at > NOW() - INTERVAL '7 days'`).
    Select("id", "total").From("recent").All(ctx, &results)

// Migrations
db.MigrateUp(embedFS, "migrations")

// Conditions
Cond.Eq, Cond.NotEq, Cond.Gt, Cond.Lt, Cond.GtOrEq, Cond.LtOrEq
Cond.Like, Cond.ILike, Cond.IsNull, Cond.IsNotNull
Cond.In, Cond.Between, Cond.Search, Cond.Raw
Cond.And(preds...), Cond.Or(preds...)

// Raw SQL
db.RawOne(ctx, &count, "SELECT COUNT(*) FROM users WHERE active = $1", true)
```

## Database support

| Database | Placeholder | Driver |
|---|---|---|
| PostgreSQL | `$N` | `lib/pq` or `pgx` |
| MySQL / MariaDB | `?` | `go-sql-driver/mysql` |
| SQLite | `?` | `mattn/go-sqlite3` or `modernc.org/sqlite` |
