package polars

import "os"

// writeFile is a tiny helper for tests that need to drop a hand-crafted
// fixture into a temp dir (e.g. a no-header CSV) without dragging in
// the io_test package's other helpers.
func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
