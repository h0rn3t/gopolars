package expr

import (
	"math"
	"testing"

	"github.com/h0rn3t/gopolars/pkg/dtypes"
)

// TestEvalBinArithmetic exercises arith() through evalBin for both int and float
// operands, including the division-by-zero and type-mismatch error paths.
func TestEvalBinArithmetic(t *testing.T) {
	t.Parallel()

	cases := []struct {
		op    string
		l, r  any
		want  any
		isErr bool
	}{
		{"add", int64(2), int64(3), int64(5), false},
		{"sub", int64(7), int64(4), int64(3), false},
		{"mul", int64(6), int64(7), int64(42), false},
		{"div", int64(8), int64(2), int64(4), false},
		{"add", 2.5, 0.5, 3.0, false},
		{"div", 1.0, 0.0, nil, true},           // float divide by zero
		{"div", int64(1), int64(0), nil, true}, // int divide by zero
		{"add", "go", "lang", "golang", false}, // string concat
		{"sub", "a", "b", nil, true},           // unsupported string op
		{"add", int64(1), 2.0, nil, true},      // type mismatch
		{"add", nil, int64(1), nil, false},     // null short-circuits to nil
	}
	for _, tc := range cases {
		got, err := evalBin(tc.op, tc.l, tc.r)
		if tc.isErr {
			if err == nil {
				t.Errorf("%s(%v,%v): expected error, got %v", tc.op, tc.l, tc.r, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s(%v,%v): unexpected error %v", tc.op, tc.l, tc.r, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s(%v,%v) = %v, want %v", tc.op, tc.l, tc.r, got, tc.want)
		}
	}
}

// TestEvalBinComparisons covers gt/ge/lt/le and eq/ne through evalBin, including
// the null-yields-null rule and the NaN special cases.
func TestEvalBinComparisons(t *testing.T) {
	t.Parallel()

	cases := []struct {
		op   string
		l, r any
		want any
	}{
		{"gt", int64(3), int64(2), true},
		{"ge", int64(2), int64(2), true},
		{"lt", 1.0, 2.0, true},
		{"le", 2.0, 1.0, false},
		{"gt", "b", "a", true},
		{"eq", int64(5), int64(5), true},
		{"ne", int64(5), int64(6), true},
		{"gt", nil, int64(1), nil},     // null comparison -> null
		{"eq", nil, int64(1), nil},     // null equality -> null
		{"eq", math.NaN(), 1.0, false}, // NaN never equal
		{"ne", math.NaN(), 1.0, true},  // NaN always not-equal
	}
	for _, tc := range cases {
		got, err := evalBin(tc.op, tc.l, tc.r)
		if err != nil {
			t.Errorf("%s(%v,%v): unexpected error %v", tc.op, tc.l, tc.r, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s(%v,%v) = %v, want %v", tc.op, tc.l, tc.r, got, tc.want)
		}
	}
}

// TestCmpHelpers exercises cmpInts/cmpFloats/cmpStrings directly across all four
// operators plus the unknown-op fallthrough.
func TestCmpHelpers(t *testing.T) {
	t.Parallel()

	if !cmpInts("gt", 3, 2) || !cmpInts("ge", 2, 2) || !cmpInts("lt", 1, 2) || !cmpInts("le", 2, 2) {
		t.Fatal("cmpInts true cases failed")
	}
	if cmpInts("gt", 2, 3) || cmpInts("??", 1, 1) {
		t.Fatal("cmpInts false/unknown cases failed")
	}

	if !cmpFloats("gt", 3, 2) || !cmpFloats("ge", 2, 2) || !cmpFloats("lt", 1, 2) || !cmpFloats("le", 2, 2) {
		t.Fatal("cmpFloats true cases failed")
	}
	if cmpFloats("??", 1, 1) {
		t.Fatal("cmpFloats unknown op should be false")
	}

	if !cmpStrings("gt", "b", "a") || !cmpStrings("ge", "a", "a") || !cmpStrings("lt", "a", "b") || !cmpStrings("le", "a", "a") {
		t.Fatal("cmpStrings true cases failed")
	}
	if cmpStrings("??", "a", "a") {
		t.Fatal("cmpStrings unknown op should be false")
	}
}

// TestCast covers the cast() helper across all supported destination types and
// the failure path.
func TestCast(t *testing.T) {
	t.Parallel()

	cases := []struct {
		v     any
		dt    dtypes.DataType
		want  any
		isErr bool
	}{
		{int64(3), dtypes.Float64, 3.0, false},
		{2.9, dtypes.Int64, int64(2), false},
		{true, dtypes.Int64, int64(1), false},
		{"42", dtypes.Int64, int64(42), false},
		{"3.5", dtypes.Float64, 3.5, false},
		{int64(7), dtypes.String, "7", false},
		{"true", dtypes.Boolean, true, false},
		{nil, dtypes.Int64, nil, false}, // nil passes through
		{"not-a-number", dtypes.Int64, nil, true},
		{[]any{1}, dtypes.Float64, nil, true},
	}
	for _, tc := range cases {
		got, err := cast(tc.v, tc.dt)
		if tc.isErr {
			if err == nil {
				t.Errorf("cast(%v,%v): expected error, got %v", tc.v, tc.dt, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("cast(%v,%v): unexpected error %v", tc.v, tc.dt, err)
			continue
		}
		if got != tc.want {
			t.Errorf("cast(%v,%v) = %v, want %v", tc.v, tc.dt, got, tc.want)
		}
	}
}

// TestKleeneBool covers three-valued boolean logic through evalBin's and/or,
// plus the kleeneBool classifier directly.
func TestKleeneBool(t *testing.T) {
	t.Parallel()

	// Classifier.
	if b, isNull, ok := kleeneBool(nil); !ok || !isNull || b {
		t.Fatalf("kleeneBool(nil) = (%v,%v,%v)", b, isNull, ok)
	}
	if b, isNull, ok := kleeneBool(true); !ok || isNull || !b {
		t.Fatalf("kleeneBool(true) = (%v,%v,%v)", b, isNull, ok)
	}
	if _, _, ok := kleeneBool(int64(1)); ok {
		t.Fatal("kleeneBool(int) should report ok=false")
	}

	// AND truth table with nulls: F∧null=F, T∧null=null, null∧null=null.
	and := func(l, r any) any { v, _ := evalBin("and", l, r); return v }
	if got := and(false, nil); got != false {
		t.Fatalf("F AND null = %v, want false", got)
	}
	if got := and(true, nil); got != nil {
		t.Fatalf("T AND null = %v, want nil", got)
	}
	if got := and(nil, nil); got != nil {
		t.Fatalf("null AND null = %v, want nil", got)
	}
	if got := and(true, true); got != true {
		t.Fatalf("T AND T = %v, want true", got)
	}

	// OR truth table with nulls: T∨null=T, F∨null=null.
	or := func(l, r any) any { v, _ := evalBin("or", l, r); return v }
	if got := or(true, nil); got != true {
		t.Fatalf("T OR null = %v, want true", got)
	}
	if got := or(false, nil); got != nil {
		t.Fatalf("F OR null = %v, want nil", got)
	}
	if got := or(false, false); got != false {
		t.Fatalf("F OR F = %v, want false", got)
	}

	// Non-bool operands error.
	if _, err := evalBin("and", int64(1), true); err == nil {
		t.Fatal("and with non-bool should error")
	}
}

// TestRoundToSigFigs covers the significant-figures rounding helper.
func TestRoundToSigFigs(t *testing.T) {
	t.Parallel()

	if got := roundToSigFigs(0, 3); got != 0 {
		t.Fatalf("roundToSigFigs(0) = %v, want 0", got)
	}
	if got := roundToSigFigs(123.456, 0); got != 0 {
		t.Fatalf("roundToSigFigs(_,0) = %v, want 0", got)
	}
	if got := roundToSigFigs(123.456, 2); got != 120 {
		t.Fatalf("roundToSigFigs(123.456,2) = %v, want 120", got)
	}
	if got := roundToSigFigs(0.0123456, 3); math.Abs(got-0.0123) > 1e-9 {
		t.Fatalf("roundToSigFigs(0.0123456,3) = %v, want ~0.0123", got)
	}
}

// TestCompileLikePattern covers the SQL LIKE -> regexp translation: '%' (any
// run), '_' (one char), and literal escaping of metacharacters.
func TestCompileLikePattern(t *testing.T) {
	t.Parallel()

	re, err := compileLikePattern("a%")
	if err != nil {
		t.Fatalf("compile a%%: %v", err)
	}
	if !re.MatchString("abc") || re.MatchString("xabc") {
		t.Fatalf("a%% anchoring wrong: abc=%v xabc=%v", re.MatchString("abc"), re.MatchString("xabc"))
	}

	re, err = compileLikePattern("a_c")
	if err != nil {
		t.Fatalf("compile a_c: %v", err)
	}
	if !re.MatchString("abc") || re.MatchString("ac") || re.MatchString("abbc") {
		t.Fatalf("a_c single-char wrong")
	}

	// '.' is a metacharacter that must be matched literally.
	re, err = compileLikePattern("a.c")
	if err != nil {
		t.Fatalf("compile a.c: %v", err)
	}
	if !re.MatchString("a.c") || re.MatchString("axc") {
		t.Fatalf("a.c literal-dot wrong")
	}
}

// TestSQLSubstr covers SUBSTRING semantics: 1-based start, length clamp, and the
// out-of-range / non-positive-length edge cases.
func TestSQLSubstr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		s      string
		start  int
		length int
		want   string
	}{
		{"hello", 1, 3, "hel"},
		{"hello", 2, 3, "ell"},
		{"hello", 0, 2, "he"},  // start < 1 clamps to 1
		{"hello", 4, 10, "lo"}, // length clamps to end
		{"hello", 9, 2, ""},    // start past end
		{"hello", 2, 0, ""},    // non-positive length
		{"héllo", 1, 2, "hé"},  // rune-aware
	}
	for _, tc := range cases {
		if got := sqlSubstr(tc.s, tc.start, tc.length); got != tc.want {
			t.Errorf("sqlSubstr(%q,%d,%d) = %q, want %q", tc.s, tc.start, tc.length, got, tc.want)
		}
	}
}
