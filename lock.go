package xdb

import "fmt"

// ── Lock strengths ──────────────────────────────────────

const (
	LockUpdate      = "UPDATE"
	LockNoKeyUpdate = "NO KEY UPDATE"
	LockShare       = "SHARE"
	LockKeyShare    = "KEY SHARE"
)

// ── Lock wait options ───────────────────────────────────

const (
	LockNoWait     = "NOWAIT"
	LockSkipLocked = "SKIP LOCKED"
)

// LockBuilder builds a FOR UPDATE / FOR SHARE clause.
// Create one with ForUpdate(), ForNoKeyUpdate(), ForShare(), or forKeyShare(),
// then optionally chain .SkipLocked(), .NoWait(), or .Of(tables...).
//
// Usage:
//
//	db.Select("*").From("orders").Lock(ForUpdate().SkipLocked()).All(ctx, &results)
type LockBuilder struct {
	strength string
	wait     string
	tables   []string
}

func ForUpdate() LockBuilder     { return LockBuilder{strength: LockUpdate} }
func ForNoKeyUpdate() LockBuilder { return LockBuilder{strength: LockNoKeyUpdate} }
func ForShare() LockBuilder       { return LockBuilder{strength: LockShare} }
func forKeyShare() LockBuilder    { return LockBuilder{strength: LockKeyShare} }

func (lb LockBuilder) SkipLocked() LockBuilder { lb.wait = LockSkipLocked; return lb }
func (lb LockBuilder) NoWait() LockBuilder     { lb.wait = LockNoWait; return lb }

// Of restricts the row lock to specific tables.
func (lb LockBuilder) Of(tables ...string) LockBuilder {
	lb.tables = tables
	return lb
}

func (lb LockBuilder) String() string {
	s := "FOR " + lb.strength
	if len(lb.tables) > 0 {
		s += " OF " + formatIdentList(lb.tables)
	}
	if lb.wait != "" {
		s += " " + lb.wait
	}
	return s
}

func formatIdentList(idents []string) string {
	if len(idents) == 0 {
		return ""
	}
	s := idents[0]
	for _, id := range idents[1:] {
		s += ", " + id
	}
	return s
}

// Lock applies a locking clause (FOR UPDATE / FOR SHARE etc.) to the query.
//
//	results := []Order{}
//	err := db.Select("*").From("orders").Lock(ForUpdate().SkipLocked()).All(ctx, &results)
func (b SelectBuilder) Lock(lb LockBuilder) SelectBuilder {
	return b.Suffix(lb.String())
}

// Ensure LockBuilder is usable in fmt.Stringer contexts.
var _ fmt.Stringer = LockBuilder{}
