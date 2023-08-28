package dbutil_test

import (
	"testing"

	"github.com/nyaruka/gocommon/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSliceScan(t *testing.T) {
	db := getTestDB()

	defer func() { db.MustExec(`DROP TABLE foo`) }()

	db.MustExec(`CREATE TABLE foo (id serial NOT NULL PRIMARY KEY, name VARCHAR(10))`)
	db.MustExec(`INSERT INTO foo (name) VALUES('Ann')`)
	db.MustExec(`INSERT INTO foo (name) VALUES('Bob')`)
	db.MustExec(`INSERT INTO foo (name) VALUES('Cat')`)

	rows, err := db.Query(`SELECT id FROM foo ORDER BY id`)
	require.NoError(t, err)

	ids := make([]int, 0, 2)
	ids, err = dbutil.SliceScan(rows, ids)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, ids)

	rows, err = db.Query(`SELECT name FROM foo ORDER BY id DESC`)
	require.NoError(t, err)

	names := make([]string, 0, 2)
	names, err = dbutil.SliceScan(rows, names)
	require.NoError(t, err)
	assert.Equal(t, []string{"Cat", "Bob", "Ann"}, names)
}

func TestMapScan(t *testing.T) {
	db := getTestDB()

	defer func() { db.MustExec(`DROP TABLE foo`) }()

	db.MustExec(`CREATE TABLE foo (id serial NOT NULL PRIMARY KEY, name VARCHAR(10))`)
	db.MustExec(`INSERT INTO foo (name) VALUES('Ann')`)
	db.MustExec(`INSERT INTO foo (name) VALUES('Bob')`)
	db.MustExec(`INSERT INTO foo (name) VALUES('Cat')`)

	rows, err := db.Query(`SELECT id, name FROM foo`)
	require.NoError(t, err)

	nameByID := make(map[int]string, 2)
	err = dbutil.MapScan(rows, nameByID)
	require.NoError(t, err)
	assert.Equal(t, map[int]string{1: "Ann", 2: "Bob", 3: "Cat"}, nameByID)

	rows, err = db.Query(`SELECT name, id FROM foo`)
	require.NoError(t, err)

	idByName := make(map[string]int, 2)
	err = dbutil.MapScan(rows, idByName)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"Ann": 1, "Bob": 2, "Cat": 3}, idByName)
}
