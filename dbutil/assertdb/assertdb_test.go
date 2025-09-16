package assertdb_test

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/nyaruka/gocommon/dbutil/assertdb"
)

func TestAssertQuery(t *testing.T) {
	db := getTestDB()

	defer func() {
		db.MustExec(`DROP TABLE foo`)
	}()

	db.MustExec(`CREATE TABLE foo (id serial NOT NULL PRIMARY KEY, name VARCHAR(3), age INT, status VARCHAR(1))`)
	db.MustExec(`INSERT INTO foo (name, age, status) VALUES ('Ann', 32, 'A'), ('Bob', 44, 'A'), ('Cat', 23, 'A'), ('Dan', 50, 'B');`)

	assertdb.Query(t, db, `SELECT COUNT(*) FROM foo`).Returns(4)
	assertdb.Query(t, db, `SELECT name, age, status FROM foo WHERE id = $1`, 2).Columns(map[string]any{"name": "Bob", "age": 44, "status": "A"})
	assertdb.Query(t, db, `SELECT name FROM foo ORDER BY name`).Slice([]any{"Ann", "Bob", "Cat", "Dan"})
	assertdb.Query(t, db, `SELECT status, count(*) FROM foo GROUP BY 1`).Map(map[string]any{"A": 3, "B": 1})
}

// returns an open test database pool
func getTestDB() *sqlx.DB {
	return sqlx.MustOpen("postgres", "postgres://gocommon_test:temba@localhost/gocommon_test?sslmode=disable&Timezone=UTC")
}
