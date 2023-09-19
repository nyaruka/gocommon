package dbutil_test

import (
	"context"
	"database/sql"
	"testing"

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

	queryRows := func(sql string, args ...any) *sql.Rows {
		rows, err := db.QueryContext(ctx, sql, args...)
		require.NoError(t, err)
		return rows
	}

	// if query returns valid JSON which can be unmarshaled into our struct, all good
	rows := queryRows(`SELECT ROW_TO_JSON(r) FROM (SELECT f.uuid, f.name, f.age FROM foo f WHERE id = 1) r`)
	require.True(t, rows.Next())

	f := &foo{}
	err := dbutil.ScanAndValidateJSON(rows, f)
	assert.NoError(t, err)
	assert.Equal(t, "11163af6-a2ee-486d-b6dc-984174f10eec", f.UUID)
	assert.Equal(t, "Bob", f.Name)
	assert.Equal(t, 40, f.Age)

	rows = queryRows(`SELECT ROW_TO_JSON(r) FROM (SELECT f.uuid, f.name, f.age FROM foo f WHERE id = 2) r`)
	require.True(t, rows.Next())

	err = dbutil.ScanAndValidateJSON(rows, f)
	assert.NoError(t, err)
	assert.Equal(t, "57d3f887-9ae1-4292-8fa4-ffc11e31e2f7", f.UUID)
	assert.Equal(t, "Cathy", f.Name)
	assert.Equal(t, 30, f.Age)

	// error if row value is not JSON
	rows = queryRows(`SELECT id FROM foo f WHERE id = 1`)
	require.True(t, rows.Next())

	err = dbutil.ScanAndValidateJSON(rows, f)
	assert.EqualError(t, err, `error scanning row JSON: sql: Scan error on column index 0, name "id": unsupported Scan, storing driver.Value type int64 into type *json.RawMessage`)

	// error if we can't marshal into the struct
	rows = queryRows(`SELECT ROW_TO_JSON(r) FROM (SELECT f.uuid as uuid, f.name AS age FROM foo f WHERE id = 1) r`)
	require.True(t, rows.Next())

	err = dbutil.ScanAndValidateJSON(rows, f)
	assert.EqualError(t, err, "error unmarshalling row JSON: json: cannot unmarshal string into Go struct field foo.age of type int")

	// error if rows aren't ready to be scanned - e.g. next hasn't been called
	rows = queryRows(`SELECT ROW_TO_JSON(r) FROM (SELECT f.uuid as uuid, f.name AS name FROM foo f WHERE id = 1) r`)

	err = dbutil.ScanAndValidateJSON(rows, f)
	assert.EqualError(t, err, "error scanning row JSON: sql: Scan called without calling Next")

	// error if we request validation and returned JSON is invalid
	rows = queryRows(`SELECT ROW_TO_JSON(r) FROM (SELECT f.uuid, f.name, f.age FROM foo f WHERE id = 3) r`)
	require.True(t, rows.Next())

	err = dbutil.ScanAndValidateJSON(rows, f)
	assert.EqualError(t, err, "error validating unmarsalled JSON: Key: 'foo.Age' Error:Field validation for 'Age' failed on the 'min' tag")

	rows.Close()

	// no error if we don't do validation
	rows = queryRows(`SELECT ROW_TO_JSON(r) FROM (SELECT f.uuid, f.name, f.age FROM foo f WHERE id = 3) r`)
	require.True(t, rows.Next())

	err = dbutil.ScanJSON(rows, f)
	assert.NoError(t, err)
	assert.Equal(t, "a5850c89-dd29-46f6-9de1-d068b3c2db94", f.UUID)
	assert.Equal(t, "George", f.Name)
	assert.Equal(t, -1, f.Age)

	rows.Close()

	// can all scan all rows with ScanAllJSON
	rows = queryRows(`SELECT ROW_TO_JSON(r) FROM (SELECT f.uuid, f.name, f.age FROM foo f) r`)

	var foos []*foo
	foos, err = dbutil.ScanAllJSON(rows, foos)
	assert.NoError(t, err)
	assert.Len(t, foos, 3)
}

func TestScanAllSlice(t *testing.T) {
	db := getTestDB()

	defer func() { db.MustExec(`DROP TABLE foo`) }()

	db.MustExec(`CREATE TABLE foo (id serial NOT NULL PRIMARY KEY, name VARCHAR(10))`)
	db.MustExec(`INSERT INTO foo (name) VALUES('Ann')`)
	db.MustExec(`INSERT INTO foo (name) VALUES('Bob')`)
	db.MustExec(`INSERT INTO foo (name) VALUES('Cat')`)

	rows, err := db.Query(`SELECT id FROM foo ORDER BY id`)
	require.NoError(t, err)

	ids := make([]int, 0, 2)
	ids, err = dbutil.ScanAllSlice(rows, ids)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, ids)

	rows, err = db.Query(`SELECT name FROM foo ORDER BY id DESC`)
	require.NoError(t, err)

	names := make([]string, 0, 2)
	names, err = dbutil.ScanAllSlice(rows, names)
	require.NoError(t, err)
	assert.Equal(t, []string{"Cat", "Bob", "Ann"}, names)
}

func TestScanAllMap(t *testing.T) {
	db := getTestDB()

	defer func() { db.MustExec(`DROP TABLE foo`) }()

	db.MustExec(`CREATE TABLE foo (id serial NOT NULL PRIMARY KEY, name VARCHAR(10))`)
	db.MustExec(`INSERT INTO foo (name) VALUES('Ann')`)
	db.MustExec(`INSERT INTO foo (name) VALUES('Bob')`)
	db.MustExec(`INSERT INTO foo (name) VALUES('Cat')`)

	rows, err := db.Query(`SELECT id, name FROM foo`)
	require.NoError(t, err)

	nameByID := make(map[int]string, 2)
	err = dbutil.ScanAllMap(rows, nameByID)
	require.NoError(t, err)
	assert.Equal(t, map[int]string{1: "Ann", 2: "Bob", 3: "Cat"}, nameByID)

	rows, err = db.Query(`SELECT name, id FROM foo`)
	require.NoError(t, err)

	idByName := make(map[string]int, 2)
	err = dbutil.ScanAllMap(rows, idByName)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"Ann": 1, "Bob": 2, "Cat": 3}, idByName)
}
