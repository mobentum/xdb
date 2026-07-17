package xdb

import (
	"errors"
	"fmt"
)

// Sentinel errors returned by the query builder execution methods.
// Callers use errors.Is() to distinguish them.
var (
	// ErrNotFound is returned when GetContext finds no matching row.
	ErrNotFound = errors.New("not found")

	// ErrNoRows is returned by ExecMustAffect when 0 rows were changed.
	ErrNoRows = errors.New("no rows affected")
)

// QueryError wraps an error with the query SQL and arguments that caused it.
// Use errors.Is() and errors.As() to inspect the wrapped error.
type QueryError struct {
	Op   string
	SQL  string
	Args []any
	Err  error
}

func (e *QueryError) Error() string {
	return fmt.Sprintf("%s: %v\nquery: %s\nargs: %v", e.Op, e.Err, e.SQL, e.Args)
}

func (e *QueryError) Unwrap() error { return e.Err }
