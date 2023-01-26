package dbutil_test

import (
	"database/sql/driver"
	"testing"

	"github.com/nyaruka/gocommon/dbutil"
	"github.com/nyaruka/gocommon/dbutil/assertdb"
	"github.com/stretchr/testify/assert"
)

type TestString string

func (s *TestString) Scan(v any) error {
	return dbutil.ScanNullString(v, s)
}

func (s TestString) Value() (driver.Value, error) {
	return dbutil.NullStringValue(s)
}

func TestNullString(t *testing.T) {
	db := getTestDB()

	defer func() { db.MustExec(`DROP TABLE foo`) }()

	db.MustExec(`CREATE TABLE foo (id int PRIMARY KEY, name VARCHAR(10))`)
	db.MustExec(`INSERT INTO foo (id, name) VALUES(1, 'Bob')`)
	db.MustExec(`INSERT INTO foo (id, name) VALUES(2, '')`)
	db.MustExec(`INSERT INTO foo (id, name) VALUES(3, NULL)`)

	type foo struct {
		ID   int        `db:"id"`
		Name TestString `db:"name"`
	}

	var foos []foo
	err := db.Select(&foos, `SELECT * FROM foo ORDER BY id`)
	assert.NoError(t, err)
	assert.Equal(t, TestString("Bob"), foos[0].Name)
	assert.Equal(t, TestString(""), foos[1].Name)
	assert.Equal(t, TestString(""), foos[2].Name)

	var s TestString
	err = db.Get(&s, `SELECT id FROM foo WHERE id = 1`)
	assert.EqualError(t, err, `sql: Scan error on column index 0, name "id": unable to scan int64 as *dbutil_test.TestString`)

	db.MustExec(`INSERT INTO foo (id, name) VALUES(4, $1)`, TestString("Eve"))
	db.MustExec(`INSERT INTO foo (id, name) VALUES(5, $1)`, TestString(""))

	assertdb.Query(t, db, `SELECT count(*) FROM foo WHERE id = 4 AND name = 'Eve'`).Returns(1)
	assertdb.Query(t, db, `SELECT count(*) FROM foo WHERE id = 5 AND name IS NULL`).Returns(1)
}
