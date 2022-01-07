package dbutil_test

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/nyaruka/gocommon/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadJSONRow(t *testing.T) {
	ctx := context.Background()
	db := getTestDB()

	defer func() {
		db.MustExec(`DROP TABLE foo`)
	}()

	db.MustExec(`CREATE TABLE foo (id serial NOT NULL PRIMARY KEY, uuid UUID NOT NULL, name VARCHAR(10))`)
	db.MustExec(`INSERT INTO foo (uuid, name) VALUES('11163af6-a2ee-486d-b6dc-984174f10eec', 'Bob')`)
	db.MustExec(`INSERT INTO foo (uuid, name) VALUES('57d3f887-9ae1-4292-8fa4-ffc11e31e2f7', 'Cathy')`)
	db.MustExec(`INSERT INTO foo (uuid, name) VALUES('a5850c89-dd29-46f6-9de1-d068b3c2db94', 'George')`)

	type foo struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	}

	queryRows := func(sql string, args ...interface{}) *sqlx.Rows {
		rows, err := db.QueryxContext(ctx, sql, args...)
		require.NoError(t, err)
		require.True(t, rows.Next())
		return rows
	}

	// if query returns valid JSON which can be unmarshaled into our struct, all good
	rows := queryRows(`SELECT ROW_TO_JSON(r) FROM (SELECT f.uuid as uuid, f.name AS name FROM foo f WHERE id = 1) r`)

	f := &foo{}
	err := dbutil.ReadJSONRow(rows, f)
	assert.NoError(t, err)
	assert.Equal(t, "11163af6-a2ee-486d-b6dc-984174f10eec", f.UUID)
	assert.Equal(t, "Bob", f.Name)

	rows = queryRows(`SELECT ROW_TO_JSON(r) FROM (SELECT f.uuid as uuid, f.name AS name FROM foo f WHERE id = 3) r`)

	err = dbutil.ReadJSONRow(rows, f)
	assert.NoError(t, err)
	assert.Equal(t, "a5850c89-dd29-46f6-9de1-d068b3c2db94", f.UUID)
	assert.Equal(t, "George", f.Name)

	// error if row value is not JSON
	rows = queryRows(`SELECT id FROM foo f WHERE id = 1`)
	err = dbutil.ReadJSONRow(rows, f)
	assert.EqualError(t, err, "error unmarshalling row JSON: json: cannot unmarshal number into Go value of type dbutil_test.foo")

	// error if rows aren't ready to be scanned - e.g. next hasn't been called
	rows, err = db.QueryxContext(ctx, `SELECT ROW_TO_JSON(r) FROM (SELECT f.uuid as uuid, f.name AS name FROM foo f WHERE id = 1) r`)
	require.NoError(t, err)
	err = dbutil.ReadJSONRow(rows, f)
	assert.EqualError(t, err, "error scanning row JSON: sql: Scan called without calling Next")
}
