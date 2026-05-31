package unit

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func TestDataFrameFilterGroupByJoinLazy(t *testing.T) {
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
			{Name: "city", Values: []any{"kyiv", "kyiv", "lviv", "lviv"}},
			{Name: "value", Values: []any{int64(10), int64(20), int64(30), int64(40)}},
		},
	})
	if err != nil {
		t.Fatalf("new dataframe failed: %v", err)
	}

	filtered, err := df.Filter(polars.Col("value").Gt(polars.Lit(int64(20))))
	if err != nil {
		t.Fatalf("filter failed: %v", err)
	}
	if filtered.Height() != 2 {
		t.Fatalf("unexpected filtered height: %d", filtered.Height())
	}

	agg, err := df.GroupBy("city").Agg(polars.Sum(polars.Col("value")))
	if err != nil {
		t.Fatalf("groupby failed: %v", err)
	}
	if agg.Height() != 2 {
		t.Fatalf("unexpected agg height: %d", agg.Height())
	}

	right, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2), int64(4)}},
			{Name: "flag", Values: []any{true, false, true}},
		},
	})
	if err != nil {
		t.Fatalf("new right dataframe failed: %v", err)
	}
	joined, err := df.Join(polars.JoinInput{
		Other:   right,
		LeftOn:  []string{"id"},
		RightOn: []string{"id"},
		How:     polars.JoinTypeInner,
	})
	if err != nil {
		t.Fatalf("join failed: %v", err)
	}
	if joined.Height() != 3 {
		t.Fatalf("unexpected join height: %d", joined.Height())
	}

	lazy := df.Lazy().
		Filter(polars.Col("value").Gt(polars.Lit(int64(15)))).
		Select(polars.Col("id"), polars.Col("value")).
		Limit(2)
	collected, err := lazy.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}
	if collected.Height() != 2 || collected.Width() != 2 {
		t.Fatalf("unexpected collected shape: %d x %d", collected.Height(), collected.Width())
	}

	streamed, err := lazy.CollectStreaming(context.Background(), 2)
	if err != nil {
		t.Fatalf("collect streaming failed: %v", err)
	}
	if streamed.Height() != 2 || streamed.Width() != 2 {
		t.Fatalf("unexpected streamed shape: %d x %d", streamed.Height(), streamed.Width())
	}
}

func TestCSVAndJSONRoundtrip(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "data.csv")
	jsonPath := filepath.Join(dir, "data.ndjson")
	ipcPath := filepath.Join(dir, "data.ipc")

	df, err := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2)}},
			{Name: "name", Values: []any{"a", "b"}},
		},
	})
	if err != nil {
		t.Fatalf("new dataframe failed: %v", err)
	}

	if err := df.WriteCSV(polars.WriteCSVInput{Path: csvPath, IncludeHeader: true, Separator: ','}); err != nil {
		t.Fatalf("write csv failed: %v", err)
	}
	if err := df.WriteJSON(polars.WriteJSONInput{Path: jsonPath, NDJSON: true}); err != nil {
		t.Fatalf("write json failed: %v", err)
	}
	if err := df.WriteIPC(polars.WriteIPCInput{Path: ipcPath}); err != nil {
		t.Fatalf("write ipc failed: %v", err)
	}

	io := polars.NewIO()
	dfCSV, err := io.ReadCSV(polars.ReadCSVInput{Path: csvPath, HasHeader: true, Separator: ','})
	if err != nil {
		t.Fatalf("read csv failed: %v", err)
	}
	dfJSON, err := io.ReadJSON(polars.ReadJSONInput{Path: jsonPath, NDJSON: true})
	if err != nil {
		t.Fatalf("read json failed: %v", err)
	}
	dfIPC, err := io.ReadIPC(polars.ReadIPCInput{Path: ipcPath})
	if err != nil {
		t.Fatalf("read ipc failed: %v", err)
	}
	if dfCSV.Height() != 2 || dfJSON.Height() != 2 || dfIPC.Height() != 2 {
		t.Fatalf("roundtrip height mismatch")
	}
	lfIPC, err := io.ScanIPC(polars.ScanIPCInput{Path: ipcPath})
	if err != nil {
		t.Fatalf("scan ipc failed: %v", err)
	}
	collected, err := lfIPC.Collect(context.Background())
	if err != nil {
		t.Fatalf("collect ipc lazy failed: %v", err)
	}
	if collected.Height() != 2 {
		t.Fatalf("unexpected ipc lazy height")
	}

	if _, err := os.Stat(csvPath); err != nil {
		t.Fatalf("csv file not found")
	}
	if _, err := os.Stat(ipcPath); err != nil {
		t.Fatalf("ipc file not found")
	}
}
