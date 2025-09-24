package assertdb

import (
	"bytes"
	"encoding/json"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
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

	if a.Columns != nil {
		return assertColumns(t, db, a.Query, a.Args, a.Columns, msgAndArgs...)
	} else if a.Map != nil {
		return assertMap(t, db, a.Query, a.Args, a.Map, msgAndArgs...)
	} else if a.List != nil {
		return assertList(t, db, a.Query, a.Args, a.List, msgAndArgs...)
	} else if a.Set != nil {
		return assertSet(t, db, a.Query, a.Args, a.Set, msgAndArgs...)
	} else {
		return assertReturns(t, db, a.Query, a.Args, a.Returns, msgAndArgs...)
	}
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

	return nil
}
