package conformance

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/eugeneshershen/gopolars/pkg/frame"
	"github.com/eugeneshershen/gopolars/pkg/polars"
)

func TestNamespaceParityFixtureV05(t *testing.T) {
	df, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "s", Values: []any{"AbC", "XyZ"}},
			{Name: "ts", Values: []any{time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)}},
			{Name: "arr", Values: []any{[]any{"a", "b"}, []any{"x"}}},
		},
	})
	out, err := df.Select(
		polars.Col("s").StrLower().Alias("s_low"),
		polars.Col("s").StrReplace("b", "q").Alias("s_rep"),
		polars.Col("ts").DtMonth().Alias("m"),
		polars.Col("arr").ListContains(polars.Lit("a")).Alias("has_a"),
	)
	if err != nil {
		t.Fatalf("namespace fixture failed: %v", err)
	}
	if out.Height() != 2 {
		t.Fatalf("unexpected fixture rows")
	}
}

func TestSQLDifferentialFixtureV05(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	df, _ := polars.NewDataFrame(polars.NewDataFrameInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2), int64(3)}},
			{Name: "v", Values: []any{int64(10), int64(20), int64(30)}},
		},
	})
	lf, err := polars.SQLFromDataFrame(context.Background(), df, "SELECT id FROM t WHERE v > 10 UNION SELECT id FROM t WHERE id = 1", "t")
	if err != nil {
		t.Fatalf("sql fixture parse failed: %v", err)
	}
	out, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("sql fixture collect failed: %v", err)
	}
	if out.Height() != 3 {
		t.Fatalf("unexpected sql fixture rows")
	}
}
