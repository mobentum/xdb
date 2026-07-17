// Package xdb provides a SQL query builder.
//
// Cond is the global condition builder for constructing WHERE predicates.
//
// Usage:
//
//	db.Select("*").From("users").Where(Cond.Eq("id", 1)).All(ctx, &users)
package xdb
