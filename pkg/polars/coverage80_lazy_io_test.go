package polars

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
	"github.com/h0rn3t/gopolars/pkg/frame"
)

// TestResolveObjectStorePathDefault covers the passthrough (non-object-store)
// branch of resolveObjectStorePath via a plain local read.
func TestResolveObjectStorePathDefault(t *testing.T) {
	io := NewIO()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "x.csv")
	if err := writeRawFile(t, path, "a,b\n1,2\n"); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := io.ReadCSV(ReadCSVInput{Path: path, HasHeader: true})
	if err != nil {
		t.Fatalf("ReadCSV: %v", err)
	}
	if got.Height() != 1 {
		t.Errorf("height = %d, want 1", got.Height())
	}
}

// TestObjectStorePathUnconfigured covers the env-not-set error branch of
// mapObjectStorePath (via the s3:// prefix).
func TestObjectStorePathUnconfigured(t *testing.T) {
	t.Setenv("GOPOLARS_S3_ROOT", "")
	io := NewIO()
	if _, err := io.ReadCSV(ReadCSVInput{Path: "s3://bucket/x.csv"}); err == nil {
		t.Fatalf("ReadCSV with unconfigured GOPOLARS_S3_ROOT returned nil error, want non-nil")
	}
}

// TestObjectStorePathConfigured covers the configured mapObjectStorePath
// happy path: the s3:// path is rewritten under the configured root.
func TestObjectStorePathConfigured(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("GOPOLARS_S3_ROOT", tmp)
	// Write the file at <root>/data/x.csv so s3://data/x.csv maps to it.
	if err := os.MkdirAll(filepath.Join(tmp, "data"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := writeRawFile(t, filepath.Join(tmp, "data", "x.csv"), "a,b\n1,2\n3,4\n"); err != nil {
		t.Fatalf("write: %v", err)
	}
	io := NewIO()
	got, err := io.ReadCSV(ReadCSVInput{Path: "s3://data/x.csv", HasHeader: true})
	if err != nil {
		t.Fatalf("ReadCSV via s3 mapping: %v", err)
	}
	if got.Height() != 2 {
		t.Errorf("height = %d, want 2", got.Height())
	}
}

// TestObjectStorePathGCSAndAzureUnconfigured covers the gcs:// and az://
// prefix branches of resolveObjectStorePath.
func TestObjectStorePathGCSAndAzureUnconfigured(t *testing.T) {
	t.Setenv("GOPOLARS_GCS_ROOT", "")
	t.Setenv("GOPOLARS_AZURE_ROOT", "")
	io := NewIO()
	if _, err := io.ReadParquet(ReadParquetInput{Path: "gcs://bucket/x.parquet"}); err == nil {
		t.Fatalf("gcs read with unconfigured root want error")
	}
	if _, err := io.ReadJSON(ReadJSONInput{Path: "az://bucket/x.json"}); err == nil {
		t.Fatalf("az read with unconfigured root want error")
	}
}

// TestReadParquetDirectory covers readParquetSource's directory-walk branch by
// writing a parquet file into a directory and scanning the directory.
func TestReadParquetDirectory(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "ds")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	df := newIOTestFrame(t)
	if err := df.WriteParquet(WriteParquetInput{Path: filepath.Join(dir, "part-0.parquet")}); err != nil {
		t.Fatalf("WriteParquet: %v", err)
	}
	io := NewIO()
	got, err := io.ReadParquet(ReadParquetInput{Path: dir})
	if err != nil {
		t.Fatalf("ReadParquet directory: %v", err)
	}
	if got.Height() != df.Height() {
		t.Errorf("directory read height = %d, want %d", got.Height(), df.Height())
	}
}

// TestReadParquetEmptyDirectory covers readParquetSource's empty-dir branch.
func TestReadParquetEmptyDirectory(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "empty")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	io := NewIO()
	got, err := io.ReadParquet(ReadParquetInput{Path: dir})
	if err != nil {
		t.Fatalf("ReadParquet empty directory: %v", err)
	}
	if got.Height() != 0 {
		t.Errorf("empty directory height = %d, want 0", got.Height())
	}
}

