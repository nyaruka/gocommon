package assertdb_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/nyaruka/gocommon/dbutil/assertdb"
	"github.com/nyaruka/gocommon/jsonx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type assertTest struct {
	Assert json.RawMessage `json:"assert"`
	Pass   bool            `json:"pass"`
	Actual json.RawMessage `json:"actual,omitempty"`
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

	mt := &MockTestingT{t: t}

	for i, tc := range tcs {
		a := &assertdb.Assert{}
		err = json.Unmarshal(tc.Assert, a)
		require.NoError(t, err, "%d: could not unmarshal assert: %s", i, err)

		if tc.Pass {
			assert.True(mt, a.Check(mt, db), "%d: expected check to return true", i)
			assert.Len(t, mt.errors, 0)

			actual := a.Actual(t, db)
			assert.Equal(t, a, actual, "%d: actual does not match original", i)

		} else {
			assert.False(mt, a.Check(mt, db), "%d: expected check to return false", i)
			assert.Len(t, mt.errors, 1)
			mt.errors = nil

			actual := a.Actual(t, db)
			assert.JSONEq(t, string(tc.Actual), string(jsonx.MustMarshal(actual)), "%d: actual doesn't match", i)
		}

		// re-marshal and check we get the same thing back
		marshaled, err := json.Marshal(a)
		require.NoError(t, err, "%d: could not marshal assert: %s", i, err)

		assert.JSONEq(t, string(tc.Assert), string(marshaled), "%d: marshaled assert does not match original", i)
	}

	assertdb.Query(t, db, `SELECT COUNT(*) FROM foo`).Returns(4)
	assertdb.Query(t, db, `SELECT name, age, status FROM foo WHERE id = $1`, 2).Columns(map[string]any{"name": "Bob", "age": 44, "status": "A"})
	assertdb.Query(t, db, `SELECT status, count(*) FROM foo GROUP BY 1`).Map(map[string]any{"A": 3, "B": 1})
	assertdb.Query(t, db, `SELECT name FROM foo ORDER BY name`).List([]any{"Ann", "Bob", "Cat", "Dan"})
}

type MockTestingT struct {
	t      *testing.T
	errors []string
}

func (m *MockTestingT) Errorf(format string, args ...any) {
	m.errors = append(m.errors, fmt.Sprintf(format, args...))
}

func (m *MockTestingT) Context() (ctx context.Context) {
	return m.t.Context()
}

// returns an open test database pool
func getTestDB() *sqlx.DB {
	return sqlx.MustOpen("pgx", "postgres://gocommon_test:temba@localhost/gocommon_test?sslmode=disable&Timezone=UTC")
}
