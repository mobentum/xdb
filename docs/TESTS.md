# Tests

xdb has three test tiers, each with a different dependency profile.

## Tier 1: Unit tests

```
go test ./... -count=1
```

- **286 tests**
- No external dependencies
- Test the builder chain methods, predicate rendering, placeholder formatting, immutability, and error wrapping
- Run on every `make test`

## Tier 2: sqlmock integration tests

```
go test ./... -count=1 -tags=integration
```

or:

```
make test-integration
```

- **35 tests** in `tests/db_integration_test.go`
- Uses `github.com/DATA-DOG/go-sqlmock` to simulate a PostgreSQL driver
- Verifies actual SQL generation against expected patterns: placeholders, column order, clause ordering
- Covers: SELECT (One, All, Count, Exists, Each), INSERT (Exec, One), UPDATE (Exec, ExecMustAffect, One), DELETE (Exec, ExecMustAffect), transactions (commit, rollback, panic recovery), raw SQL helpers, pagination, CTE
- Included in `make verify` alongside unit tests

## Tier 3: Real database tests

```
make test-realdb              # defaults to SQLite :memory:
XDB_TEST_DRIVER=postgres make test-realdb   # against local PostgreSQL
```

- **17 tests** in `tests/realdb_test.go`
- Requires a real database driver:
  - SQLite: `_ "github.com/mattn/go-sqlite3"` (CGO required)
  - PostgreSQL: `_ "github.com/lib/pq"` (Docker Compose or local install)
- Build tag: `realdb` (excluded from `go test ./...`)
- Covers: CRUD round-trips, transactions (commit + rollback), pagination, streaming (Each), raw SQL helpers, WHERE conditions (IsNull, And), **migrations** (up, down, idempotent, missing DSN)

### PostgreSQL with Docker

```bash
make docker-up     # docker compose -f tests/docker-compose.yml up -d --wait
make test-realdb   # XDB_TEST_DRIVER defaults to sqlite3, set explicitly:
XDB_TEST_DRIVER=postgres make test-realdb
make docker-down
```

## Coverage

```
go test ./... -tags=integration -coverprofile=coverage.out
go tool cover -func=coverage.out
```

Current: **~87%** statement coverage across all tiers.

| Package | Files | Coverage |
|---|---|---|
| Root (xdb) | predicate.go, sqlbuilder.go, db.go, select.go, mutate.go, tx.go, lock.go, cte.go, page.go, migrate.go, errors.go, dialect.go | ~87% |
| Tests | sqlmock + realdb | N/A (test code excluded) |

## Example functions

`example_test.go` contains **18 Example functions** that are compiled and run as part of `go test`. They demonstrate every major API surface using `xdb.Wrap(&sqlx.DB{})` (no real connection needed) and verify the generated SQL and args match expected output.

These examples also appear in the `go doc` output and on `pkg.go.dev`.
