package expr

import (
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// TestZeroCoverageBinBuilders covers the binary-op builders that the row-wise
// evaluator can execute directly: Eq, And_, Or_, Filter, Floordiv.
func TestZeroCoverageBinBuilders(t *testing.T) {
	t.Parallel()

	// Eq builds an "eq" binary expr.
	eq := Col("a").Eq(Col("b"))
	if eq.Kind() != KindBin || eq.Op() != "eq" {
		t.Fatalf("Eq kind/op = %q/%q", eq.Kind(), eq.Op())
	}
	got, err := Eval(eq, mapRow{"a": int64(5), "b": int64(5)})
	if err != nil || got != true {
		t.Fatalf("Eq eval = %v err=%v, want true", got, err)
	}
	got, err = Eval(eq, mapRow{"a": int64(5), "b": int64(6)})
	if err != nil || got != false {
		t.Fatalf("Eq eval = %v err=%v, want false", got, err)
	}

	// And_ / Or_ are Kleene boolean ops, aliasing and/or.
	andRow := mapRow{"p": true, "q": false}
	gotAnd, err := Eval(Col("p").And_(Col("q")), andRow)
	if err != nil || gotAnd != false {
		t.Fatalf("And_ eval = %v err=%v, want false", gotAnd, err)
	}
	gotOr, err := Eval(Col("p").Or_(Col("q")), andRow)
	if err != nil || gotOr != true {
		t.Fatalf("Or_ eval = %v err=%v, want true", gotOr, err)
	}

	// Filter keeps the value when the mask is true, else nil.
	kept, err := Eval(Col("v").Filter(Lit(true)), mapRow{"v": int64(9)})
	if err != nil || kept != int64(9) {
		t.Fatalf("Filter(true) = %v err=%v, want 9", kept, err)
	}
	dropped, err := Eval(Col("v").Filter(Lit(false)), mapRow{"v": int64(9)})
	if err != nil || dropped != nil {
		t.Fatalf("Filter(false) = %v err=%v, want nil", dropped, err)
	}

	// Floordiv is an alias for FloorDiv: floor(a/b).
	fd := Col("a").Floordiv(Lit(int64(2)))
	if fd.Kind() != KindBin || fd.Op() != "floordiv" {
		t.Fatalf("Floordiv kind/op = %q/%q", fd.Kind(), fd.Op())
	}
	gotFd, err := Eval(fd, mapRow{"a": int64(7)})
	if err != nil || gotFd != float64(3) {
		t.Fatalf("Floordiv eval = %v err=%v, want 3", gotFd, err)
	}
}

// TestZeroCoverageTernaryBuilders covers the "by"-style ternary builders that the
// row-wise evaluator treats as pass-throughs (return the target value unchanged):
// BottomKBy, EwmMeanBy, ExtendConstant, InterpolateBy, MaxBy, MinBy, RollingMaxBy.
func TestZeroCoverageTernaryBuilders(t *testing.T) {
	t.Parallel()

	row := mapRow{"v": int64(42), "w": int64(1)}
	by := Col("w")

	cases := []struct {
		name   string
		e      Expr
		wantOp string
	}{
		{"bottom_k_by", Col("v").BottomKBy(by, 3), "bottom_k_by:3"},
		{"ewm_mean_by", Col("v").EwmMeanBy(by), "ewm_mean_by"},
		{"extend_constant", Col("v").ExtendConstant(Lit(int64(0))), "extend_constant"},
		{"interpolate_by", Col("v").InterpolateBy(by), "interpolate_by"},
		{"max_by", Col("v").MaxBy(by), "max_by"},
		{"min_by", Col("v").MinBy(by), "min_by"},
		{"rolling_max_by", Col("v").RollingMaxBy(by, 2), "rolling_max_by:2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.e.Kind() != KindTern {
				t.Fatalf("%s kind = %q, want ternary", tc.name, tc.e.Kind())
			}
			if tc.e.Op() != tc.wantOp {
				t.Fatalf("%s op = %q, want %q", tc.name, tc.e.Op(), tc.wantOp)
			}
			if tc.e.Target() == nil || tc.e.Target().ColName() != "v" {
				t.Fatalf("%s target not preserved: %+v", tc.name, tc.e.Target())
			}
			got, err := Eval(tc.e, row)
			if err != nil || got != int64(42) {
				t.Fatalf("%s eval = %v err=%v, want pass-through 42", tc.name, got, err)
			}
		})
	}
}

