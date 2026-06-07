//go:build cgo

package database

import (
	"context"
	"fmt"

	"github.com/apache/arrow-adbc/go/adbc/drivermgr"
)

// driverManagerDriverKey is the ADBC driver-manager option naming the shared
// library (or manifest) to load for a driver name.
const driverManagerDriverKey = "driver"

// openByDriverName opens an ADBC connection through the C driver manager
// (requires CGO). driverName is the shared library / manifest to load;
// driverOptions carry DSN/URI/auth passed through to the driver.
func openByDriverName(ctx context.Context, driverName string, driverOptions map[string]string) (resolvedConn, error) {
	opts := make(map[string]string, len(driverOptions)+1)
	for k, v := range driverOptions {
		opts[k] = v
	}
	if _, ok := opts[driverManagerDriverKey]; !ok {
		opts[driverManagerDriverKey] = driverName
	}

	var drv drivermgr.Driver
	db, err := drv.NewDatabaseWithContext(ctx, opts)
	if err != nil {
		return resolvedConn{}, fmt.Errorf("open ADBC database (driver %q): %w", driverName, err)
	}
	conn, err := db.Open(ctx)
	if err != nil {
		_ = db.Close()
		return resolvedConn{}, fmt.Errorf("open ADBC connection (driver %q): %w", driverName, err)
	}
	return resolvedConn{conn: conn, db: db, ownsConn: true}, nil
}
