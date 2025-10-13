package dbutil_test

import (
	"context"
	"testing"

	"github.com/nyaruka/gocommon/dbutil"
	"github.com/stretchr/testify/assert"
)

func TestStringArray(t *testing.T) {
	ctx := context.Background()
	db := getTestDB()

	defer func() {
		db.MustExec(`DROP TABLE foo`)
	}()

	db.MustExec(`CREATE TABLE foo (id serial NOT NULL PRIMARY KEY, names text[], letters CHAR(1)[])`)
	db.MustExec(`INSERT INTO foo (names, letters) VALUES('{"Ann", "Bob", "Cat"}', '{"A", "B", "C"}')`)
	db.MustExec(`INSERT INTO foo (names, letters) VALUES('{}', '{}')`)

	type Foo struct {
		ID      int                `db:"id"`
		Names   dbutil.StringArray `db:"names"`
		Letters dbutil.StringArray `db:"letters"`
	}

	var foo Foo
	err := db.GetContext(ctx, &foo, `SELECT id, names, letters FROM foo WHERE id = 1`)
	assert.NoError(t, err)
	assert.Equal(t, []string{"Ann", "Bob", "Cat"}, []string(foo.Names))
	assert.Equal(t, []string{"A", "B", "C"}, []string(foo.Letters))

	err = db.GetContext(ctx, &foo, `SELECT id, names, letters FROM foo WHERE id = 2`)
	assert.NoError(t, err)
	assert.Equal(t, []string{}, []string(foo.Names))
	assert.Equal(t, []string{}, []string(foo.Letters))
}
