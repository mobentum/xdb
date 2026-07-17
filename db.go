package xdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// DBConfig holds connection-pool settings passed to New.
type DBConfig struct {
	Driver          string        // "postgres" (default), "mysql", "sqlite3"
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	Logger          *slog.Logger // optional; defaults to slog.Default()
	LogQueries      bool         // log all generated queries at Debug level
}

// queryExecutor is satisfied by both *sqlx.DB and *sqlx.Tx.
type queryExecutor interface {
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error)
}

// execContext carries the execution dependencies shared by all builders.
// Both *sqlx.DB (pool) and *sqlx.Tx (transaction) implement queryExecutor,
// so the same builder types work for both.
type execContext struct {
	ext        queryExecutor
	pf         placeholderFormat
	driverName string // "postgres", "mysql", "sqlite3"
	logfn      func(ctx context.Context, op, query string, args []any)
	errfn      func(op string, err error, query string, args []any) error
}

func (ec execContext) log(ctx context.Context, op, query string, args []any) {
	if ec.logfn != nil {
		ec.logfn(ctx, op, query, args)
	}
}

func (ec execContext) wrapErr(op string, err error, query string, args []any) error {
	if ec.errfn != nil {
		return ec.errfn(op, err, query, args)
	}
	return fmt.Errorf("%s: %w", op, err)
}

func (ec execContext) supportsReturning() bool {
	return ec.driverName == "postgres" || ec.driverName == "pgx"
}

func (ec execContext) conflictKeyword() string {
	if ec.driverName == "mysql" || ec.driverName == "mariadb" {
		return "ON DUPLICATE KEY"
	}
	return "ON CONFLICT"
}

// DB is the unified entry point: it wraps sqlx for execution and
// exposes builder methods for constructing queries.
type DB struct {
	db         *sqlx.DB
	dialect    Dialect
	pf         placeholderFormat
	logger     *slog.Logger
	logQueries bool
	dsn        string // stored for migration support
}

func (d *DB) ec() execContext {
	return execContext{
		ext:        d.db,
		pf:         d.pf,
		driverName: d.dialect.DriverName(),
		logfn:      d.logQuery,
		errfn:      d.wrapErr,
	}
}

