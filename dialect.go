package xdb

import "strings"

// Dialect encapsulates SQL dialect differences between supported databases.
type Dialect interface {
	// Placeholder returns the placeholder format function for this dialect.
	Placeholder() placeholderFormat
	// DriverName returns the database driver name (e.g., "postgres", "mysql", "sqlite3").
	DriverName() string
	// SupportsReturning returns true if the dialect supports RETURNING clauses.
	SupportsReturning() bool
	// SupportsOnConflict returns true if the dialect supports ON CONFLICT / ON DUPLICATE KEY.
	SupportsOnConflict() bool
	// ILike builds a case-insensitive LIKE predicate.
	ILike(col, pattern string) Predicate
	// QuoteIdent quotes an identifier (table/column name) per dialect rules.
	QuoteIdent(ident string) string
}

// ── PostgreSQL ───────────────────────────────────────────

type postgresDialect struct{}

func (postgresDialect) Placeholder() placeholderFormat   { return dollarPlaceholder }
func (postgresDialect) DriverName() string                 { return "postgres" }
func (postgresDialect) SupportsReturning() bool            { return true }
func (postgresDialect) SupportsOnConflict() bool           { return true }
func (postgresDialect) ILike(col, pattern string) Predicate {
	return iLikePred{col: col, pattern: pattern}
}
func (postgresDialect) QuoteIdent(ident string) string { return `"` + ident + `"` }

// ── MySQL / MariaDB ──────────────────────────────────────

type mysqlDialect struct{}

func (mysqlDialect) Placeholder() placeholderFormat  { return questionPlaceholder }
func (mysqlDialect) DriverName() string                { return "mysql" }
func (mysqlDialect) SupportsReturning() bool           { return false }
func (mysqlDialect) SupportsOnConflict() bool          { return true }
func (mysqlDialect) ILike(col, pattern string) Predicate {
	return likePred{col: "LOWER(" + col + ")", pattern: strings.ToLower(pattern)}
}
func (mysqlDialect) QuoteIdent(ident string) string { return "`" + ident + "`" }

// ── SQLite3 ──────────────────────────────────────────────

type sqliteDialect struct{}

func (sqliteDialect) Placeholder() placeholderFormat  { return questionPlaceholder }
func (sqliteDialect) DriverName() string                { return "sqlite3" }
func (sqliteDialect) SupportsReturning() bool           { return false }
func (sqliteDialect) SupportsOnConflict() bool          { return false }
func (sqliteDialect) ILike(col, pattern string) Predicate {
	return iLikePred{col: col, pattern: pattern}
}
func (sqliteDialect) QuoteIdent(ident string) string { return `"` + ident + `"` }

// ── Resolution ───────────────────────────────────────────

var knownDialects = map[string]Dialect{
	"postgres": postgresDialect{},
	"pgx":      postgresDialect{},
	"mysql":    mysqlDialect{},
	"mariadb":  mysqlDialect{},
	"sqlite3":  sqliteDialect{},
}

func resolveDialect(driver string) Dialect {
	if d, ok := knownDialects[driver]; ok {
		return d
	}
	return postgresDialect{} // default
}

// Dialect returns the dialect associated with this DB instance.
func (d *DB) Dialect() Dialect { return d.dialect }
