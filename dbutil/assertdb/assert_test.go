package assertdb_test

import (
	"encoding/json"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/nyaruka/gocommon/dbutil/assertdb"
	"github.com/stretchr/testify/assert"
)

const assertsJSON = `[
	{
		"query": "SELECT COUNT(*) FROM foo",
		"returns": 4
	},
	{
		"query": "SELECT name, age, status FROM foo",
		"columns": {"name": "Ann", "age": 32, "status": "A"} 
	},
	{
		"query": "SELECT status, count(*) FROM foo GROUP BY 1",
		"map": {"A": 3, "B": 1}
	},
	{
		"query": "SELECT name FROM foo ORDER BY name",
		"list": ["Ann", "Bob", "Cat", "Dan"]
	},
	{
		"query": "SELECT name FROM foo ORDER BY name",
		"set": ["Cat", "Dan", "Ann", "Bob"]
	}
]`

func TestAsserts(t *testing.T) {
	db := getTestDB()

	defer func() {
		db.MustExec(`DROP TABLE foo`)
	}()

	db.MustExec(`CREATE TABLE foo (id serial NOT NULL PRIMARY KEY, name VARCHAR(3), age INT, status VARCHAR(1))`)
	db.MustExec(`INSERT INTO foo (name, age, status) VALUES ('Ann', 32, 'A'), ('Bob', 44, 'A'), ('Cat', 23, 'A'), ('Dan', 50, 'B');`)

	assertdb.Query(t, db, `SELECT COUNT(*) FROM foo`).Returns(4)
	assertdb.Query(t, db, `SELECT name, age, status FROM foo WHERE id = $1`, 2).Columns(map[string]any{"name": "Bob", "age": 44, "status": "A"})
	assertdb.Query(t, db, `SELECT status, count(*) FROM foo GROUP BY 1`).Map(map[string]any{"A": 3, "B": 1})
	assertdb.Query(t, db, `SELECT name FROM foo ORDER BY name`).List([]any{"Ann", "Bob", "Cat", "Dan"})

	var asserts []assertdb.Assert
	err := json.Unmarshal([]byte(assertsJSON), &asserts)
	assert.NoError(t, err)

	for _, a := range asserts {
		assert.True(t, a.Run(t, db))
	}
}

// returns an open test database pool
func getTestDB() *sqlx.DB {
	return sqlx.MustOpen("postgres", "postgres://gocommon_test:temba@localhost/gocommon_test?sslmode=disable&Timezone=UTC")
}