// TestZeroCoverageUnaryASTBuilders covers the series/group-context unary builders
// (cut/deserialize/explode/hist/inspect). The row-wise evaluator treats them as
// pass-throughs (returning the target value unchanged), which the frame/series
// layers rely on.
func TestZeroCoverageUnaryASTBuilders(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		e      Expr
		wantOp string
	}{
		{"cut", Col("v").Cut(), "cut"},
		{"deserialize", Col("v").Deserialize(), "deserialize"},
		{"explode", Col("v").Explode(), "explode"},
		{"hist", Col("v").Hist(), "hist"},
		{"inspect", Col("v").Inspect(), "inspect"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.e.Kind() != KindUnary {
				t.Fatalf("%s kind = %q, want unary", tc.name, tc.e.Kind())
			}
			if tc.e.Op() != tc.wantOp {
				t.Fatalf("%s op = %q, want %q", tc.name, tc.e.Op(), tc.wantOp)
			}
			if tc.e.Target() == nil || tc.e.Target().ColName() != "v" {
				t.Fatalf("%s target not preserved", tc.name)
			}
			got, err := Eval(tc.e, mapRow{"v": int64(1)})
			if err != nil || got != int64(1) {
				t.Fatalf("%s eval = %v err=%v, want pass-through 1", tc.name, got, err)
			}
		})
	}
}

// TestMeanFreeFunc covers the package-level Mean aggregation constructor (distinct
// from the (Expr).Mean method).
func TestMeanFreeFunc(t *testing.T) {
	t.Parallel()

	m := Mean(Col("v"))
	if m.Kind() != KindAgg || m.Op() != "mean" {
		t.Fatalf("Mean kind/op = %q/%q", m.Kind(), m.Op())
	}
	if m.Target() == nil || m.Target().ColName() != "v" {
		t.Fatalf("Mean target not preserved")
	}
	if name := m.Name(); name != "mean_v" {
		t.Fatalf("Mean name = %q, want mean_v", name)
	}
}

// TestRoundSigFigsBuilder covers RoundSigFigs end to end (the op is row-wise
// evaluable via roundToSigFigs).
func TestRoundSigFigsBuilder(t *testing.T) {
	t.Parallel()

	e := Col("v").RoundSigFigs(2)
	if e.Kind() != KindUnary || e.Op() != "round_sig_figs:2" {
		t.Fatalf("RoundSigFigs kind/op = %q/%q", e.Kind(), e.Op())
	}
	got, err := Eval(e, mapRow{"v": float64(123.456)})
	if err != nil || got != float64(120) {
		t.Fatalf("RoundSigFigs eval = %v err=%v, want 120", got, err)
	}
	// Non-numeric input must error.
	if _, err := Eval(e, mapRow{"v": "x"}); err == nil {
		t.Fatal("RoundSigFigs on string should error")
	}
}

// TestFromJsonBuilder covers FromJson (alias of FromJSON) including its evaluation
// over a JSON string and the non-string error path.
func TestFromJsonBuilder(t *testing.T) {
	t.Parallel()

	e := Col("j").FromJson()
	if e.Kind() != KindUnary || e.Op() != "from_json" {
		t.Fatalf("FromJson kind/op = %q/%q", e.Kind(), e.Op())
	}
	got, err := Eval(e, mapRow{"j": `{"a":1}`})
	if err != nil {
		t.Fatalf("FromJson eval err=%v", err)
	}
	m, ok := got.(map[string]any)
	if !ok || m["a"] != float64(1) {
		t.Fatalf("FromJson eval = %v, want map with a=1", got)
	}
	// Non-string input must error.
	if _, err := Eval(e, mapRow{"j": int64(1)}); err == nil {
		t.Fatal("FromJson on non-string should error")
	}
	// Invalid JSON must error.
	if _, err := Eval(Col("j").FromJson(), mapRow{"j": "not json"}); err == nil {
		t.Fatal("FromJson on invalid JSON should error")
	}
}

