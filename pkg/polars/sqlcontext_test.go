package polars

import (
	"context"
	"sort"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
)

// newCtxFrame builds a small DataFrame used by SQLContext tests.
func newCtxFrame(t *testing.T, name string) DataFrame {
	t.Helper()
	df, err := NewDataFrame(NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "g", Values: []any{"x", "x", "y", "y"}},
		{Name: "v", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
	}})
	if err != nil {
		t.Fatalf("newCtxFrame: %v", err)
	}
	return df
}

// TestNewSQLContextReturnsNonNil verifies the documented contract.
func TestNewSQLContextReturnsNonNil(t *testing.T) {
	if NewSQLContext() == nil {
		t.Fatalf("NewSQLContext returned nil")
	}
}

// TestSQLContextIndependence verifies two contexts do not share state.
func TestSQLContextIndependence(t *testing.T) {
	a := NewSQLContext()
	b := NewSQLContext()
	df := newCtxFrame(t, "t")
	if err := a.Register("t", df); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if got := a.Tables(); len(got) != 1 || got[0] != "t" {
		t.Errorf("a.Tables() = %v, want [t]", got)
	}
	if got := b.Tables(); len(got) != 0 {
		t.Errorf("b.Tables() = %v, want []", got)
	}
}

// TestSQLContextRegisterAndExecute pins the Register → Execute round-trip.
func TestSQLContextRegisterAndExecute(t *testing.T) {
	ctx := NewSQLContext()
	df := newCtxFrame(t, "t")
	if err := ctx.Register("t", df); err != nil {
		t.Fatalf("Register: %v", err)
	}
	lf, err := ctx.Execute(context.Background(), "SELECT * FROM t")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if out.Height() != df.Height() {
		t.Errorf("Execute result height = %d, want %d", out.Height(), df.Height())
	}
}

// TestSQLContextRegisterEmptyName returns an error.
func TestSQLContextRegisterEmptyName(t *testing.T) {
	ctx := NewSQLContext()
	df := newCtxFrame(t, "t")
	if err := ctx.Register("", df); err == nil {
		t.Fatalf("Register(\"\") returned nil error, want non-nil")
	}
}

// TestSQLContextExecuteMissingTable returns an error.
func TestSQLContextExecuteMissingTable(t *testing.T) {
	ctx := NewSQLContext()
	_, err := ctx.Execute(context.Background(), "SELECT * FROM nope")
	if err == nil {
		t.Fatalf("Execute on missing table returned nil error, want non-nil")
	}
}

// TestSQLContextUnregister removes the table name from Tables().
func TestSQLContextUnregister(t *testing.T) {
	ctx := NewSQLContext()
	df := newCtxFrame(t, "t")
	if err := ctx.Register("t", df); err != nil {
		t.Fatalf("Register: %v", err)
	}
	ctx.Unregister("t")
	if got := ctx.Tables(); len(got) != 0 {
		t.Errorf("after Unregister, Tables() = %v, want []", got)
	}
}

// TestSQLContextTablesSorted verifies Tables() returns names in sorted order.
func TestSQLContextTablesSorted(t *testing.T) {
	ctx := NewSQLContext()
	df := newCtxFrame(t, "t")
	for _, name := range []string{"z", "a", "m"} {
		if err := ctx.Register(name, df); err != nil {
			t.Fatalf("Register(%s): %v", name, err)
		}
	}
	got := ctx.Tables()
	want := []string{"a", "m", "z"}
	if !sort.StringsAreSorted(got) {
		t.Errorf("Tables() = %v, not sorted", got)
	}
	if len(got) != len(want) {
		t.Fatalf("Tables() len = %d, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("Tables()[%d] = %q, want %q", i, got[i], w)
		}
	}
}

// TestSQLContextRegisterMany pins the bulk-registration API.
func TestSQLContextRegisterMany(t *testing.T) {
	ctx := NewSQLContext()
	df := newCtxFrame(t, "t")
	if err := ctx.RegisterMany(map[string]DataFrame{"a": df, "b": df}); err != nil {
		t.Fatalf("RegisterMany: %v", err)
	}
	got := ctx.Tables()
	sort.Strings(got)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("Tables() = %v, want [a b]", got)
	}
}

// TestSQLContextExecuteSelect runs a SELECT * and returns the right rows.
func TestSQLContextExecuteSelect(t *testing.T) {
	ctx := NewSQLContext()
	df := newCtxFrame(t, "t")
	if err := ctx.Register("t", df); err != nil {
		t.Fatalf("Register: %v", err)
	}
	lf, err := ctx.Execute(context.Background(), "SELECT * FROM t")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out, err := lf.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if out.Height() != 4 {
		t.Errorf("result height = %d, want 4", out.Height())
	}
	if out.Width() != 2 {
		t.Errorf("result width = %d, want 2", out.Width())
	}
}

// TestSQLContextExecuteGlobalDispatchesToExecute.
func TestSQLContextExecuteGlobalDispatchesToExecute(t *testing.T) {
	ctx := NewSQLContext()
	df := newCtxFrame(t, "t")
	if err := ctx.Register("t", df); err != nil {
		t.Fatalf("Register: %v", err)
	}
	lf, err := ctx.ExecuteGlobal(context.Background(), "SELECT * FROM t")
	if err != nil {
		t.Fatalf("ExecuteGlobal: %v", err)
	}
	if lf == nil {
		t.Fatalf("ExecuteGlobal returned nil LazyFrame")
	}
}

// TestSQLContextRegisterGlobals pins the global-table alias.
func TestSQLContextRegisterGlobals(t *testing.T) {
	ctx := NewSQLContext()
	df := newCtxFrame(t, "t")
	if err := ctx.RegisterGlobals(map[string]DataFrame{"g1": df}); err != nil {
		t.Fatalf("RegisterGlobals: %v", err)
	}
	if got := ctx.Tables(); len(got) != 1 || got[0] != "g1" {
		t.Errorf("Tables() = %v, want [g1]", got)
	}
}

// TestSQLContextExecuteUnparseable returns an error.
func TestSQLContextExecuteUnparseable(t *testing.T) {
	ctx := NewSQLContext()
	if _, err := ctx.Execute(context.Background(), "NOT VALID SQL AT ALL"); err == nil {
		t.Fatalf("Execute(bad) returned nil error, want non-nil")
	}
}
