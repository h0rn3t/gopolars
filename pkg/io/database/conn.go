package database

import (
	"context"
	"fmt"

	"github.com/apache/arrow-adbc/go/adbc"
)

// resolvedConn carries the connection used for an operation plus enough state to
// close anything this package opened (a caller-provided connection is never
// closed here).
type resolvedConn struct {
	conn     adbc.Connection
	db       adbc.Database
	ownsConn bool
}

// Close releases resources this package opened; it is a no-op for a
// caller-provided connection.
func (r resolvedConn) Close() {
	if !r.ownsConn {
		return
	}
	if r.conn != nil {
		_ = r.conn.Close()
	}
	if r.db != nil {
		_ = r.db.Close()
	}
}

// resolveConnection returns an ADBC connection: the caller-provided one if conn
// is set, otherwise one opened from driverName + driverOptions. It errors
// clearly when neither is supplied.
func resolveConnection(ctx context.Context, conn any, driverName string, driverOptions map[string]string) (resolvedConn, error) {
	if conn != nil {
		c, ok := conn.(adbc.Connection)
		if !ok {
			return resolvedConn{}, fmt.Errorf("conn must be an adbc.Connection, got %T", conn)
		}
		return resolvedConn{conn: c, ownsConn: false}, nil
	}
	if driverName == "" {
		return resolvedConn{}, fmt.Errorf(
			"no connection: pass an open adbc.Connection via Conn, or a DriverName (with DriverOptions) to open one")
	}
	return openByDriverName(ctx, driverName, driverOptions)
}
