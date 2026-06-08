//go:build !cgo

package database

import (
	"context"
	"fmt"
)

// openByDriverName requires the ADBC driver manager, which is CGO-only. Without
// CGO the caller must pass an already-open adbc.Connection via Conn.
func openByDriverName(_ context.Context, driverName string, _ map[string]string) (resolvedConn, error) {
	return resolvedConn{}, fmt.Errorf(
		"opening ADBC driver %q by name needs CGO (the ADBC driver manager); "+
			"rebuild with CGO_ENABLED=1 or pass an open adbc.Connection via Conn", driverName)
}