// New connects to the database specified by cfg and returns a ready *DB.
// The caller is expected to import the appropriate database driver
// (e.g. _ "github.com/lib/pq", _ "github.com/go-sql-driver/mysql", _ "github.com/mattn/go-sqlite3").
func New(cfg DBConfig) (*DB, error) {
	driver := cfg.Driver
	if driver == "" {
		driver = "postgres"
	}
	dialect := resolveDialect(driver)

	db, err := sqlx.Connect(driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("xdb.New: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &DB{
		db:         db,
		dialect:    dialect,
		pf:         dialect.Placeholder(),
		logger:     logger,
		logQueries: cfg.LogQueries,
		dsn:        cfg.DSN,
	}, nil
}

// Wrap creates a *DB from an existing *sqlx.DB using the PostgreSQL dialect.
func Wrap(db *sqlx.DB) *DB { return WrapWithDialect(db, postgresDialect{}) }

// WrapWithDialect creates a *DB from an existing *sqlx.DB with the given Dialect.
func WrapWithDialect(db *sqlx.DB, d Dialect) *DB {
	return &DB{
		db:      db,
		dialect: d,
		pf:      d.Placeholder(),
		logger:  slog.Default(),
	}
}

// Close closes the underlying connection pool.
func (d *DB) Close() error { return d.db.Close() }

// Ping verifies the connection is alive.
func (d *DB) Ping(ctx context.Context) error { return d.db.PingContext(ctx) }

// Underlying returns the raw *sqlx.DB for escape-hatch use (e.g. named queries).
func (d *DB) Underlying() *sqlx.DB { return d.db }

// ILike builds a case-insensitive LIKE predicate compatible with the database dialect.
func (d *DB) ILike(col, pattern string) Predicate { return d.dialect.ILike(col, pattern) }

// ── Observability helpers ─────────────────────────────────

func (d *DB) logQuery(ctx context.Context, op, query string, args []any) {
	if d.logQueries && d.logger != nil {
		d.logger.LogAttrs(ctx, slog.LevelDebug, op,
			slog.String("query", query),
			slog.Any("args", args),
		)
	}
}

func (d *DB) wrapErr(op string, err error, query string, args []any) error {
	return &QueryError{Op: op, SQL: query, Args: args, Err: err}
}

// ── Builder entry points ─────────────────────────────────────

// Select starts a SELECT builder.
func (d *DB) Select(cols ...string) SelectBuilder {
	return SelectBuilder{ec: d.ec(), data: selectData{columns: cols}}
}

// WithCTE starts a CTE builder to define Common Table Expressions.
func (d *DB) WithCTE() CTEBuilder {
	return CTEBuilder{ec: d.ec(), ctes: []CTE{}}
}

// Insert starts an INSERT builder for the given table.
func (d *DB) Insert(table string) InsertBuilder {
	return InsertBuilder{ec: d.ec(), data: insertData{table: table}}
}

// Update starts an UPDATE builder for the given table.
func (d *DB) Update(table string) UpdateBuilder {
	return UpdateBuilder{ec: d.ec(), data: updateData{table: table}}
}

// Delete starts a DELETE builder for the given table.
func (d *DB) Delete(table string) DeleteBuilder {
	return DeleteBuilder{ec: d.ec(), data: deleteData{table: table}}
}

// ── Transaction ──────────────────────────────────────────────

// Tx begins a transaction, calls fn with a *TxDB that exposes the same
// builder API, then commits or rolls back automatically.
//
// If fn returns an error the transaction is rolled back.
// If fn panics the transaction is rolled back and the panic is re-raised.
func (d *DB) Tx(ctx context.Context, fn func(*TxDB) error) error {
	tx, err := d.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	tdb := &TxDB{tx: tx, ec: d.ec()}
	if err := fn(tdb); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback (%v) after: %w", rbErr, err)
		}
		return err
	}
	return tx.Commit()
}

// ── Raw escape hatches ───────────────────────────────────────

// RawOne executes rawSQL with args and scans a single row into dest.
func (d *DB) RawOne(ctx context.Context, dest any, rawSQL string, args ...any) error {
	return rawOne(ctx, d.ec(), dest, rawSQL, args...)
}

// RawAll executes rawSQL with args and scans all rows into dest.
func (d *DB) RawAll(ctx context.Context, dest any, rawSQL string, args ...any) error {
	return rawAll(ctx, d.ec(), dest, rawSQL, args...)
}

// RawExec executes rawSQL and returns the number of rows affected.
func (d *DB) RawExec(ctx context.Context, rawSQL string, args ...any) (int64, error) {
	return rawExec(ctx, d.ec(), rawSQL, args...)
}

// ── Internal helpers ─────────────────────────────────────────

func isNoRows(err error) bool {
	return err != nil && (errors.Is(err, sql.ErrNoRows) ||
		strings.Contains(err.Error(), "no rows in result set"))
}

func rawOne(ctx context.Context, ec execContext, dest any, rawSQL string, args ...any) error {
	ec.log(ctx, "RawOne", rawSQL, args)
	if err := ec.ext.GetContext(ctx, dest, rawSQL, args...); err != nil {
		if isNoRows(err) {
			return ErrNotFound
		}
		return ec.wrapErr("RawOne", err, rawSQL, args)
	}
	return nil
}

func rawAll(ctx context.Context, ec execContext, dest any, rawSQL string, args ...any) error {
	ec.log(ctx, "RawAll", rawSQL, args)
	if err := ec.ext.SelectContext(ctx, dest, rawSQL, args...); err != nil {
		return ec.wrapErr("RawAll", err, rawSQL, args)
	}
	return nil
}

func rawExec(ctx context.Context, ec execContext, rawSQL string, args ...any) (int64, error) {
	ec.log(ctx, "RawExec", rawSQL, args)
	res, err := ec.ext.ExecContext(ctx, rawSQL, args...)
	if err != nil {
		return 0, ec.wrapErr("RawExec", err, rawSQL, args)
	}
	n, _ := res.RowsAffected()
	return n, nil
}
