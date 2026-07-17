package xdb

import "strings"

// CTE represents a single Common Table Expression (WITH clause).
type CTE struct {
	name string
	sql  string
	args []any
}

// CTEBuilder accumulates CTEs before being attached to a SelectBuilder.
// It maintains immutability by returning a new CTEBuilder on each operation.
type CTEBuilder struct {
	ec   execContext
	ctes []CTE
}

// WithCTE adds a CTE with a raw SQL definition.
func (b CTEBuilder) WithCTE(name, sql string, args ...any) CTEBuilder {
	ctes := make([]CTE, len(b.ctes)+1)
	copy(ctes, b.ctes)
	ctes[len(b.ctes)] = CTE{name: name, sql: sql, args: args}
	return CTEBuilder{ec: b.ec, ctes: ctes}
}

// WithSelectCTE adds a CTE built from a SelectBuilder.
func (b CTEBuilder) WithSelectCTE(name string, sb SelectBuilder) CTEBuilder {
	sql, args, err := sb.ToSQL()
	if err != nil {
		return b
	}
	return b.WithCTE(name, sql, args...)
}

// Select begins a SELECT query with the accumulated CTEs.
func (b CTEBuilder) Select(cols ...string) SelectBuilder {
	return SelectBuilder{
		ec:   b.ec,
		ctes: b.ctes,
		data: selectData{columns: cols},
	}
}

// ─────────────────────────────────────────────────────────────

// NewCTEBuilder creates a fresh CTEBuilder to start defining CTEs.
func NewCTEBuilder() CTEBuilder {
	return CTEBuilder{}
}

// ─────────────────────────────────────────────────────────────
// SelectBuilder CTE integration
// ─────────────────────────────────────────────────────────────

// buildWithClause constructs the WITH clause prefix from accumulated CTEs.
func buildWithClause(ctes []CTE) (string, []any) {
	if len(ctes) == 0 {
		return "", []any{}
	}

	var buf strings.Builder
	var allArgs []any

	buf.WriteString("WITH ")
	for i, cte := range ctes {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(cte.name)
		buf.WriteString(" AS (")
		buf.WriteString(cte.sql)
		buf.WriteString(")")
		allArgs = append(allArgs, cte.args...)
	}
	buf.WriteString(" ")

	return buf.String(), allArgs
}

// prependWithClause prepends the WITH clause to the main SQL query.
func prependWithClause(withClause string, withArgs []any, mainSQL string, mainArgs []any) (string, []any) {
	if withClause == "" {
		return mainSQL, mainArgs
	}

	allArgs := make([]any, 0, len(withArgs)+len(mainArgs))
	allArgs = append(allArgs, withArgs...)
	allArgs = append(allArgs, mainArgs...)

	return withClause + mainSQL, allArgs
}
