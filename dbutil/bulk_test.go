package dbutil_test

import (
	"context"
	"testing"

	_ "github.com/lib/pq"
	"github.com/nyaruka/gocommon/dbutil"
	"github.com/nyaruka/gocommon/dbutil/assertdb"
	"github.com/stretchr/testify/assert"
	"github.com/vinovest/sqlx"
)

func TestBulkSQL(t *testing.T) {
	db := getTestDB()

	defer func() { db.MustExec(`DROP TABLE foo`) }()

	db.MustExec(`CREATE TABLE foo (id serial NOT NULL PRIMARY KEY, name VARCHAR(10))`)
	db.MustExec(`INSERT INTO foo (name) VALUES('Bob')`)
	db.MustExec(`INSERT INTO foo (name) VALUES('Cathy')`)
	db.MustExec(`INSERT INTO foo (name) VALUES('George')`)

	type contact struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
	}

	// error if we use a query without a VALUES clause
	_, _, err := dbutil.BulkSQL(db, `UPDATE foo SET name = :name WHERE id = :id`, []any{contact{ID: 1, Name: "Bob"}})
	assert.EqualError(t, err, "error extracting VALUES from sql: UPDATE foo SET name = ? WHERE id = ?")

	// try with missing parentheses
	_, _, err = dbutil.BulkSQL(db, `INSERT INTO foo (id, name) VALUES(:id, :name`, []any{contact{ID: 1, Name: "Bob"}})
	assert.EqualError(t, err, "error converting bulk insert args: missing closing bracket in VALUES")

	sql := `INSERT INTO foo (id, name) VALUES(:id, :name)`

	// try with zero structs
	_, _, err = dbutil.BulkSQL(db, sql, []any{})
	assert.EqualError(t, err, "can't generate bulk sql with zero structs")

	// try with one struct
	query, args, err := dbutil.BulkSQL(db, sql, []any{contact{ID: 1, Name: "Bob"}})
	assert.NoError(t, err)
	assert.Equal(t, `INSERT INTO foo (id, name) VALUES($1, $2)`, query)
	assert.Equal(t, []any{1, "Bob"}, args)

	// try with multiple...
	query, args, err = dbutil.BulkSQL(db, sql, []any{contact{ID: 1, Name: "Bob"}, contact{ID: 2, Name: "Cathy"}, contact{ID: 3, Name: "George"}})
	assert.NoError(t, err)
	assert.Equal(t, `INSERT INTO foo (id, name) VALUES($1, $2),($3, $4),($5, $6)`, query)
	assert.Equal(t, []any{1, "Bob", 2, "Cathy", 3, "George"}, args)

	// try with multiple...
	query, args, err = dbutil.BulkSQL(db, sql, []any{contact{ID: 1, Name: "Bob"}, contact{ID: 2, Name: "Cathy"}, contact{ID: 3, Name: "George"}})
	assert.NoError(t, err)
	assert.Equal(t, `INSERT INTO foo (id, name) VALUES($1, $2),($3, $4),($5, $6)`, query)
	assert.Equal(t, []any{1, "Bob", 2, "Cathy", 3, "George"}, args)

	// try a select
	sql = `SELECT * FROM foo WHERE (id, name) IN (VALUES(CAST(:id AS int), :name))`

	query, args, err = dbutil.BulkSQL(db, sql, []any{contact{ID: 1, Name: "Bob"}, contact{ID: 2, Name: "Cathy"}})
	assert.NoError(t, err)
	assert.Equal(t, `SELECT * FROM foo WHERE (id, name) IN (VALUES(CAST($1 AS int), $2),(CAST($3 AS int), $4))`, query)
	assert.Equal(t, []any{1, "Bob", 2, "Cathy"}, args)
}

func TestBulkQuery(t *testing.T) {
	ctx := context.Background()
	db := getTestDB()

	defer func() {
		db.MustExec(`DROP TABLE foo`)
	}()

	db.MustExec(`CREATE TABLE foo (id serial NOT NULL PRIMARY KEY, name VARCHAR(3), age INT)`)

	type foo struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
		Age  int    `db:"age"`
	}

	foo1 := &foo{Name: "Bob", Age: 64}
	foo2 := &foo{Name: "Jon", Age: 34}

	// error if no VALUES clause
	err := dbutil.BulkQuery(ctx, db, `INSERT INTO foo (name, age) RETURNING id`, []any{foo1, foo2})
	assert.EqualError(t, err, "error extracting VALUES from sql: INSERT INTO foo (name, age) RETURNING id")

	sql := `INSERT INTO foo (name, age) VALUES(:name, :age) RETURNING id`

	// noop with zero structs
	err = dbutil.BulkQuery(ctx, db, sql, []*foo{})
	assert.NoError(t, err)

	// returned ids are scanned into structs
	err = dbutil.BulkQuery(ctx, db, sql, []*foo{foo1, foo2})
	assert.NoError(t, err)
	assert.Equal(t, 1, foo1.ID)
	assert.Equal(t, 2, foo2.ID)

	assertdb.Query(t, db, `SELECT count(*) FROM foo WHERE name = 'Bob' AND age = 64`).Returns(1)
	assertdb.Query(t, db, `SELECT count(*) FROM foo WHERE name = 'Jon' AND age = 34`).Returns(1)

	// returning ids is optional
	foo3 := &foo{Name: "Jim", Age: 54}
	err = dbutil.BulkQuery(ctx, db, `INSERT INTO foo (name, age) VALUES(:name, :age)`, []any{foo3})
	assert.NoError(t, err)
	assert.Equal(t, 0, foo3.ID)

	assertdb.Query(t, db, `SELECT count(*) FROM foo WHERE name = 'Jim' AND age = 54`).Returns(1)

	// try with a struct that is invalid
	foo4 := &foo{Name: "Jonny", Age: 34}
	err = dbutil.BulkQuery(ctx, db, `INSERT INTO foo (name, age) VALUES(:name, :age)`, []any{foo4})
	assert.EqualError(t, err, "error making bulk query: pq: value too long for type character varying(3)")
	assert.Equal(t, 0, foo4.ID)
}

// returns an open test database pool
func getTestDB() *sqlx.DB {
	return sqlx.MustOpen("postgres", "postgres://gocommon_test:temba@postgres/gocommon_test?sslmode=disable&Timezone=UTC")
}
