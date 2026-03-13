package unit

import (
	"testing"

	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func TestClassifyChange(t *testing.T) {
	if polars.ClassifyChange("deprecate old API") != polars.ChangeClassDeprecating {
		t.Fatalf("expected deprecating class")
	}
	if polars.ClassifyChange("remove method X") != polars.ChangeClassBreaking {
		t.Fatalf("expected breaking class")
	}
	if polars.ClassifyChange("add new optional method") != polars.ChangeClassCompatible {
		t.Fatalf("expected compatible class")
	}
}