// TestCompareTimeAndMismatch covers the time.Time branch of compare and its
// type-mismatch error paths, which are otherwise uncovered.
func TestCompareTimeAndMismatch(t *testing.T) {
	t.Parallel()

	t1 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	timeCases := []struct {
		op   string
		l, r time.Time
		want bool
	}{
		{"gt", t2, t1, true},
		{"gt", t1, t2, false},
		{"ge", t1, t1, true},
		{"lt", t1, t2, true},
		{"le", t1, t1, true},
		{"le", t2, t1, false},
	}
	for _, tc := range timeCases {
		got, err := compare(tc.op, tc.l, tc.r)
		if err != nil {
			t.Fatalf("compare(%s, time) err=%v", tc.op, err)
		}
		if got != tc.want {
			t.Fatalf("compare(%s, time) = %v, want %v", tc.op, got, tc.want)
		}
	}

	// nil operand -> false, no error.
	if got, err := compare("gt", nil, int64(1)); err != nil || got != false {
		t.Fatalf("compare(nil) = %v err=%v", got, err)
	}

	// Type mismatches must error for every supported left type.
	mismatches := []struct {
		l, r any
	}{
		{int64(1), float64(1)},
		{float64(1), int64(1)},
		{"a", int64(1)},
		{t1, int64(1)},
	}
	for _, tc := range mismatches {
		if _, err := compare("gt", tc.l, tc.r); err == nil {
			t.Fatalf("compare mismatch (%T,%T) should error", tc.l, tc.r)
		}
	}

	// An unsupported left type errors via the final fallthrough.
	if _, err := compare("gt", true, true); err == nil {
		t.Fatal("compare(bool) should error (unsupported types)")
	}
}

// TestToFloatToInt64Default covers the default (non-numeric) branches of toFloat
// and toInt64, which return ok=false.
func TestToFloatToInt64Default(t *testing.T) {
	t.Parallel()

	if f, ok := toFloat("x"); ok || f != 0 {
		t.Fatalf("toFloat(string) = (%v,%v), want (0,false)", f, ok)
	}
	if f, ok := toFloat(nil); ok || f != 0 {
		t.Fatalf("toFloat(nil) = (%v,%v), want (0,false)", f, ok)
	}
	if i, ok := toInt64("x"); ok || i != 0 {
		t.Fatalf("toInt64(string) = (%v,%v), want (0,false)", i, ok)
	}
	// float64 converts to int64 with truncation.
	if i, ok := toInt64(float64(3.9)); !ok || i != 3 {
		t.Fatalf("toInt64(3.9) = (%v,%v), want (3,true)", i, ok)
	}
}

// TestCastDatetimeBranches covers the Datetime cast branch (both the success and
// the non-time failure path), which existing tests do not exercise.
func TestCastDatetimeBranches(t *testing.T) {
	t.Parallel()

	now := time.Date(2022, 3, 4, 5, 6, 7, 0, time.UTC)
	got, err := cast(now, dtypes.Datetime)
	if err != nil || got != now {
		t.Fatalf("cast(time, Datetime) = %v err=%v", got, err)
	}
	// A non-time value cannot cast to Datetime.
	if _, err := cast(int64(1), dtypes.Datetime); err == nil {
		t.Fatal("cast(int, Datetime) should error")
	}
	// Boolean from an unparseable string errors.
	if _, err := cast("notabool", dtypes.Boolean); err == nil {
		t.Fatal("cast(bad string, Boolean) should error")
	}
}

// TestMapColumnNamesErrorAndStructure covers MapColumnNames' error propagation and
// its recursion through literal/target/left/right/extra slots.
func TestMapColumnNamesErrorAndStructure(t *testing.T) {
	t.Parallel()

	// Error from fn aborts and propagates for a plain column.
	if _, err := MapColumnNames(Col("a"), func(string) (string, error) {
		return "", errSentinel{}
	}); err == nil {
		t.Fatal("MapColumnNames should propagate fn error for column")
	}

	// A literal is returned unchanged (KindLit branch).
	lit, err := MapColumnNames(Lit(int64(5)), func(string) (string, error) {
		return "x", nil
	})
	if err != nil || lit.Value() != int64(5) {
		t.Fatalf("MapColumnNames(lit) = %v err=%v", lit.Value(), err)
	}

	// A ternary When() exercises the left/right/extra recursion; an error in any
	// nested column must propagate.
	when := When(Col("cond"), Col("a"), Col("b"))
	if _, err := MapColumnNames(when, func(string) (string, error) {
		return "", errSentinel{}
	}); err == nil {
		t.Fatal("MapColumnNames should propagate fn error through ternary children")
	}

	// A cast wraps a target; renaming should rewrite the inner column.
	casted := Col("n").Cast(dtypes.Float64)
	out, err := MapColumnNames(casted, func(name string) (string, error) {
		return "renamed_" + name, nil
	})
	if err != nil {
		t.Fatalf("MapColumnNames(cast) err=%v", err)
	}
	if out.Target() == nil || out.Target().ColName() != "renamed_n" {
		t.Fatalf("MapColumnNames(cast) inner not rewritten: %+v", out.Target())
	}
}

type errSentinel struct{}

func (errSentinel) Error() string { return "sentinel" }