// TestScanParquetPartitionedWithPushdown covers the partition path of
// readParquetSource: Hive-style key=value directories, partition column
// injection, and pushed-down equality filter -> partition pruning.
func TestScanParquetPartitionedWithPushdown(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "events")

	writePart := func(region string, ids []any, vals []any) {
		t.Helper()
		pdir := filepath.Join(root, "region="+region)
		if err := os.MkdirAll(pdir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		part, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
			{Name: "id", Values: ids},
			{Name: "v", Values: vals},
		}})
		if err != nil {
			t.Fatalf("part frame: %v", err)
		}
		if err := part.WriteParquet(WriteParquetInput{Path: filepath.Join(pdir, "data.parquet")}); err != nil {
			t.Fatalf("WriteParquet: %v", err)
		}
	}
	writePart("us", []any{int64(1), int64(2)}, []any{1.0, 2.0})
	writePart("eu", []any{int64(3)}, []any{3.0})

	io := NewIO()

	// Full scan: both partitions, partition column "region" injected.
	lf, err := io.ScanParquet(ScanParquetInput{Path: root})
	if err != nil {
		t.Fatalf("ScanParquet: %v", err)
	}
	all, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect full: %v", err)
	}
	if all.Height() != 3 {
		t.Errorf("full partitioned height = %d, want 3", all.Height())
	}
	if _, ok := all.Series("region"); !ok {
		t.Errorf("partition column 'region' not injected")
	}

	// Pushdown: filter region == "us" prunes the eu partition (2 rows).
	lf2, err := io.ScanParquet(ScanParquetInput{Path: root})
	if err != nil {
		t.Fatalf("ScanParquet: %v", err)
	}
	pruned, err := lf2.Filter(Col("region").Eq(Lit("us"))).Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect pruned: %v", err)
	}
	if pruned.Height() != 2 {
		t.Errorf("pruned height = %d, want 2 (us only)", pruned.Height())
	}
}

// TestScanParquetMissingPathError covers resolveSource's parquet stat error.
func TestScanParquetMissingPathError(t *testing.T) {
	io := NewIO()
	lf, err := io.ScanParquet(ScanParquetInput{Path: "/does/not/exist/ds"})
	if err != nil {
		t.Fatalf("ScanParquet: %v", err)
	}
	if _, err := lf.Collect(context.Background()); err == nil {
		t.Fatalf("Collect on missing parquet path returned nil error, want non-nil")
	}
}

// TestLazySinksRoundTrip covers SinkCSV/SinkIPC/SinkNDJSON write paths.
func TestLazySinksRoundTrip(t *testing.T) {
	d := newLFFrame(t)
	dir := t.TempDir()
	ctx := context.Background()

	csvPath := filepath.Join(dir, "out.csv")
	if err := d.Lazy().SinkCSV(ctx, WriteCSVInput{Path: csvPath, IncludeHeader: true}); err != nil {
		t.Fatalf("SinkCSV: %v", err)
	}

	ipcPath := filepath.Join(dir, "out.ipc")
	if err := d.Lazy().SinkIPC(ctx, WriteIPCInput{Path: ipcPath}); err != nil {
		t.Fatalf("SinkIPC: %v", err)
	}

	ndPath := filepath.Join(dir, "out.ndjson")
	if err := d.Lazy().SinkNDJSON(ctx, WriteJSONInput{Path: ndPath}); err != nil {
		t.Fatalf("SinkNDJSON: %v", err)
	}

	// SinkBatches delegates to CollectBatches.
	ch := d.Lazy().SinkBatches(ctx, 2)
	count := 0
	for range ch {
		count++
	}
	if count == 0 {
		t.Errorf("SinkBatches produced no batches")
	}
}

// TestLazySinksUnsupported covers SinkDelta/SinkIceberg (always error).
func TestLazySinksUnsupported(t *testing.T) {
	d := newLFFrame(t)
	ctx := context.Background()
	if err := d.Lazy().SinkDelta(ctx, "/tmp/x"); err == nil {
		t.Fatalf("SinkDelta want error")
	}
	if err := d.Lazy().SinkIceberg(ctx, "/tmp/x"); err == nil {
		t.Fatalf("SinkIceberg want error")
	}
}

// TestLazySinkCSVErrorPath covers SinkCSV's collect/write error propagation.
func TestLazySinkCSVErrorPath(t *testing.T) {
	d := newLFFrame(t)
	ctx := context.Background()
	if err := d.Lazy().SinkCSV(ctx, WriteCSVInput{Path: "/no/such/dir/out.csv"}); err == nil {
		t.Fatalf("SinkCSV to unwritable path want error")
	}
	if err := d.Lazy().SinkIPC(ctx, WriteIPCInput{Path: "/no/such/dir/out.ipc"}); err == nil {
		t.Fatalf("SinkIPC to unwritable path want error")
	}
	if err := d.Lazy().SinkNDJSON(ctx, WriteJSONInput{Path: "/no/such/dir/out.ndjson"}); err == nil {
		t.Fatalf("SinkNDJSON to unwritable path want error")
	}
}

