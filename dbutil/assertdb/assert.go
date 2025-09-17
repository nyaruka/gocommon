package assertdb

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/jmoiron/sqlx"
	"github.com/nyaruka/gocommon/dbutil"
	"github.com/stretchr/testify/assert"
)

func assertReturns(t assert.TestingT, db *sqlx.DB, query string, args []any, expected any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}

	// get a variable of same type to hold actual result
	actual := expected

	err := db.GetContext(context.Background(), &actual, query, args...)
	assert.NoError(t, err, msgAndArgs...)

	return assert.Equal(t, simplifyValue(expected), actual, msgAndArgs...)
}

func assertColumns(t assert.TestingT, db *sqlx.DB, query string, args []any, expected map[string]any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}

	actual := make(map[string]any, len(expected))

	err := db.QueryRowxContext(context.Background(), query, args...).MapScan(actual)
	assert.NoError(t, err, msgAndArgs...)

	return assert.Equal(t, simplifyMap(expected), actual, msgAndArgs...)
}

func assertMap(t assert.TestingT, db *sqlx.DB, query string, args []any, expected map[string]any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}

	rows, err := db.QueryContext(context.Background(), query, args...)
	assert.NoError(t, err, msgAndArgs...)

	actual := make(map[string]any, len(expected))
	err = dbutil.ScanAllMap(rows, actual)
	assert.NoError(t, err, msgAndArgs...)

	return assert.Equal(t, simplifyMap(expected), actual, msgAndArgs...)
}

func assertList(t assert.TestingT, db *sqlx.DB, query string, args []any, expected []any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}

	rows, err := db.QueryContext(context.Background(), query, args...)
	assert.NoError(t, err, msgAndArgs...)

	actual := make([]any, 0, len(expected))
	actual, err = dbutil.ScanAllSlice(rows, actual)
	assert.NoError(t, err, msgAndArgs...)

	return assert.Equal(t, simplifySlice(expected), actual, msgAndArgs...)
}

func assertSet(t assert.TestingT, db *sqlx.DB, query string, args []any, expected []any, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}

	rows, err := db.QueryContext(context.Background(), query, args...)
	assert.NoError(t, err, msgAndArgs...)

	actual := make([]any, 0, len(expected))
	actual, err = dbutil.ScanAllSlice(rows, actual)
	assert.NoError(t, err, msgAndArgs...)

	return assert.ElementsMatch(t, simplifySlice(expected), actual, msgAndArgs...)
}

func simplifyMap(m map[string]any) map[string]any {
	simplified := make(map[string]any, len(m))
	for k, v := range m {
		simplified[k] = simplifyValue(v)
	}
	return simplified
}

func simplifySlice(s []any) []any {
	simplified := make([]any, len(s))
	for i, v := range s {
		simplified[i] = simplifyValue(v)
	}
	return simplified
}

func simplifyValue(v any) any {
	switch typed := v.(type) {
	case json.Number:
		// try to convert to int first, then float
		if i, err := typed.Int64(); err == nil {
			return i
		} else if f, err := typed.Float64(); err == nil {
			return f
		}
	}

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

type tHelper = interface {
	Helper()
}
