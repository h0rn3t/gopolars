package conformance

import (
	"os"
	"testing"
)

func TestNightlyV04Suite(t *testing.T) {
	if os.Getenv("GOPOLARS_NIGHTLY") != "1" {
		t.Skip("nightly suite is disabled")
	}
	TestDifferentialGroupByAgainstPythonPolars(t)
}
