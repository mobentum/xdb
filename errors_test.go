package xdb

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrNotFound(t *testing.T) {
	assert.NotNil(t, ErrNotFound)
	assert.NotEmpty(t, ErrNotFound.Error())
}

func TestErrNoRows(t *testing.T) {
	assert.NotNil(t, ErrNoRows)
	assert.NotEmpty(t, ErrNoRows.Error())
}

func TestErrNotFound_Is(t *testing.T) {
	assert.True(t, errors.Is(ErrNotFound, ErrNotFound))
}

func TestErrNoRows_Is(t *testing.T) {
	assert.True(t, errors.Is(ErrNoRows, ErrNoRows))
}

func TestErrNotFound_NotEqual_ErrNoRows(t *testing.T) {
	assert.False(t, errors.Is(ErrNotFound, ErrNoRows))
}

func TestErrNoRows_NotEqual_ErrNotFound(t *testing.T) {
	assert.False(t, errors.Is(ErrNoRows, ErrNotFound))
}

func TestErrNotFound_CustomError_Check(t *testing.T) {
	err := ErrNotFound
	assert.True(t, errors.Is(err, ErrNotFound))
	assert.False(t, errors.Is(err, ErrNoRows))
}

func TestErrNoRows_CustomError_Check(t *testing.T) {
	err := ErrNoRows
	assert.True(t, errors.Is(err, ErrNoRows))
	assert.False(t, errors.Is(err, ErrNotFound))
}

func TestErrors_Distinct_Sentinel_Values(t *testing.T) {
	assert.NotEqual(t, ErrNotFound, ErrNoRows)
	assert.NotEqual(t, &ErrNotFound, &ErrNoRows)
}

func TestErrNotFound_ErrorMessage_Content(t *testing.T) {
	assert.Equal(t, "not found", ErrNotFound.Error())
}

func TestErrNoRows_ErrorMessage_Content(t *testing.T) {
	assert.Equal(t, "no rows affected", ErrNoRows.Error())
}

func TestErrors_Usage_Pattern(t *testing.T) {
	testError := func(err error) string {
		if errors.Is(err, ErrNotFound) {
			return "not found"
		} else if errors.Is(err, ErrNoRows) {
			return "no rows affected"
		}
		return "other error"
	}

	assert.Equal(t, "not found", testError(ErrNotFound))
	assert.Equal(t, "no rows affected", testError(ErrNoRows))
}

func TestErrors_AsNilInterface(t *testing.T) {
	assert.NotNil(t, ErrNotFound)
	assert.NotNil(t, ErrNoRows)
}

func TestQueryError_Error_ContainsQueryAndArgs(t *testing.T) {
	err := &QueryError{
		Op:   "select.One",
		SQL:  "SELECT * FROM users WHERE id = $1",
		Args: []any{42},
		Err:  assert.AnError,
	}
	msg := err.Error()
	assert.Contains(t, msg, "select.One")
	assert.Contains(t, msg, "SELECT * FROM users WHERE id = $1")
	assert.Contains(t, msg, "42")
}

func TestQueryError_Unwrap(t *testing.T) {
	inner := assert.AnError
	err := &QueryError{Err: inner}
	assert.ErrorIs(t, err, inner)
}

func TestQueryError_Unwrap_Nil(t *testing.T) {
	err := &QueryError{}
	assert.NoError(t, err.Unwrap())
}

func TestQueryError_Is_WrapsInner(t *testing.T) {
	err := &QueryError{Err: ErrNotFound}
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestQueryError_IsNot_WrapsDifferentError(t *testing.T) {
	err := &QueryError{Err: ErrNotFound}
	assert.False(t, errors.Is(err, ErrNoRows))
}

func TestQueryError_As(t *testing.T) {
	inner := assert.AnError
	err := &QueryError{Err: inner}
	var target *QueryError
	assert.True(t, errors.As(err, &target))
	assert.Equal(t, err, target)
}

func TestQueryError_EmptyOp(t *testing.T) {
	err := &QueryError{SQL: "SELECT 1", Err: assert.AnError}
	msg := err.Error()
	assert.Contains(t, msg, "SELECT 1")
	assert.NotContains(t, msg, "<nil>")
}
