# Limitations

## Execution & scanning

- **sqlx dependency required** — struct scanning, transactions, and `Rows` iteration are delegated to `github.com/jmoiron/sqlx`. xdb cannot execute queries without it.
- **`One()` / `All()` / `Each()` need a real DB** — `ToSQL()` works without a connection, but execution methods require a live `*sqlx.DB`.

## SQL coverage

- **No full-text search** — no `tsvector`/`tsquery` helpers; use `Cond.Raw`.
- **No INSERT ... SELECT** — cannot express `INSERT INTO t SELECT ...`; use raw SQL.
- **No UPDATE FROM** — no `UPDATE t SET ... FROM other WHERE ...`; use raw SQL.
- **No RETURNING on DELETE/UPDATE for MySQL** — the `SupportsReturning` flag is `false` for MySQL; `One()` returns an error.

## Migrations

- **DSN required** — `MigrateUp`/`MigrateDown` etc. fail if the DB was created via `Wrap()` (no stored DSN).
- **Separate connection** — golang-migrate opens its own database connection; for SQLite `:memory:` this creates an isolated database invisible to xdb.
- **Blank import required** — the caller must import the golang-migrate database driver (`_ "github.com/golang-migrate/migrate/v4/database/postgres"` etc.).

## ORM features (intentionally absent)

- **No auto-migrate** — no schema creation, no `db.AutoMigrate(&User{})`.
- **No associations** — no `HasMany`, `BelongsTo`, `Preload`, or eager loading.
- **No hooks / callbacks** — no `BeforeCreate`, `AfterUpdate`, etc.
- **No soft delete** — no built-in `WHERE deleted_at IS NULL` injection.
- **No optimistic locking** — no automatic version increment or `ErrConflict`.
- **No audit columns** — no auto-population of `created_at`/`updated_at`.
- **No multi-tenant helpers** — no automatic tenant ID injection.
- **No encrypted columns** — no transparent encryption/decryption.

## Database coverage

- **No SQL Server** — no `@pN` placeholder or `OUTPUT INSERTED.*` support.
- **No CockroachDB / YugabyteDB** — should work via `pgx`/`lib/pq` but not tested.
- **MySQL RETURNING** — not supported by MySQL < 8.0.21; xdb returns an error.
- **MySQL ILIKE** — uses `LOWER(col) LIKE ?` workaround; no native `ILIKE`.

## Performance

- **Allocations per chain call** — immutability copies the entire builder struct on every method call (by design).
- **`ToSQL()` regenerates every call** — value receivers prevent caching the result on the builder. The `strings.Builder` implementation is fast (~800ns for a 5-call chain).
- **`Paginate[T]` runs 2 round-trips** — uses separate COUNT and data queries. Use `PaginateWithCount[T]` for a single-round-trip alternative with `COUNT(*) OVER()`.
- **`PaginateWithCount` uses reflection** — the single-query optimization scans result columns programmatically for struct mapping. Overhead is negligible for pagination (one call per request).

## Testing

- **`&sqlx.DB{}` in unit tests** — tests construct a bare `sqlx.DB` with nil internal state; works only because no methods are called on it.
- **Real DB tests need drivers** — SQLite requires CGO (`mattn/go-sqlite3`); PostgreSQL needs `lib/pq` or `pgx`.
- **SQLite `:memory:` + migrations** — file-based SQLite required for `MigrateUp`/`MigrateDown` tests (golang-migrate uses a separate connection).
