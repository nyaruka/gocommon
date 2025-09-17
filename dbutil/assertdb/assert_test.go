package assertdb_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/nyaruka/gocommon/dbutil/assertdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type assertTest struct {
	Assert *assertdb.Assert `json:"assert"`
	Pass   bool             `json:"pass"`
}

func TestAsserts(t *testing.T) {
	db := getTestDB()

	defer func() {
		db.MustExec(`DROP TABLE foo`)
	}()

	db.MustExec(`CREATE TABLE foo (id serial NOT NULL PRIMARY KEY, name VARCHAR(3), age INT, status VARCHAR(1))`)
	db.MustExec(`INSERT INTO foo (name, age, status) VALUES ('Ann', 32, 'A'), ('Bob', 44, 'A'), ('Cat', 23, 'A'), ('Dan', 50, 'B');`)

	d, err := os.ReadFile("./testdata/asserts.json")
	require.NoError(t, err)

	var tcs []assertTest
	err = json.Unmarshal(d, &tcs)
	assert.NoError(t, err)

	mt := &MockTestingT{}

	for i, tc := range tcs {
		if tc.Pass {
			assert.True(mt, tc.Assert.Check(mt, db), "%d: expected check to return true", i)
			assert.Len(t, mt.errors, 0)
		} else {
			assert.False(mt, tc.Assert.Check(mt, db), "%d: expected check to return false", i)
			assert.Len(t, mt.errors, 1)
			mt.errors = nil
		}
	}

	assertdb.Query(t, db, `SELECT COUNT(*) FROM foo`).Returns(4)
	assertdb.Query(t, db, `SELECT name, age, status FROM foo WHERE id = $1`, 2).Columns(map[string]any{"name": "Bob", "age": 44, "status": "A"})
	assertdb.Query(t, db, `SELECT status, count(*) FROM foo GROUP BY 1`).Map(map[string]any{"A": 3, "B": 1})
	assertdb.Query(t, db, `SELECT name FROM foo ORDER BY name`).List([]any{"Ann", "Bob", "Cat", "Dan"})
}

type MockTestingT struct {
	errors []string
}

func (m *MockTestingT) Errorf(format string, args ...any) {
	m.errors = append(m.errors, fmt.Sprintf(format, args...))
}

// returns an open test database pool
func getTestDB() *sqlx.DB {
	return sqlx.MustOpen("postgres", "postgres://gocommon_test:temba@localhost/gocommon_test?sslmode=disable&Timezone=UTC")
}
