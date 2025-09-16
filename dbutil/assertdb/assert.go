package assertdb

import (
	"reflect"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/nyaruka/gocommon/dbutil"
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

	err := q.db.GetContext(q.t.Context(), &actual, q.sql, q.args...)
	assert.NoError(q.t, err, msgAndArgs...)

	return assert.Equal(q.t, simplifyValue(expected), actual, msgAndArgs...)
}

// Columns asserts that the query returns the given column values
func (q *TestQuery) Columns(expected map[string]any, msgAndArgs ...any) bool {
	q.t.Helper()

	actual := make(map[string]any, len(expected))

	err := q.db.QueryRowxContext(q.t.Context(), q.sql, q.args...).MapScan(actual)
	assert.NoError(q.t, err, msgAndArgs...)

	return assert.Equal(q.t, simplifyMap(expected), actual, msgAndArgs...)
}

// Map scans two column rows into a map and asserts that it matches the expected
func (q *TestQuery) Map(expected map[string]any, msgAndArgs ...any) bool {
	q.t.Helper()

	rows, err := q.db.QueryContext(q.t.Context(), q.sql, q.args...)
	assert.NoError(q.t, err, msgAndArgs...)

	actual := make(map[string]any, len(expected))
	err = dbutil.ScanAllMap(rows, actual)
	assert.NoError(q.t, err, msgAndArgs...)

	return assert.Equal(q.t, simplifyMap(expected), actual, msgAndArgs...)
}

func simplifyMap(m map[string]any) map[string]any {
	simplified := make(map[string]any, len(m))
	for k, v := range m {
		simplified[k] = simplifyValue(v)
	}
	return simplified
}

func simplifyValue(v any) any {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint()
	case reflect.Float32, reflect.Float64:
		return rv.Float()
	case reflect.Bool:
		return rv.Bool()
	case reflect.String:
		return rv.String()
	default:
		return v
	}
}
