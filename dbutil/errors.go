package dbutil

import (
	"errors"
	"fmt"

	"github.com/lib/pq"
)

// IsUniqueViolation returns true if the given error is a violation of unique constraint
func IsUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code.Name() == "unique_violation"
	}
	return false
}

// QueryError is an error type for failed SQL queries
type QueryError struct {
	cause   error
	message string
	sql     string
	sqlArgs []any
}

func (e *QueryError) Error() string {
	if e.cause != nil {
		return e.message + ": " + e.cause.Error()
	}
	return e.message
}

func (e *QueryError) Unwrap() error {
	return e.cause
}

func (e *QueryError) Query() (string, []any) {
	return e.sql, e.sqlArgs
}

func QueryErrorWrapf(cause error, sql string, sqlArgs []any, message string, msgArgs ...any) error {
	if cause == nil {
		return nil
	}
	return newQueryErrorf(cause, sql, sqlArgs, message, msgArgs...)
}

func QueryErrorf(sql string, sqlArgs []any, message string, msgArgs ...any) error {
	return newQueryErrorf(nil, sql, sqlArgs, message, msgArgs...)
}

func newQueryErrorf(cause error, sql string, sqlArgs []any, message string, msgArgs ...any) error {
	return &QueryError{
		cause:   cause,
		message: fmt.Sprintf(message, msgArgs...),
		sql:     sql,
		sqlArgs: sqlArgs,
	}
}

func AsQueryError(err error) *QueryError {
	var qerr *QueryError
	errors.As(err, &qerr)
	return qerr
}