// TestLazyCollectBatches covers chunked CollectBatches.
func TestLazyCollectBatches(t *testing.T) {
	d := newLFFrame(t) // 4 rows
	ch := d.Lazy().CollectBatches(context.Background(), 2)
	total := 0
	batches := 0
	for r := range ch {
		if r.Error != nil {
			t.Fatalf("batch error: %v", r.Error)
		}
		if r.DataFrame != nil {
			total += r.DataFrame.Height()
			batches++
		}
	}
	if total != 4 {
		t.Errorf("CollectBatches total rows = %d, want 4", total)
	}
	if batches != 2 {
		t.Errorf("CollectBatches batches = %d, want 2", batches)
	}
}

// TestLazyCollectStreaming covers CollectStreaming.
func TestLazyCollectStreaming(t *testing.T) {
	d := newLFFrame(t)
	out, err := d.Lazy().CollectStreaming(context.Background(), 2)
	if err != nil {
		t.Fatalf("CollectStreaming: %v", err)
	}
	if out.Height() != 4 {
		t.Errorf("CollectStreaming height = %d, want 4", out.Height())
	}
}

// TestLazyProfile covers Profile returning a report map.
func TestLazyProfile(t *testing.T) {
	d := newLFFrame(t)
	out, report, err := d.Lazy().Select(Col("a")).Profile(context.Background())
	if err != nil {
		t.Fatalf("Profile: %v", err)
	}
	if out == nil {
		t.Fatalf("Profile DataFrame is nil")
	}
	if report["schema_version"] == nil {
		t.Errorf("Profile report missing schema_version")
	}
}

// TestLazyShow covers Show (collect + render).
func TestLazyShow(t *testing.T) {
	d := newLFFrame(t)
	s := d.Lazy().Show(2)
	if s == "" {
		t.Errorf("Show returned empty string")
	}
}

// TestLazyShowError covers Show's error path (returns the error string).
func TestLazyShowError(t *testing.T) {
	d := newLFFrame(t)
	s := d.Lazy().Select(Col("missing")).Show(2)
	if s == "" {
		t.Errorf("Show on bad plan returned empty, want error string")
	}
}

// TestLazyMapBatches covers MapBatches transform and its nil-fn branch.
func TestLazyMapBatches(t *testing.T) {
	d := newLFFrame(t)
	out, err := d.Lazy().MapBatches(func(in DataFrame) (DataFrame, error) {
		return in.Limit(1), nil
	}).Collect(context.Background())
	if err != nil {
		t.Fatalf("MapBatches Collect: %v", err)
	}
	if out.Height() != 1 {
		t.Errorf("MapBatches height = %d, want 1", out.Height())
	}

	// nil fn returns the same lazy frame.
	out2, err := d.Lazy().MapBatches(nil).Collect(context.Background())
	if err != nil {
		t.Fatalf("MapBatches(nil) Collect: %v", err)
	}
	if out2.Height() != 4 {
		t.Errorf("MapBatches(nil) height = %d, want 4", out2.Height())
	}
}

// TestLazyMergeSorted covers MergeSorted.
func TestLazyMergeSorted(t *testing.T) {
	a, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "k", Values: []any{int64(1), int64(3)}},
	}})
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	b, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "k", Values: []any{int64(2), int64(4)}},
	}})
	if err != nil {
		t.Fatalf("b: %v", err)
	}
	out, err := a.Lazy().MergeSorted(b.Lazy(), "k").Collect(context.Background())
	if err != nil {
		t.Fatalf("MergeSorted Collect: %v", err)
	}
	if out.Height() != 4 {
		t.Errorf("MergeSorted height = %d, want 4", out.Height())
	}
}

