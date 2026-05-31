package unit

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

func mustDF(t *testing.T, columns []frame.SeriesInput) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: columns})
	if err != nil {
		t.Fatalf("не вдалося створити DataFrame: %v", err)
	}
	return df
}

func TestDataFrameShiftLag(t *testing.T) {
	df := mustDF(t, []frame.SeriesInput{
		{Name: "x", Values: []any{int64(10), int64(20), int64(30)}},
	})

	shifted, err := df.Shift(1)
	if err != nil {
		t.Fatalf("shift failed: %v", err)
	}
	if shifted.Height() != 3 {
		t.Fatalf("очікували висоту 3, отримали %d", shifted.Height())
	}

	col, _ := shifted.Series("x")
	if col.Value(0) != nil {
		t.Fatalf("перший рядок після shift(1) має бути null, отримали %v", col.Value(0))
	}
	if col.Value(1) != int64(10) || col.Value(2) != int64(20) {
		t.Fatalf("некоректний зсув: %v, %v", col.Value(1), col.Value(2))
	}
}

func TestDataFrameIterSlices(t *testing.T) {
	df := mustDF(t, []frame.SeriesInput{
		{Name: "id", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5)}},
	})

	chunks := df.IterSlices(2)
	if len(chunks) != 3 {
		t.Fatalf("очікували 3 частини, отримали %d", len(chunks))
	}
	if chunks[0].Height() != 2 || chunks[1].Height() != 2 || chunks[2].Height() != 1 {
		t.Fatalf("некоректні розміри частин: %d, %d, %d",
			chunks[0].Height(), chunks[1].Height(), chunks[2].Height())
	}

	idCol, _ := chunks[2].Series("id")
	if idCol.Value(0) != int64(5) {
		t.Fatalf("остання частина має містити id=5, отримали %v", idCol.Value(0))
	}
}

func TestDataFrameSampleDeterministic(t *testing.T) {
	df := mustDF(t, []frame.SeriesInput{
		{Name: "v", Values: []any{int64(1), int64(2), int64(3), int64(4), int64(5), int64(6)}},
	})

	a := df.Sample(3, 42)
	b := df.Sample(3, 42)
	c := df.Sample(3, 99)

	if a.Height() != 3 || b.Height() != 3 {
		t.Fatalf("sample має повертати 3 рядки")
	}

	eqAB, err := a.Equals(b)
	if err != nil || !eqAB {
		t.Fatalf("однаковий seed має давати однаковий sample: eq=%v err=%v", eqAB, err)
	}

	eqAC, err := a.Equals(c)
	if err != nil {
		t.Fatalf("equals failed: %v", err)
	}
	if eqAC {
		t.Fatal("різні seed не повинні давати ідентичний sample")
	}
}

func TestDataFrameSerializeRoundtrip(t *testing.T) {
	df := mustDF(t, []frame.SeriesInput{
		{Name: "name", Values: []any{"alpha", "beta"}},
		{Name: "n", Values: []any{int64(1), int64(2)}},
	})

	raw, err := df.Serialize()
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}

	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("очікували 2 рядки в JSON, отримали %d", len(rows))
	}
	if rows[0]["name"] != "alpha" || rows[1]["n"] != float64(2) {
		t.Fatalf("некоректний вміст JSON: %#v", rows)
	}

	if !bytes.Contains(raw, []byte(`"name"`)) {
		t.Fatalf("JSON не містить очікуваних ключів: %s", string(raw))
	}
}

func TestDataFrameIsDuplicatedAndUnique(t *testing.T) {
	// IsDuplicated порівнює повні рядки (усі колонки), а не окремий ключ.
	df := mustDF(t, []frame.SeriesInput{
		{Name: "k", Values: []any{"a", "a", "b"}},
		{Name: "v", Values: []any{int64(1), int64(1), int64(2)}},
	})

	dup := df.IsDuplicated()
	uniq := df.IsUnique()
	if dup.Len() != 3 || uniq.Len() != 3 {
		t.Fatalf("очікували серії довжини 3")
	}
	if dup.Value(0) != true || dup.Value(1) != true || dup.Value(2) != false {
		t.Fatalf("рядки 0 і 1 — дублікати, рядок 2 — унікальний: %v, %v, %v",
			dup.Value(0), dup.Value(1), dup.Value(2))
	}
	if uniq.Value(2) != true || uniq.Value(0) != false {
		t.Fatalf("is_unique має бути інверсією is_duplicated")
	}
}

func TestDataFrameItemErrors(t *testing.T) {
	df := mustDF(t, []frame.SeriesInput{
		{Name: "x", Values: []any{int64(1)}},
	})

	if _, err := df.Item(-1, "x"); err == nil {
		t.Fatal("очікували помилку для від'ємного індексу рядка")
	}
	if _, err := df.Item(0, "missing"); err == nil {
		t.Fatal("очікували помилку для неіснуючої колонки")
	}

	val, err := df.Item(0, "x")
	if err != nil || val != int64(1) {
		t.Fatalf("коректний Item(0,x): val=%v err=%v", val, err)
	}
}

func TestDataFrameExtendAndEquals(t *testing.T) {
	left := mustDF(t, []frame.SeriesInput{
		{Name: "id", Values: []any{int64(1), int64(2)}},
	})
	right := mustDF(t, []frame.SeriesInput{
		{Name: "id", Values: []any{int64(3)}},
	})

	extended, err := left.Extend(right)
	if err != nil {
		t.Fatalf("extend failed: %v", err)
	}
	if extended.Height() != 3 {
		t.Fatalf("після extend очікували 3 рядки, отримали %d", extended.Height())
	}

	clone := left.Clone()
	eq, err := left.Equals(clone)
	if err != nil || !eq {
		t.Fatalf("clone має бути рівним оригіналу: eq=%v err=%v", eq, err)
	}

	neq, err := left.Equals(extended)
	if err != nil || neq {
		t.Fatalf("різні DataFrame не повинні бути рівними: eq=%v err=%v", neq, err)
	}
}

func TestDataFrameToNumpyAndShape(t *testing.T) {
	df := mustDF(t, []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(2)}},
		{Name: "b", Values: []any{"x", "y"}},
	})

	shape := df.Shape()
	if shape[0] != 2 || shape[1] != 2 {
		t.Fatalf("shape: очікували [2,2], отримали %v", shape)
	}

	matrix := df.ToNumpy()
	if len(matrix) != 2 || len(matrix[0]) != 2 {
		t.Fatalf("to_numpy: очікували 2x2, отримали %d x %d", len(matrix), len(matrix[0]))
	}
	if matrix[0][0] != int64(1) || matrix[1][1] != "y" {
		t.Fatalf("некоректний вміст to_numpy: %#v", matrix)
	}
}

func TestDataFrameGatherEvery(t *testing.T) {
	df := mustDF(t, []frame.SeriesInput{
		{Name: "n", Values: []any{int64(0), int64(1), int64(2), int64(3), int64(4)}},
	})

	every := df.GatherEvery(2, 1)
	if every.Height() != 2 {
		t.Fatalf("gather_every(2,1): очікували 2 рядки, отримали %d", every.Height())
	}
	col, _ := every.Series("n")
	if col.Value(0) != int64(1) || col.Value(1) != int64(3) {
		t.Fatalf("некоректний gather_every: %v, %v", col.Value(0), col.Value(1))
	}
}
