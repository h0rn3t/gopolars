package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func TestObjectStorePathResolutionForCSV(t *testing.T) {
	root := t.TempDir()
	bucketDir := filepath.Join(root, "bucket-a")
	if err := os.MkdirAll(bucketDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	localPath := filepath.Join(bucketDir, "data.csv")

	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2)}},
			{Name: "city", Values: []any{"kyiv", "lviv"}},
		},
	})
	if err != nil {
		t.Fatalf("new dataframe failed: %v", err)
	}
	if err := df.WriteCSV(polars.WriteCSVInput{Path: localPath, IncludeHeader: true, Separator: ','}); err != nil {
		t.Fatalf("write csv failed: %v", err)
	}

	t.Setenv("GOPOLARS_S3_ROOT", root)
	io := polars.NewIO()
	out, err := io.ReadCSV(polars.ReadCSVInput{
		Path:      "s3://bucket-a/data.csv",
		HasHeader: true,
		Separator: ',',
	})
	if err != nil {
		t.Fatalf("read object-store csv failed: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("unexpected object-store read height: %d", out.Height())
	}
}
