package assertdb

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

// Query creates a new query on which one can assert things
func Query(t *testing.T, db *sqlx.DB, sql string, args ...any) *TestQuery {
	return &TestQuery{t, db, sql, args}
}

// TestQuery is a query that we can assert the result of
type TestQuery struct {
	t    *testing.T
	db   *sqlx.DB
	sql  string
	args []any
}

// Returns asserts that the query returns a single value
func (q *TestQuery) Returns(expected any, msgAndArgs ...any) bool {
	q.t.Helper()

	// get a variable of same type to hold actual result
	actual := expected

	err := q.db.Get(&actual, q.sql, q.args...)
	assert.NoError(q.t, err, msgAndArgs...)

	// not sure why but if you pass an int you get back an int64..
	switch expected.(type) {
	case int:
		actual = int(actual.(int64))
	}

	return assert.Equal(q.t, expected, actual, msgAndArgs...)
}

// Columns asserts that the query returns the given column values
func (q *TestQuery) Columns(expected map[string]any, msgAndArgs ...any) bool {
	q.t.Helper()

	actual := make(map[string]any, len(expected))

	err := q.db.QueryRowx(q.sql, q.args...).MapScan(actual)
	assert.NoError(q.t, err, msgAndArgs...)
	return assert.Equal(q.t, expected, actual, msgAndArgs...)
}
