package xdb

import (
	"fmt"
	"strings"
)

// Predicate is a SQL condition fragment that can render itself
// with ? placeholders and bound arguments.
type Predicate interface {
	ToSQL() (string, []any, error)
}

// placeholderFormat replaces ? placeholders with the dialect-specific form.
type placeholderFormat func(string) string

func dollarPlaceholder(sql string) string {
	var buf strings.Builder
	n := 0
	for i := 0; i < len(sql); i++ {
		if sql[i] == '?' {
			n++
			buf.WriteString(fmt.Sprintf("$%d", n))
		} else {
			buf.WriteByte(sql[i])
		}
	}
	return buf.String()
}

func questionPlaceholder(sql string) string { return sql }

// ── Raw expression ────────────────────────────────────────

type exprPred struct {
	sql  string
	args []any
}

func (e exprPred) ToSQL() (string, []any, error) { return e.sql, e.args, nil }

// ── Comparison predicates ─────────────────────────────────

type eqPred struct {
	col string
	val any
}

func (p eqPred) ToSQL() (string, []any, error) {
	if p.val == nil {
		return p.col + " IS NULL", nil, nil
	}
	return p.col + " = ?", []any{p.val}, nil
}

type notEqPred struct {
	col string
	val any
}

func (p notEqPred) ToSQL() (string, []any, error) {
	if p.val == nil {
		return p.col + " IS NOT NULL", nil, nil
	}
	return p.col + " != ?", []any{p.val}, nil
}

type gtPred struct {
	col string
	val any
}

func (p gtPred) ToSQL() (string, []any, error) {
	return p.col + " > ?", []any{p.val}, nil
}

type ltPred struct {
	col string
	val any
}

func (p ltPred) ToSQL() (string, []any, error) {
	return p.col + " < ?", []any{p.val}, nil
}

type gtOrEqPred struct {
	col string
	val any
}

func (p gtOrEqPred) ToSQL() (string, []any, error) {
	return p.col + " >= ?", []any{p.val}, nil
}

type ltOrEqPred struct {
	col string
	val any
}

func (p ltOrEqPred) ToSQL() (string, []any, error) {
	return p.col + " <= ?", []any{p.val}, nil
}

type likePred struct {
	col     string
	pattern string
}

func (p likePred) ToSQL() (string, []any, error) {
	return p.col + " LIKE ?", []any{p.pattern}, nil
}

type iLikePred struct {
	col     string
	pattern string
}

func (p iLikePred) ToSQL() (string, []any, error) {
	return p.col + " ILIKE ?", []any{p.pattern}, nil
}

// ── NULL checks ───────────────────────────────────────────

type isNullPred struct{ col string }

func (p isNullPred) ToSQL() (string, []any, error) {
	return p.col + " IS NULL", nil, nil
}

type isNotNullPred struct{ col string }

func (p isNotNullPred) ToSQL() (string, []any, error) {
	return p.col + " IS NOT NULL", nil, nil
}

// ── IN ────────────────────────────────────────────────────

type inPred struct {
	col  string
	vals []any
}

func (p inPred) ToSQL() (string, []any, error) {
	if len(p.vals) == 0 {
		return "1=0", nil, nil
	}
	placeholders := make([]string, len(p.vals))
	for i := range p.vals {
		placeholders[i] = "?"
	}
	return p.col + " IN (" + strings.Join(placeholders, ", ") + ")", p.vals, nil
}

// ── AND / OR ──────────────────────────────────────────────

type andPred struct{ preds []Predicate }

func (p andPred) ToSQL() (string, []any, error) {
	if len(p.preds) == 0 {
		return "1=1", nil, nil
	}
	var parts []string
	var args []any
	for _, sub := range p.preds {
		sql, subArgs, err := sub.ToSQL()
		if err != nil {
			return "", nil, err
		}
		parts = append(parts, sql)
		args = append(args, subArgs...)
	}
	if len(parts) == 1 {
		return parts[0], args, nil
	}
	return "(" + strings.Join(parts, " AND ") + ")", args, nil
}

type orPred struct{ preds []Predicate }

func (p orPred) ToSQL() (string, []any, error) {
	if len(p.preds) == 0 {
		return "1=0", nil, nil
	}
	var parts []string
	var args []any
	for _, sub := range p.preds {
		sql, subArgs, err := sub.ToSQL()
		if err != nil {
			return "", nil, err
		}
		parts = append(parts, sql)
		args = append(args, subArgs...)
	}
	if len(parts) == 1 {
		return parts[0], args, nil
	}
	return "(" + strings.Join(parts, " OR ") + ")", args, nil
}

// ── Between (AND composite) ───────────────────────────────

type betweenPred struct {
	col string
	lo  any
	hi  any
}

