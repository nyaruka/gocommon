package assertdb

import (
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

// Query creates a new query on which one can assert things
func Query(t assert.TestingT, db *sqlx.DB, sql string, args ...any) *TestQuery {
	return &TestQuery{t, db, sql, args}
}

// TestQuery is a query that we can assert the result of
type TestQuery struct {
	t    assert.TestingT
	db   *sqlx.DB
	sql  string
	args []any
}

// Returns asserts that the query returns a single value
func (q *TestQuery) Returns(expected any, msgAndArgs ...any) bool {
	q.helper()

	return assertReturns(q.t, q.db, q.sql, q.args, expected, msgAndArgs...)
}

// Columns asserts that the query returns the given column values
func (q *TestQuery) Columns(expected map[string]any, msgAndArgs ...any) bool {
	q.helper()

	return assertColumns(q.t, q.db, q.sql, q.args, expected, msgAndArgs...)
}

// Map scans two column rows into a map and asserts that it matches the expected
func (q *TestQuery) Map(expected map[string]any, msgAndArgs ...any) bool {
	q.helper()

	return assertMap(q.t, q.db, q.sql, q.args, expected, msgAndArgs...)
}

// List scans single column rows into a list and asserts that it matches the expected
func (q *TestQuery) List(expected []any, msgAndArgs ...any) bool {
	q.helper()

	return assertList(q.t, q.db, q.sql, q.args, expected, msgAndArgs...)
}

// Set scans single column rows into a unordered set and asserts that it matches the expected
func (q *TestQuery) Set(expected []any, msgAndArgs ...any) bool {
	q.helper()

	return assertSet(q.t, q.db, q.sql, q.args, expected, msgAndArgs...)
}

func (q *TestQuery) helper() {
	if h, ok := q.t.(tHelper); ok {
		h.Helper()
	}
}
