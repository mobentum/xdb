package xdb

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// TxDB mirrors *DB but all operations run inside the same transaction.
// Obtained exclusively via DB.Tx().
type TxDB struct {
	tx *sqlx.Tx
	ec execContext
}

// Select starts a SELECT builder on the transaction.
func (t *TxDB) Select(cols ...string) SelectBuilder {
	ec := t.ec
	ec.ext = t.tx
	return SelectBuilder{ec: ec, data: selectData{columns: cols}}
}

// Insert starts an INSERT builder on the transaction.
func (t *TxDB) Insert(table string) InsertBuilder {
	ec := t.ec
	ec.ext = t.tx
	return InsertBuilder{ec: ec, data: insertData{table: table}}
}

// Update starts an UPDATE builder on the transaction.
func (t *TxDB) Update(table string) UpdateBuilder {
	ec := t.ec
	ec.ext = t.tx
	return UpdateBuilder{ec: ec, data: updateData{table: table}}
}

// Delete starts a DELETE builder on the transaction.
func (t *TxDB) Delete(table string) DeleteBuilder {
	ec := t.ec
	ec.ext = t.tx
	return DeleteBuilder{ec: ec, data: deleteData{table: table}}
}

// RawOne executes rawSQL on the transaction and scans a single row into dest.
func (t *TxDB) RawOne(ctx context.Context, dest any, rawSQL string, args ...any) error {
	ec := t.ec
	ec.ext = t.tx
	return rawOne(ctx, ec, dest, rawSQL, args...)
}

// RawAll executes rawSQL on the transaction and scans all rows into dest.
func (t *TxDB) RawAll(ctx context.Context, dest any, rawSQL string, args ...any) error {
	ec := t.ec
	ec.ext = t.tx
	return rawAll(ctx, ec, dest, rawSQL, args...)
}

// RawExec executes rawSQL on the transaction and returns rows affected.
func (t *TxDB) RawExec(ctx context.Context, rawSQL string, args ...any) (int64, error) {
	ec := t.ec
	ec.ext = t.tx
	return rawExec(ctx, ec, rawSQL, args...)
}