func (p betweenPred) ToSQL() (string, []any, error) {
	return p.col + " >= ? AND " + p.col + " <= ?", []any{p.lo, p.hi}, nil
}

// ── Search (OR-composite ILIKE) ───────────────────────────

type searchPred struct {
	term string
	cols []string
}

func (p searchPred) ToSQL() (string, []any, error) {
	if len(p.cols) == 0 {
		return "1=0", nil, nil
	}
	t := "%" + strings.ToLower(p.term) + "%"
	parts := make([]string, len(p.cols))
	for i, c := range p.cols {
		parts[i] = c + " ILIKE ?"
	}
	return "(" + strings.Join(parts, " OR ") + ")", repeated(t, len(p.cols)), nil
}

func repeated(v any, n int) []any {
	if n <= 0 {
		return nil
	}
	a := make([]any, n)
	for i := range a {
		a[i] = v
	}
	return a
}

// ── Cond builder ──────────────────────────────────────────

var Cond = condBuilder{}

type condBuilder struct{}

func (condBuilder) Eq(col string, val any) Predicate   { return eqPred{col: col, val: val} }
func (condBuilder) NotEq(col string, val any) Predicate { return notEqPred{col: col, val: val} }
func (condBuilder) Gt(col string, val any) Predicate    { return gtPred{col: col, val: val} }
func (condBuilder) Lt(col string, val any) Predicate    { return ltPred{col: col, val: val} }
func (condBuilder) GtOrEq(col string, val any) Predicate { return gtOrEqPred{col: col, val: val} }
func (condBuilder) LtOrEq(col string, val any) Predicate { return ltOrEqPred{col: col, val: val} }
func (condBuilder) Like(col, pattern string) Predicate    { return likePred{col: col, pattern: pattern} }
func (condBuilder) ILike(col, pattern string) Predicate   { return iLikePred{col: col, pattern: pattern} }
func (condBuilder) IsNull(col string) Predicate           { return isNullPred{col: col} }
func (condBuilder) IsNotNull(col string) Predicate        { return isNotNullPred{col: col} }
func (condBuilder) In(col string, vals ...any) Predicate   { return inPred{col: col, vals: vals} }
func (condBuilder) Raw(sql string, args ...any) Predicate  { return exprPred{sql: sql, args: args} }

func (condBuilder) And(preds ...Predicate) Predicate {
	return andPred{preds: preds}
}

func (condBuilder) Or(preds ...Predicate) Predicate {
	return orPred{preds: preds}
}

func (condBuilder) Search(term string, cols ...string) Predicate {
	return searchPred{term: term, cols: cols}
}

func (condBuilder) Between(col string, lo, hi any) Predicate {
	return betweenPred{col: col, lo: lo, hi: hi}
}

// ── JSONB operators ────────────────────────────────────

type jsonContainsPred struct {
	col string
	val any
}

func (p jsonContainsPred) ToSQL() (string, []any, error) {
	return p.col + " @> ?", []any{p.val}, nil
}

type jsonHasKeyPred struct {
	col string
	key string
}

func (p jsonHasKeyPred) ToSQL() (string, []any, error) {
	return "jsonb_exists(" + p.col + ", ?)", []any{p.key}, nil
}

// JsonContains generates col @> ? for JSONB containment checks.
// val should be a JSON string, e.g. `{"status": "active"}`.
func (condBuilder) JsonContains(col string, val any) Predicate {
	return jsonContainsPred{col: col, val: val}
}

// JsonHasKey generates jsonb_exists(col, ?) for JSONB key existence.
func (condBuilder) JsonHasKey(col, key string) Predicate {
	return jsonHasKeyPred{col: col, key: key}
}

// ── Array operators ─────────────────────────────────────

type arrayOverlapsPred struct {
	col  string
	vals []any
}

func (p arrayOverlapsPred) ToSQL() (string, []any, error) {
	ph := make([]string, len(p.vals))
	for i := range p.vals {
		ph[i] = "?"
	}
	return p.col + " && ARRAY[" + strings.Join(ph, ",") + "]", p.vals, nil
}

type arrayContainsPred struct {
	col  string
	vals []any
}

func (p arrayContainsPred) ToSQL() (string, []any, error) {
	ph := make([]string, len(p.vals))
	for i := range p.vals {
		ph[i] = "?"
	}
	return p.col + " @> ARRAY[" + strings.Join(ph, ",") + "]", p.vals, nil
}

// ArrayOverlaps generates col && ARRAY[?, …] to test if arrays share any elements.
func (condBuilder) ArrayOverlaps(col string, vals ...any) Predicate {
	return arrayOverlapsPred{col: col, vals: vals}
}

// ArrayContains generates col @> ARRAY[?, …] to test if an array contains all values.
func (condBuilder) ArrayContains(col string, vals ...any) Predicate {
	return arrayContainsPred{col: col, vals: vals}
}
