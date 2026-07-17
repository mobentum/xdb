# Architecture

## Overview

xdb is a fluent, immutable SQL query builder for Go. It wraps `sqlx` for execution and struct scanning, providing a builder API over raw SQL.

```
┌─────────────────────────────────────────────────────────────────┐
│  User code                                                      │
│  db.Select("id").From("users").Where(Cond.Eq("id", 1)).All(…)   │
└───────────────┬─────────────────────────────────────────────────┘
                │
┌───────────────▼─────────────────────────────────────────────────┐
│  xdb public API                                                 │
│  DB / TxDB                                                       │
│  SelectBuilder / InsertBuilder / UpdateBuilder / DeleteBuilder   │
│  Cond, Page, PageResult, LockBuilder, CTEBuilder                │
│  DB.New / DB.Wrap / DB.Tx / DB.MigrateUp / MigrateDown          │
└───────┬───────────────────────┬─────────────────────────────────┘
        │                       │
┌───────▼───────────────┐ ┌─────▼─────────────────────────────────┐
│  execContext           │ │  sqlbuilder.go                        │
│  queryExecutor         │ │  selectData / insertData             │
│  (shared by pool & tx) │ │  updateData / deleteData / Window    │
│  log / wrapErr         │ │  strings.Builder ToSQL               │
│  supportsReturning     │ │                                       │
└───────┬───────────────┘ └─────┬─────────────────────────────────┘
        │                       │
┌───────▼───────────────────────▼─────────────────────────────────┐
│  predicate.go                  │  dialect.go                     │
│  Predicate interface + 20+    │  Placeholder: $N or ?           │
│  Cond builder + JSONB/array   │  ILike, SupportsReturning,      │
│  operators + WindowBuilder    │  QuoteIdent                     │
└───────┬───────────────────────┴─────────────────────────────────┘
        │
┌───────▼─────────────────────────────────────────────────────────┐
│  Execution layer (db.go / select.go / mutate.go / tx.go)        │
│  sqlx: GetContext / SelectContext / ExecContext / BeginTxx      │
│  Observability: slog logging via execContext, QueryError         │
└─────────────────────────────────────────────────────────────────┘
```

## Layers

### 1. Public API (`db.go`, `select.go`, `mutate.go`, `tx.go`, `lock.go`, `cte.go`, `page.go`, `migrate.go`)

Entry points for users. Each builder type uses value receivers for **immutability** — every method returns a new copy, enabling safe fork-and-extend patterns.

| Type | File | Purpose |
|---|---|---|---|
| `DB` | `db.go` | Connection pool, builder constructors, Tx, raw SQL |
| `TxDB` | `tx.go` | Transaction handle; returns same builder types as `DB` |
| `SelectBuilder` | `select.go` | SELECT query builder |
| `InsertBuilder` | `mutate.go` | INSERT query builder |
| `UpdateBuilder` | `mutate.go` | UPDATE query builder |
| `DeleteBuilder` | `mutate.go` | DELETE query builder |
| `CTEBuilder` | `cte.go` | Common Table Expression builder |
| `LockBuilder` | `lock.go` | FOR UPDATE / FOR SHARE builder |
| `Page` / `PageResult` | `page.go` | Pagination types + generic helper |

### 2. Internal SQL builder (`sqlbuilder.go`)

The core types `selectData`, `insertData`, `updateData`, `deleteData` generate SQL using `strings.Builder` — no reflection, no intermediate tree structures.

```
SELECT cols FROM table
  JOIN / LEFT JOIN
  WHERE pred1 AND pred2 ...
  GROUP BY cols
  HAVING pred
  ORDER BY cols
  LIMIT n OFFSET m
  suffixes (RETURNING, FOR UPDATE, etc.)
  prefixes (WITH clauses)
```

### 3. Predicate system (`predicate.go`)

The `Predicate` interface with 16 concrete types (`eqPred`, `gtPred`, `andPred`, `orPred`, etc.) renders SQL conditions from the Cond builder. All types are plain structs — zero reflection.

### 4. Shared execution context (`db.go`)

The `execContext` struct carries all execution dependencies shared by every builder:

```go
type execContext struct {
    ext        queryExecutor    // *sqlx.DB or *sqlx.Tx
    pf         placeholderFormat
    driverName string           // "postgres", "mysql", "sqlite3"
    logfn, errfn func(...)
}
```

A single set of methods (`log`, `wrapErr`, `supportsReturning`, `conflictKeyword`) lives on `execContext` instead of being duplicated across every builder type. Both `DB` and `TxDB` create an `execContext` and pass it to the builders, with `ext` pointing to the pool (`*sqlx.DB`) or the transaction (`*sqlx.Tx`).

```go
// DB.Select uses the pool.
func (d *DB) Select(cols ...string) SelectBuilder {
    return SelectBuilder{ec: d.ec(), data: selectData{columns: cols}}
}

// TxDB.Select uses the same builder type, wired to the transaction.
func (t *TxDB) Select(cols ...string) SelectBuilder {
    ec := t.ec
    ec.ext = t.tx
    return SelectBuilder{ec: ec, data: selectData{columns: cols}}
}
```

### 6. Dialect system (`dialect.go`)

Each dialect implements `Placeholder()` (dollar vs question), `ILike` (native vs LOWER()), `SupportsReturning`, `SupportsOnConflict`, and `QuoteIdent`. Currently supports **PostgreSQL**, **MySQL/MariaDB**, and **SQLite**.

## Immutability

Every builder method uses value receivers:

```go
func (b SelectBuilder) Where(pred Predicate) SelectBuilder {
    b.data = b.data.Where(pred)  // modifies the copy
    return b
}
```

This guarantees that a base query can be forked:

```go
base := db.Select("*").From("users")
admins := base.Where(Cond.Eq("role", "admin"))   // original unchanged
active := base.Where(Cond.Eq("active", true))    // original unchanged
```

## Error handling

Execution errors are wrapped in `QueryError` which includes the SQL and args:

```go
type QueryError struct {
    Op   string
    SQL  string
    Args []any
    Err  error
}
```

Always use `errors.Is`/`errors.As` to inspect. The sentinel errors `ErrNotFound` (One finds no row) and `ErrNoRows` (ExecMustAffect hits zero rows) are the only direct comparisons.

## Dependencies

| Dependency | Why |
|---|---|
| `github.com/jmoiron/sqlx` | Struct scanning, `queryExecutor` interface shared by pool & tx |
| `github.com/golang-migrate/migrate/v4` | Migration engine (optional, only when calling Migrate*) |

## Design principles

1. **Explicit over magic** — no callbacks, hooks, auto-migrate, or implicit joins
2. **Immutable builders** — safe to fork, no shared mutable state
3. **Zero reflection** in SQL construction — all builder methods are plain struct manipulations
4. **Small core** — the library provides query building + execution; migrations, validation, metrics are extension concerns
5. **Dialect-aware** — same code works on PostgreSQL, MySQL, and SQLite
