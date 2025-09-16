package assertdb

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/jmoiron/sqlx"
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

func (a *Assert) Run(t *testing.T, db *sqlx.DB) bool {
	if a.Returns != nil {
		return assertReturns(t, db, a.Query, a.Args, a.Returns, "returns assertion failed for '%q'", a.Query)
	} else if a.Columns != nil {
		return assertColumns(t, db, a.Query, a.Args, a.Columns, "columns assertion failed for '%q'", a.Query)
	} else if a.Map != nil {
		return assertMap(t, db, a.Query, a.Args, a.Map, "map assertion failed for '%q'", a.Query)
	} else if a.List != nil {
		return assertList(t, db, a.Query, a.Args, a.List, "list assertion failed for '%q'", a.Query)
	} else if a.Set != nil {
		return assertSet(t, db, a.Query, a.Args, a.Set, "set assertion failed for '%q'", a.Query)
	} else {
		t.Errorf("no assertion specified for '%q'", a.Query)
	}

	return true
}

func (a *Assert) UnmarshalJSON(data []byte) error {
	type Alias Assert

	// so we can use json.Number
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	aux := (*Alias)(a)
	if err := decoder.Decode(aux); err != nil {
		return err
	}

	return nil
}