// TestLazyPipeAndPipeWithSchema covers Pipe/PipeWithSchema including nil-fn.
func TestLazyPipeAndPipeWithSchema(t *testing.T) {
	d := newLFFrame(t)
	out, err := d.Lazy().Pipe(func(in LazyFrame) LazyFrame {
		return in.Limit(2)
	}).Collect(context.Background())
	if err != nil {
		t.Fatalf("Pipe Collect: %v", err)
	}
	if out.Height() != 2 {
		t.Errorf("Pipe height = %d, want 2", out.Height())
	}

	// nil branch.
	if d.Lazy().Pipe(nil) == nil {
		t.Errorf("Pipe(nil) returned nil")
	}

	sch := d.Lazy().Schema()
	out2, err := d.Lazy().PipeWithSchema(func(in LazyFrame, _ dtypes.Schema) LazyFrame {
		return in
	}, sch).Collect(context.Background())
	if err != nil {
		t.Fatalf("PipeWithSchema Collect: %v", err)
	}
	if out2.Height() != 4 {
		t.Errorf("PipeWithSchema height = %d, want 4", out2.Height())
	}
	if d.Lazy().PipeWithSchema(nil, nil) == nil {
		t.Errorf("PipeWithSchema(nil) returned nil")
	}
}

// TestLazyRollingMean covers RollingMean / Rolling over a datetime key.
func TestLazyRollingMean(t *testing.T) {
	d := covTimeFrame(t)
	out, err := d.Lazy().Rolling(RollingMeanInput{
		By:     "ts",
		Value:  "v",
		Window: 90 * 60 * 1_000_000_000, // 90 min in ns
		Output: "roll",
	}).Collect(context.Background())
	if err != nil {
		t.Fatalf("Rolling Collect: %v", err)
	}
	if out.Height() != 4 {
		t.Errorf("Rolling height = %d, want 4", out.Height())
	}
}

// TestLazyJoinAsof covers lf.JoinAsof default How.
func TestLazyJoinAsof(t *testing.T) {
	left, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "k", Values: []any{int64(1), int64(5)}},
		{Name: "lv", Values: []any{1.0, 5.0}},
	}})
	if err != nil {
		t.Fatalf("left: %v", err)
	}
	right, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "k", Values: []any{int64(1), int64(6)}},
		{Name: "rv", Values: []any{100.0, 600.0}},
	}})
	if err != nil {
		t.Fatalf("right: %v", err)
	}
	out, err := left.Lazy().JoinAsof(JoinInput{
		Other:   right,
		LeftOn:  []string{"k"},
		RightOn: []string{"k"},
	}).Collect(context.Background())
	if err != nil {
		t.Fatalf("JoinAsof Collect: %v", err)
	}
	if out.Height() != 2 {
		t.Errorf("JoinAsof height = %d, want 2", out.Height())
	}
}

// TestLazyUpdate covers Update (executes a sub-plan and merges). The other
// frame must carry a non-empty plan, so we attach a Select.
func TestLazyUpdate(t *testing.T) {
	a := newLFFrame(t)
	other := a.Lazy().Select(Col("a"), Col("b"), Col("g"))
	out, err := a.Lazy().Update(other).Collect(context.Background())
	if err != nil {
		t.Fatalf("Update Collect: %v", err)
	}
	if out.Height() == 0 {
		t.Errorf("Update produced empty frame")
	}
}

// TestLazyDeserialize covers Serialize + Deserialize roundtrip.
func TestLazyDeserialize(t *testing.T) {
	d := newLFFrame(t)
	payload, err := d.Lazy().Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	lf2, err := d.Lazy().Deserialize(payload)
	if err != nil {
		t.Fatalf("Deserialize: %v", err)
	}
	if lf2 == nil {
		t.Fatalf("Deserialize returned nil")
	}
	// Empty payload branch.
	if _, err := d.Lazy().Deserialize(nil); err != nil {
		t.Fatalf("Deserialize(nil): %v", err)
	}
	// Invalid JSON error branch.
	if _, err := d.Lazy().Deserialize([]byte("{not json")); err == nil {
		t.Fatalf("Deserialize(bad json) want error")
	}
}

// TestLazyRemoteAndJoinWhere covers Remote and JoinWhere passthroughs.
func TestLazyRemoteAndJoinWhere(t *testing.T) {
	d := newLFFrame(t)
	if d.Lazy().Remote("http://x") == nil {
		t.Errorf("Remote returned nil")
	}
	out, err := d.Lazy().JoinWhere(Col("a").Gt(Lit(int64(2)))).Collect(context.Background())
	if err != nil {
		t.Fatalf("JoinWhere Collect: %v", err)
	}
	if out.Height() != 2 {
		t.Errorf("JoinWhere height = %d, want 2", out.Height())
	}
}
