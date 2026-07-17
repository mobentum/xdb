package xdb

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
)

// SortDir is the ORDER BY direction.
type SortDir string

const (
	ASC  SortDir = "ASC"
	DESC SortDir = "DESC"
)

// Page carries pagination parameters (1-based page number).
type Page struct {
	Number int // 1-based; defaults to 1
	Size   int // rows per page; defaults to 20
}

func (p Page) offset() int {
	if p.Number < 1 {
		p.Number = 1
	}
	return (p.Number - 1) * p.size()
}

func (p Page) size() int {
	if p.Size < 1 {
		return 20
	}
	return p.Size
}

// PageResult wraps a typed list response with pagination metadata.
type PageResult[T any] struct {
	Items      []T
	Total      int
	Page       int
	PageSize   int
	TotalPages int
}

// Paginate runs a COUNT(*) query and a data query against the same
// SelectBuilder predicate and returns a fully populated PageResult[T].
//
// T must be a struct whose fields are tagged with `db:` to match column names.
func Paginate[T any](ctx context.Context, b SelectBuilder, p Page) (*PageResult[T], error) {
	total, err := b.Count(ctx)
	if err != nil {
		return nil, err
	}

	var items []T
	if err := b.Paginate(p).All(ctx, &items); err != nil {
		return nil, err
	}

	sz := p.size()
	totalPages := 0
	if sz > 0 {
		totalPages = (total + sz - 1) / sz
	}

	return &PageResult[T]{
		Items:      items,
		Total:      total,
		Page:       p.Number,
		PageSize:   sz,
		TotalPages: totalPages,
	}, nil
}

// PaginateWithCount is like Paginate but uses COUNT(*) OVER() to get the
// total count and data rows in a single round-trip.
//
// T must be a struct. The query automatically appends
// COUNT(*) OVER() AS xdb_total to the SELECT list, and the total count
// is extracted from the result set programmatically. T does not need an
// extra field for the count.
//
// This is an optimization over Paginate which runs two separate queries
// (COUNT + data). PaginateWithCount always executes one query.
func PaginateWithCount[T any](ctx context.Context, b SelectBuilder, p Page) (*PageResult[T], error) {
	d := b.data
	cols := make([]string, len(d.columns)+1)
	cols[0] = "COUNT(*) OVER() AS xdb_total"
	copy(cols[1:], d.columns)
	d.columns = cols
	d = d.Limit(uint64(p.size())).Offset(uint64(p.offset()))

	pb := SelectBuilder{ec: b.ec, data: d}
	query, args, err := pb.ToSQL()
	if err != nil {
		return nil, fmt.Errorf("paginate build: %w", err)
	}

	b.ec.log(ctx, "paginate", query, args)
	rows, err := b.ec.ext.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, b.ec.wrapErr("paginate", err, query, args)
	}
	defer rows.Close()

	t := reflect.TypeFor[T]()
	numFields := t.NumField()

	var total int
	var items []T
	for rows.Next() {
		dest := make([]any, 0, numFields+1)

		// Scanner for the count column (first)
		var count sql.NullInt64
		dest = append(dest, &count)

		// Scanner for each struct field
		for i := range numFields {
			f := t.Field(i)
			ft := f.Type
			for ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			dest = append(dest, reflect.New(ft).Interface())
		}

		if err := rows.Scan(dest...); err != nil {
			return nil, b.ec.wrapErr("paginate.scan", err, query, args)
		}

		if count.Valid && total == 0 {
			total = int(count.Int64)
		}

		// Build T from scanned values
		var item T
		iv := reflect.ValueOf(&item).Elem()
		for i := range numFields {
			iv.Field(i).Set(reflect.ValueOf(dest[i+1]).Elem())
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, b.ec.wrapErr("paginate.rows", err, query, args)
	}

	sz := p.size()
	totalPages := 0
	if sz > 0 {
		totalPages = (total + sz - 1) / sz
	}

	return &PageResult[T]{
		Items:      items,
		Total:      total,
		Page:       p.Number,
		PageSize:   sz,
		TotalPages: totalPages,
	}, nil
}
