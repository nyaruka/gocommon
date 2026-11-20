package dbutil

import (
	"context"
	"database/sql/driver"
	"fmt"
	"sync/atomic"

	"github.com/lib/pq"
)

// NewWriteConnector creates a connector for pools that write to a primary. The pool discards a connection once it
// gets a read-only error, because a failover can leave pooled connections on a former primary which is now read-only,
// and the driver doesn't consider those connections broken, so they'd otherwise keep failing writes until they reach
// their max lifetime.
func NewWriteConnector(dsn string) (driver.Connector, error) {
	c, err := pq.NewConnector(dsn)
	if err != nil {
		return nil, err
	}
	return &readOnlyDiscarder{c}, nil
}

// pqConn and pqRows are the interfaces database/sql uses from lib/pq's connections and rows, which wrappers must keep
type pqConn interface {
	driver.Conn
	driver.ConnBeginTx
	driver.ConnPrepareContext
	driver.ExecerContext
	driver.QueryerContext
	driver.Pinger
	driver.NamedValueChecker
	driver.SessionResetter
	driver.Validator
}

type pqRows interface {
	driver.Rows
	driver.RowsNextResultSet
	driver.RowsColumnTypeScanType
	driver.RowsColumnTypeDatabaseTypeName
	driver.RowsColumnTypeLength
	driver.RowsColumnTypePrecisionScale
}

type readOnlyDiscarder struct {
	driver.Connector
}

func (c *readOnlyDiscarder) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	pc, ok := conn.(pqConn)
	if !ok {
		conn.Close()
		return nil, fmt.Errorf("unexpected postgres connection type %T", conn)
	}
	return &discardableConn{pqConn: pc}, nil
}

// discardableConn doesn't wrap prepared statements, so errors from those don't discard the connection
type discardableConn struct {
	pqConn
	readOnly atomic.Bool
}

func (c *discardableConn) check(err error) error {
	if IsReadOnlyError(err) {
		c.readOnly.Store(true)
	}
	return err
}

func (c *discardableConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	res, err := c.pqConn.ExecContext(ctx, query, args)
	return res, c.check(err)
}

func (c *discardableConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	rows, err := c.pqConn.QueryContext(ctx, query, args)
	if err != nil {
		return nil, c.check(err)
	}
	if pr, ok := rows.(pqRows); ok {
		return &discardableRows{pqRows: pr, conn: c}, nil
	}
	return rows, nil
}

func (c *discardableConn) ResetSession(ctx context.Context) error {
	if c.readOnly.Load() {
		return driver.ErrBadConn
	}
	return c.pqConn.ResetSession(ctx)
}

func (c *discardableConn) IsValid() bool {
	return !c.readOnly.Load() && c.pqConn.IsValid()
}

// discardableRows checks errors while reading because a query can fail after it has started returning its result,
// e.g. a DELETE ... RETURNING on a read-only server
type discardableRows struct {
	pqRows
	conn *discardableConn
}

func (r *discardableRows) Next(dest []driver.Value) error {
	return r.conn.check(r.pqRows.Next(dest))
}
