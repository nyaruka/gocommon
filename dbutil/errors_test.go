package dbutil_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lib/pq"
	"github.com/nyaruka/gocommon/dbutil"
	"github.com/stretchr/testify/assert"
)

func TestIsUniqueViolation(t *testing.T) {
	var err error = &pq.Error{Code: pq.ErrorCode("23505")}

	assert.True(t, dbutil.IsUniqueViolation(err))
	assert.True(t, dbutil.IsUniqueViolation(fmt.Errorf("wrapped: %w", err)))
	assert.False(t, dbutil.IsUniqueViolation(errors.New("boom")))
}

func TestQueryError(t *testing.T) {
	qerr := dbutil.QueryErrorf("SELECT * FROM foo WHERE id = $1", []any{234}, "error selecting foo %d", 234)
	assert.Error(t, qerr)
	assert.Equal(t, `error selecting foo 234`, qerr.Error())
	assert.Equal(t, `error selecting foo 234`, fmt.Sprintf("%s", qerr))

	// can also wrap an existing error
	var err error = &pq.Error{Code: pq.ErrorCode("22025"), Message: "unsupported Unicode escape sequence"}

	qerr = dbutil.QueryErrorWrapf(err, "SELECT * FROM foo WHERE id = $1", []any{234}, "error selecting foo %d", 234)
	assert.Error(t, qerr)
	assert.Equal(t, `error selecting foo 234: pq: unsupported Unicode escape sequence`, qerr.Error())
	assert.Equal(t, `error selecting foo 234: pq: unsupported Unicode escape sequence`, fmt.Sprintf("%s", qerr))

	// can unwrap to the original error
	var pqerr *pq.Error
	assert.True(t, errors.As(qerr, &pqerr))
	assert.Equal(t, err, pqerr)

	// can unwrap a wrapped error to find the first query error
	wrapped := fmt.Errorf("error doing that: %w", fmt.Errorf("error doing this: %w", qerr))
	unwrapped := dbutil.AsQueryError(wrapped)
	assert.Equal(t, qerr, unwrapped)

	// nil if error was never a query error
	wrapped = fmt.Errorf("error doing that: %w", errors.New("error doing this"))
	assert.Nil(t, dbutil.AsQueryError(wrapped))

	query, params := unwrapped.Query()
	assert.Equal(t, "SELECT * FROM foo WHERE id = $1", query)
	assert.Equal(t, []any{234}, params)

	// wrapping a nil error returns nil
	assert.Nil(t, dbutil.QueryErrorWrapf(nil, "SELECT", nil, "ooh"))
}
