package exec

import (
	"context"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/plan/logical"
)

func TestExecuteTailSliceReverseGather(t *testing.T) {
	t.Parallel()

	source, err := frame.FromAnyColumns(frame.FromAnyColumnsInput{
		Columns: []frame.SeriesInput{
			{Name: "id", Values: []any{int64(1), int64(2), int64(3), int64(4)}},
		},
	})
	if err != nil {
		t.Fatalf("dataframe: %v", err)
	}

	engine := New()
	ctx := context.Background()

	tail, err := engine.Execute(ctx, source, []logical.Node{{Type: logical.NodeTail, IntValue: 2}})
	if err != nil || tail.Height() != 2 {
		t.Fatalf("tail: err=%v height=%d", err, tail.Height())
	}

	sliced, err := engine.Execute(ctx, source, []logical.Node{
		{Type: logical.NodeSlice, IntValue: 1, Strings: []string{"2"}},
	})
	if err != nil || sliced.Height() != 2 {
		t.Fatalf("slice: err=%v height=%d", err, sliced.Height())
	}

	rev, err := engine.Execute(ctx, source, []logical.Node{{Type: logical.NodeReverse}})
	if err != nil {
		t.Fatalf("reverse: %v", err)
	}
	id, _ := rev.Series("id")
	if id.Value(0) != int64(4) {
		t.Fatalf("reverse first: %v", id.Value(0))
	}

	every, err := engine.Execute(ctx, source, []logical.Node{
		{Type: logical.NodeGatherEvery, IntValue: 0, Strings: []string{"2"}},
	})
	if err != nil || every.Height() != 2 {
		t.Fatalf("gather every: err=%v height=%d", err, every.Height())
	}
}
