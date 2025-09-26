package assertdb

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"sort"

	"github.com/jmoiron/sqlx"
	"github.com/nyaruka/gocommon/dbutil"
	"github.com/nyaruka/gocommon/jsonx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Assert represents a single assertion that can marshaled and unmarshaled to/from JSON
type Assert struct {
	Query   string         `json:"query"`
	Args    []any          `json:"args,omitempty"`
	Returns any            `json:"returns,omitempty"`
	Columns map[string]any `json:"columns,omitempty"`
	Map     map[string]any `json:"map,omitempty"`
	List    []any          `json:"list,omitempty"`
	Set     []any          `json:"set,omitempty"`
}

func (a *Assert) Check(t assert.TestingT, db *sqlx.DB, msgAndArgs ...any) bool {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}

	actual, err := a.actual(context.TODO(), db)
	if !assert.NoError(t, err, msgAndArgs...) {
		return false
	}

	if a.Columns != nil {
		return assert.Equal(t, a.Columns, actual.Columns, msgAndArgs...)
	} else if a.Map != nil {
		return assert.Equal(t, a.Map, actual.Map, msgAndArgs...)
	} else if a.List != nil {
		return assert.Equal(t, a.List, actual.List, msgAndArgs...)
	} else if a.Set != nil {
		return assert.Equal(t, a.Set, actual.Set, msgAndArgs...)
	} else {
		return assert.Equal(t, a.Returns, actual.Returns, msgAndArgs...)
	}
}

func (a *Assert) Actual(t require.TestingT, db *sqlx.DB) *Assert {
	actual, err := a.actual(context.TODO(), db)
	require.NoError(t, err)

	return actual
}

func (a *Assert) actual(ctx context.Context, db *sqlx.DB) (*Assert, error) {
	actual := &Assert{Query: a.Query, Args: a.Args}

	if a.Columns != nil {
		val := make(map[string]any, 10)

		if err := db.QueryRowxContext(ctx, a.Query, a.Args...).MapScan(val); err != nil {
			return nil, err
		}
		actual.Columns = normalizeMap(val)

	} else if a.Map != nil {
		rows, err := db.QueryContext(ctx, a.Query, a.Args...)
		if err != nil {
			return nil, err
		}

		val := make(map[string]any, 10)
		if err := dbutil.ScanAllMap(rows, val); err != nil {
			return nil, err
		}
		actual.Map = normalizeMap(val)

	} else if a.List != nil {
		rows, err := db.QueryContext(ctx, a.Query, a.Args...)
		if err != nil {
			return nil, err
		}

		val := make([]any, 0, 10)
		val, err = dbutil.ScanAllSlice(rows, val)
		if err != nil {
			return nil, err
		}
		actual.List = normalizeList(val)

	} else if a.Set != nil {
		rows, err := db.QueryContext(ctx, a.Query, a.Args...)
		if err != nil {
			return nil, err
		}

		val := make([]any, 0, 10)
		val, err = dbutil.ScanAllSlice(rows, val)
		if err != nil {
			return nil, err
		}
		actual.Set = normalizeSet(val)

	} else {
		var val any

		if err := db.GetContext(ctx, &val, a.Query, a.Args...); err != nil {
			return nil, err
		}
		actual.Returns = normalizeValue(val)
	}

	return actual, nil
}

func (a *Assert) MarshalJSON() ([]byte, error) {
	// no values means this is an assert on returns NULL which requires special handling so returns isn't omitted
	if a.Returns == nil && a.Columns == nil && a.Map == nil && a.List == nil && a.Set == nil {
		type assertNull struct {
			Query   string `json:"query"`
			Args    []any  `json:"args,omitempty"`
			Returns any    `json:"returns"`
		}

		return json.Marshal(&assertNull{Query: a.Query, Args: a.Args, Returns: nil})
	}

	type Alias Assert
	alias := (*Alias)(a)
	return json.Marshal(alias)
}

func (a *Assert) UnmarshalJSON(data []byte) error {
	type Alias Assert

	// so we can use json.Number
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	alias := (*Alias)(a)
	if err := decoder.Decode(alias); err != nil {
		return err
	}

	if a.Columns != nil {
		a.Columns = normalizeMap(a.Columns)
	} else if a.Map != nil {
		a.Map = normalizeMap(a.Map)
	} else if a.List != nil {
		a.List = normalizeList(a.List)
	} else if a.Set != nil {
		a.Set = normalizeSet(a.Set)
	} else {
		a.Returns = normalizeValue(a.Returns)
	}

	return nil
}

func normalizeMap(m map[string]any) map[string]any {
	norm := make(map[string]any, len(m))
	for k, v := range m {
		norm[k] = normalizeValue(v)
	}
	return norm
}

func normalizeList(s []any) []any {
	norm := make([]any, len(s))
	for i, v := range s {
		norm[i] = normalizeValue(v)
	}
	return norm
}

func normalizeSet(s []any) []any {
	norm := normalizeList(s)

	// sort by JSON representation for deterministic ordering
	sort.Slice(norm, func(i, j int) bool {
		return bytes.Compare(jsonx.MustMarshal(norm[i]), jsonx.MustMarshal(norm[j])) < 0
	})

	return norm
}

func normalizeValue(v any) any {
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
