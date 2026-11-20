package dbutil_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/nyaruka/gocommon/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vinovest/sqlx"
)

func TestWriteConnector(t *testing.T) {
	ctx := context.Background()

	// temp tables can be written to in a read-only transaction so this needs a regular table
	testDB := getTestDB()
	testDB.MustExec(`CREATE TABLE readonly_test (id serial NOT NULL PRIMARY KEY)`)
	testDB.MustExec(`INSERT INTO readonly_test DEFAULT VALUES`)
	defer func() { testDB.MustExec(`DROP TABLE readonly_test`) }()

	connector, err := dbutil.NewWriteConnector(testDSN)
	require.NoError(t, err)

	db := sqlx.NewDb(sql.OpenDB(connector), "postgres")
	defer db.Close()

	// with a single connection, a new backend PID means the pool discarded the old connection
	db.SetMaxOpenConns(1)

	backendPID := func() int {
		var pid int
		require.NoError(t, db.GetContext(ctx, &pid, `SELECT pg_backend_pid()`))
		return pid
	}

	pid1 := backendPID()
	assert.Equal(t, pid1, backendPID())

	// other errors don't discard the connection
	_, err = db.ExecContext(ctx, `SELECT 1/0`)
	assert.ErrorContains(t, err, "division by zero")
	assert.Equal(t, pid1, backendPID())

	// a session made read-only behaves like a server which has become read-only
	db.MustExecContext(ctx, `SET SESSION CHARACTERISTICS AS TRANSACTION READ ONLY`)

	_, err = db.ExecContext(ctx, `DELETE FROM readonly_test`)
	assert.ErrorContains(t, err, "read-only transaction")

	pid2 := backendPID()
	assert.NotEqual(t, pid1, pid2)

	// queries can fail while reading their results
	db.MustExecContext(ctx, `SET SESSION CHARACTERISTICS AS TRANSACTION READ ONLY`)

	var ids []int
	err = db.SelectContext(ctx, &ids, `DELETE FROM readonly_test RETURNING id`)
	assert.ErrorContains(t, err, "read-only transaction")

	pid3 := backendPID()
	assert.NotEqual(t, pid2, pid3)

	// and inside a transaction the connection is discarded once the transaction ends
	tx, err := db.BeginTxx(ctx, nil)
	require.NoError(t, err)
	tx.MustExecContext(ctx, `SET TRANSACTION READ ONLY`)
	_, err = tx.ExecContext(ctx, `DELETE FROM readonly_test`)
	assert.ErrorContains(t, err, "read-only transaction")
	assert.NoError(t, tx.Rollback())

	pid4 := backendPID()
	assert.NotEqual(t, pid3, pid4)

	// and the new connection works for writes
	_, err = db.ExecContext(ctx, `DELETE FROM readonly_test`)
	assert.NoError(t, err)
	assert.Equal(t, pid4, backendPID())
}
