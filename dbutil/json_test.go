package dbutil_test

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/nyaruka/gocommon/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanJSON(t *testing.T) {
	ctx := context.Background()
	db := getTestDB()

	defer func() {
		db.MustExec(`DROP TABLE foo`)
	}()

	db.MustExec(`CREATE TABLE foo (id serial NOT NULL PRIMARY KEY, uuid UUID NOT NULL, name VARCHAR(10), age INT)`)
	db.MustExec(`INSERT INTO foo (uuid, name, age) VALUES('11163af6-a2ee-486d-b6dc-984174f10eec', 'Bob', 40)`)
	db.MustExec(`INSERT INTO foo (uuid, name, age) VALUES('57d3f887-9ae1-4292-8fa4-ffc11e31e2f7', 'Cathy', 30)`)
	db.MustExec(`INSERT INTO foo (uuid, name, age) VALUES('a5850c89-dd29-46f6-9de1-d068b3c2db94', 'George', -1)`)

	type foo struct {
		UUID string `json:"uuid" validate:"required"`
		Name string `json:"name"`
		Age  int    `json:"age" validate:"min=0"`
	}

	queryRows := func(sql string, args ...any) *sqlx.Rows {
		rows, err := db.QueryxContext(ctx, sql, args...)
		require.NoError(t, err)
		require.True(t, rows.Next())
		return rows
	}

	// if query returns valid JSON which can be unmarshaled into our struct, all good
	rows := queryRows(`SELECT ROW_TO_JSON(r) FROM (SELECT f.uuid, f.name, f.age FROM foo f WHERE id = 1) r`)

	f := &foo{}
	err := dbutil.ScanAndValidateJSON(rows, f)
	assert.NoError(t, err)
	assert.Equal(t, "11163af6-a2ee-486d-b6dc-984174f10eec", f.UUID)
	assert.Equal(t, "Bob", f.Name)
	assert.Equal(t, 40, f.Age)

	rows = queryRows(`SELECT ROW_TO_JSON(r) FROM (SELECT f.uuid, f.name, f.age FROM foo f WHERE id = 2) r`)

	err = dbutil.ScanAndValidateJSON(rows, f)
	assert.NoError(t, err)
	assert.Equal(t, "57d3f887-9ae1-4292-8fa4-ffc11e31e2f7", f.UUID)
	assert.Equal(t, "Cathy", f.Name)
	assert.Equal(t, 30, f.Age)

	// error if row value is not JSON
	rows = queryRows(`SELECT id FROM foo f WHERE id = 1`)
	err = dbutil.ScanAndValidateJSON(rows, f)
	assert.EqualError(t, err, `error scanning row JSON: sql: Scan error on column index 0, name "id": unsupported Scan, storing driver.Value type int64 into type *json.RawMessage`)

	// error if we can't marshal into the struct
	rows = queryRows(`SELECT ROW_TO_JSON(r) FROM (SELECT f.uuid as uuid, f.name AS age FROM foo f WHERE id = 1) r`)
	err = dbutil.ScanAndValidateJSON(rows, f)
	assert.EqualError(t, err, "error unmarshalling row JSON: json: cannot unmarshal string into Go struct field foo.age of type int")

	// error if rows aren't ready to be scanned - e.g. next hasn't been called
	rows, err = db.QueryxContext(ctx, `SELECT ROW_TO_JSON(r) FROM (SELECT f.uuid as uuid, f.name AS name FROM foo f WHERE id = 1) r`)
	require.NoError(t, err)
	err = dbutil.ScanAndValidateJSON(rows, f)
	assert.EqualError(t, err, "error scanning row JSON: sql: Scan called without calling Next")

	// error if we request validation and returned JSON is invalid
	rows = queryRows(`SELECT ROW_TO_JSON(r) FROM (SELECT f.uuid, f.name, f.age FROM foo f WHERE id = 3) r`)

	err = dbutil.ScanAndValidateJSON(rows, f)
	assert.EqualError(t, err, "error validating unmarsalled JSON: Key: 'foo.Age' Error:Field validation for 'Age' failed on the 'min' tag")

	// no error if we don't do validation
	rows = queryRows(`SELECT ROW_TO_JSON(r) FROM (SELECT f.uuid, f.name, f.age FROM foo f WHERE id = 3) r`)

	err = dbutil.ScanJSON(rows, f)
	assert.NoError(t, err)
	assert.Equal(t, "a5850c89-dd29-46f6-9de1-d068b3c2db94", f.UUID)
	assert.Equal(t, "George", f.Name)
	assert.Equal(t, -1, f.Age)
}
