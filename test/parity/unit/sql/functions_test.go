package sql

// Parity tests for the full-sql-surface function catalog (string, math,
// conditional, temporal, aggregate, and window functions). Expected values
// mirror py-polars SQL / PostgreSQL semantics; deviations are noted inline.

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/h0rn3t/gopolars/pkg/frame"
	"github.com/h0rn3t/gopolars/pkg/polars"
)

// stringsDF: s (strings with whitespace/nulls), e (emails), n (nullable strings).
func stringsDF(t *testing.T) polars.DataFrame {
	t.Helper()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "s", Values: []any{"  hello world  ", "johnson", "ABCdef", "a@b@c"}},
		{Name: "e", Values: []any{"ann@go.dev", "bob@px.io", "cid@go.dev", "dee@px.io"}},
		{Name: "n", Values: []any{"x", nil, "z", nil}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	return df
}

func col(t *testing.T, df polars.DataFrame, name string) []any {
	t.Helper()
	s, err := df.GetColumn(name)
	if err != nil {
		t.Fatalf("column %s: %v", name, err)
	}
	out := make([]any, s.Len())
	for i := 0; i < s.Len(); i++ {
		out[i] = s.Value(i)
	}
	return out
}

func wantFloat(t *testing.T, got any, want float64, label string) {
	t.Helper()
	f, ok := got.(float64)
	if !ok || math.Abs(f-want) > 1e-9 {
		t.Fatalf("%s: got %T(%v), want %v", label, got, got, want)
	}
}

func TestSQLFnStartsEndsWith(t *testing.T) {
	t.Parallel()
	out := runSQL(t, stringsDF(t), "SELECT s FROM t WHERE ENDS_WITH(s, 'son')")
	if out.Height() != 1 || col(t, out, "s")[0] != "johnson" {
		t.Fatalf("ends_with filter: got %v", out.ToDicts())
	}
	out = runSQL(t, stringsDF(t), "SELECT e FROM t WHERE STARTS_WITH(e, 'ann')")
	if out.Height() != 1 {
		t.Fatalf("starts_with filter: got %v", out.ToDicts())
	}
}

func TestSQLFnLeftRightTrims(t *testing.T) {
	t.Parallel()
	out := runSQL(t, stringsDF(t), "SELECT LEFT(e, 3) AS l, RIGHT(e, 2) AS r, LTRIM(s) AS lt, RTRIM(s) AS rt FROM t LIMIT 1")
	if v := col(t, out, "l")[0]; v != "ann" {
		t.Fatalf("LEFT: got %v", v)
	}
	if v := col(t, out, "r")[0]; v != "ev" {
		t.Fatalf("RIGHT: got %v", v)
	}
	if v := col(t, out, "lt")[0]; v != "hello world  " {
		t.Fatalf("LTRIM: got %q", v)
	}
	if v := col(t, out, "rt")[0]; v != "  hello world" {
		t.Fatalf("RTRIM: got %q", v)
	}
}

func TestSQLFnReverseInitcap(t *testing.T) {
	t.Parallel()
	out := runSQL(t, stringsDF(t), "SELECT REVERSE(e) AS r, INITCAP(s) AS i FROM t LIMIT 1")
	if v := col(t, out, "r")[0]; v != "ved.og@nna" {
		t.Fatalf("REVERSE: got %q", v)
	}
	if v := col(t, out, "i")[0]; v != "  Hello World  " {
		t.Fatalf("INITCAP: got %q", v)
	}
}

func TestSQLFnPad(t *testing.T) {
	t.Parallel()
	// LPAD/RPAD truncate past the target length (PostgreSQL semantics).
	out := runSQL(t, stringsDF(t), "SELECT LPAD(n, 5, '*') AS lp, RPAD(n, 3, 'ab') AS rp, LPAD(s, 4) AS trunc FROM t WHERE n IS NOT NULL LIMIT 1")
	if v := col(t, out, "lp")[0]; v != "****x" {
		t.Fatalf("LPAD: got %q", v)
	}
	if v := col(t, out, "rp")[0]; v != "xab" {
		t.Fatalf("RPAD: got %q", v)
	}
	if v := col(t, out, "trunc")[0]; v != "  he" {
		t.Fatalf("LPAD truncation: got %q", v)
	}
}

func TestSQLFnSplitPartStrpos(t *testing.T) {
	t.Parallel()
	out := runSQL(t, stringsDF(t), "SELECT SPLIT_PART(e, '@', 2) AS domain, STRPOS(e, '@') AS at FROM t")
	domains := col(t, out, "domain")
	if domains[0] != "go.dev" || domains[1] != "px.io" {
		t.Fatalf("SPLIT_PART: got %v", domains)
	}
	if v := col(t, out, "at")[0]; v != int64(4) {
		t.Fatalf("STRPOS: got %v, want 4", v)
	}
	// Out-of-range field yields an empty string; missing needle yields NULL.
	out = runSQL(t, stringsDF(t), "SELECT SPLIT_PART(e, '@', 9) AS missing, STRPOS(n, 'x') AS np FROM t")
	if v := col(t, out, "missing")[0]; v != "" {
		t.Fatalf("SPLIT_PART out of range: got %q, want empty", v)
	}
	np := col(t, out, "np")
	if np[0] != int64(1) {
		t.Fatalf("STRPOS found: got %v, want 1", np[0])
	}
	if np[1] != nil || np[2] != nil {
		t.Fatalf("STRPOS null/missing: got %v, want nil", np[1:3])
	}
}

func TestSQLFnLengthsRegexpConcatWS(t *testing.T) {
	t.Parallel()
	out := runSQL(t, stringsDF(t), "SELECT OCTET_LENGTH(n) AS ol, BIT_LENGTH(n) AS bl, REGEXP_LIKE(e, '^[a-z]+@go\\.dev$') AS m, CONCAT_WS('-', n, e) AS cw FROM t")
	if v := col(t, out, "ol")[0]; v != int64(1) {
		t.Fatalf("OCTET_LENGTH: got %v", v)
	}
	if v := col(t, out, "bl")[0]; v != int64(8) {
		t.Fatalf("BIT_LENGTH: got %v", v)
	}
	matches := col(t, out, "m")
	if matches[0] != true || matches[1] != false {
		t.Fatalf("REGEXP_LIKE: got %v", matches)
	}
	cw := col(t, out, "cw")
	if cw[0] != "x-ann@go.dev" {
		t.Fatalf("CONCAT_WS: got %v", cw[0])
	}
	// NULL operands are skipped, not propagated.
	if cw[1] != "bob@px.io" {
		t.Fatalf("CONCAT_WS null skip: got %v", cw[1])
	}
}

func TestSQLFnMath(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "x", Values: []any{-4.0, 8.0, 100.0, 1.0}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out := runSQL(t, df, "SELECT SIGN(x) AS sg, CBRT(x) AS cb, LOG(2, x) AS l2a, LOG2(x) AS l2b, LOG10(x) AS l10, PI() AS pi, DEGREES(x) AS dg, ATAN2(x, 1) AS at FROM t WHERE x = 8.0")
	wantFloat(t, col(t, out, "sg")[0], 1, "SIGN")
	wantFloat(t, col(t, out, "cb")[0], 2, "CBRT")
	wantFloat(t, col(t, out, "l2a")[0], 3, "LOG(2,x)")
	wantFloat(t, col(t, out, "l2b")[0], 3, "LOG2")
	wantFloat(t, col(t, out, "l10")[0], math.Log10(8), "LOG10")
	wantFloat(t, col(t, out, "pi")[0], math.Pi, "PI")
	wantFloat(t, col(t, out, "dg")[0], 8*180/math.Pi, "DEGREES")
	wantFloat(t, col(t, out, "at")[0], math.Atan2(8, 1), "ATAN2")
	out = runSQL(t, df, "SELECT SIN(x) AS s, COS(x) AS c, TAN(x) AS tn, ASIN(x) AS asn, ACOS(x) AS acs, ATAN(x) AS atn, COT(x) AS ct, RADIANS(x) AS rd FROM t WHERE x = 1.0")
	wantFloat(t, col(t, out, "s")[0], math.Sin(1), "SIN")
	wantFloat(t, col(t, out, "c")[0], math.Cos(1), "COS")
	wantFloat(t, col(t, out, "tn")[0], math.Tan(1), "TAN")
	wantFloat(t, col(t, out, "asn")[0], math.Asin(1), "ASIN")
	wantFloat(t, col(t, out, "acs")[0], math.Acos(1), "ACOS")
	wantFloat(t, col(t, out, "atn")[0], math.Atan(1), "ATAN")
	wantFloat(t, col(t, out, "ct")[0], 1/math.Tan(1), "COT")
	wantFloat(t, col(t, out, "rd")[0], math.Pi/180, "RADIANS")
}

func TestSQLFnConditional(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "a", Values: []any{int64(1), int64(5), nil}},
		{Name: "b", Values: []any{int64(4), int64(2), int64(7)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out := runSQL(t, df, "SELECT IFNULL(a, 0) AS ifn, IF(b > 3, 'big', 'small') AS cond, GREATEST(a, b) AS g, LEAST(a, b) AS l FROM t")
	ifn := col(t, out, "ifn")
	if ifn[2] != int64(0) {
		t.Fatalf("IFNULL: got %v", ifn)
	}
	conds := col(t, out, "cond")
	if conds[0] != "big" || conds[1] != "small" {
		t.Fatalf("IF: got %v", conds)
	}
	g := col(t, out, "g")
	if g[0] != int64(4) || g[1] != int64(5) {
		t.Fatalf("GREATEST: got %v", g)
	}
	// NULL operands are skipped: GREATEST(NULL, 7) = 7.
	if g[2] != int64(7) {
		t.Fatalf("GREATEST null skip: got %v", g[2])
	}
	l := col(t, out, "l")
	if l[0] != int64(1) || l[1] != int64(2) || l[2] != int64(7) {
		t.Fatalf("LEAST: got %v", l)
	}
}

func TestSQLFnTemporal(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "d", Values: []any{
			time.Date(2024, 2, 1, 13, 45, 30, 0, time.UTC), // Thursday, day 32
			time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),  // Sunday, day 365
		}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out := runSQL(t, df, "SELECT EXTRACT(YEAR FROM d) AS y, DATE_PART('month', d) AS m, DATE_PART('dow', d) AS dw, DAYOFWEEK(d) AS dw2, DAYOFYEAR(d) AS dy, ORDINAL_DAY(d) AS od, EXTRACT(MINUTE FROM d) AS mi FROM t")
	if v := col(t, out, "y")[0]; v != int64(2024) {
		t.Fatalf("EXTRACT YEAR: got %v", v)
	}
	if v := col(t, out, "m")[0]; v != int64(2) {
		t.Fatalf("DATE_PART month: got %v", v)
	}
	// ISO weekday: Monday=1 .. Sunday=7.
	if v := col(t, out, "dw")[0]; v != int64(4) {
		t.Fatalf("DATE_PART dow: got %v, want 4 (Thursday)", v)
	}
	if v := col(t, out, "dw2")[1]; v != int64(7) {
		t.Fatalf("DAYOFWEEK: got %v, want 7 (Sunday)", v)
	}
	if v := col(t, out, "dy")[0]; v != int64(32) {
		t.Fatalf("DAYOFYEAR: got %v", v)
	}
	if v := col(t, out, "od")[1]; v != int64(365) {
		t.Fatalf("ORDINAL_DAY: got %v", v)
	}
	if v := col(t, out, "mi")[0]; v != int64(45) {
		t.Fatalf("EXTRACT MINUTE: got %v", v)
	}
	if err := execSQLErr(t, df, "SELECT DATE_PART('femtosecond', d) FROM t"); err == nil || !strings.Contains(err.Error(), "femtosecond") {
		t.Fatalf("unsupported part: err = %v, want error naming the part", err)
	}
}

func TestSQLFnAggregates(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "g", Values: []any{"a", "a", "a", "b", "b"}},
		{Name: "x", Values: []any{2.0, 4.0, 6.0, 10.0, 10.0}},
		{Name: "s", Values: []any{"p", "q", "p", "r", nil}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out := runSQL(t, df, "SELECT g, STDDEV(x) AS sd, VARIANCE(x) AS vr, MEDIAN(x) AS md, FIRST(x) AS f, LAST(x) AS l, COUNT(DISTINCT s) AS cd FROM t GROUP BY g ORDER BY g")
	wantFloat(t, col(t, out, "sd")[0], 2, "STDDEV(a)")
	wantFloat(t, col(t, out, "vr")[0], 4, "VARIANCE(a)")
	wantFloat(t, col(t, out, "md")[0], 4, "MEDIAN(a)")
	wantFloat(t, col(t, out, "f")[0], 2, "FIRST(a)")
	wantFloat(t, col(t, out, "l")[0], 6, "LAST(a)")
	if v := col(t, out, "cd")[0]; v != int64(2) {
		t.Fatalf("COUNT(DISTINCT) group a: got %v, want 2", v)
	}
	// Group b: single distinct non-null value; STDDEV of equal values is 0.
	if v := col(t, out, "cd")[1]; v != int64(1) {
		t.Fatalf("COUNT(DISTINCT) group b: got %v, want 1 (nulls excluded)", v)
	}
	wantFloat(t, col(t, out, "sd")[1], 0, "STDDEV(b)")
	if err := execSQLErr(t, df, "SELECT SUM(DISTINCT x) FROM t"); err == nil || !strings.Contains(err.Error(), "DISTINCT") {
		t.Fatalf("SUM(DISTINCT): err = %v, want rejection", err)
	}
}

func TestSQLFnWindowRanks(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "dept", Values: []any{"a", "a", "a", "b"}},
		{Name: "salary", Values: []any{int64(100), int64(100), int64(80), int64(50)}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out := runSQL(t, df, "SELECT dept, salary, RANK() OVER (PARTITION BY dept ORDER BY salary DESC) AS r, DENSE_RANK() OVER (PARTITION BY dept ORDER BY salary DESC) AS dr FROM t ORDER BY dept, salary DESC")
	r := col(t, out, "r")
	dr := col(t, out, "dr")
	// dept a: 100,100,80 -> rank 1,1,3; dense_rank 1,1,2.
	if r[0] != int64(1) || r[1] != int64(1) || r[2] != int64(3) {
		t.Fatalf("RANK: got %v, want [1 1 3 ...]", r)
	}
	if dr[0] != int64(1) || dr[1] != int64(1) || dr[2] != int64(2) {
		t.Fatalf("DENSE_RANK: got %v, want [1 1 2 ...]", dr)
	}
	if r[3] != int64(1) {
		t.Fatalf("RANK dept b: got %v, want 1", r[3])
	}
}

func TestSQLFnWindowLagLead(t *testing.T) {
	t.Parallel()
	df, err := polars.NewDataFrame(polars.NewDataFrameInput{Columns: []frame.SeriesInput{
		{Name: "g", Values: []any{"a", "a", "a", "b"}},
		{Name: "ts", Values: []any{int64(1), int64(2), int64(3), int64(1)}},
		{Name: "x", Values: []any{10.0, 20.0, 30.0, 99.0}},
	}})
	if err != nil {
		t.Fatalf("df: %v", err)
	}
	out := runSQL(t, df, "SELECT g, ts, LAG(x) OVER (PARTITION BY g ORDER BY ts) AS lg, LEAD(x) OVER (PARTITION BY g ORDER BY ts) AS ld, LAG(x, 2, 0.0) OVER (PARTITION BY g ORDER BY ts) AS lg2, FIRST_VALUE(x) OVER (PARTITION BY g ORDER BY ts) AS fv, LAST_VALUE(x) OVER (PARTITION BY g ORDER BY ts) AS lv FROM t ORDER BY g, ts")
	lg := col(t, out, "lg")
	if lg[0] != nil {
		t.Fatalf("LAG first row: got %v, want nil", lg[0])
	}
	wantFloat(t, lg[1], 10, "LAG row 2")
	wantFloat(t, lg[2], 20, "LAG row 3")
	ld := col(t, out, "ld")
	wantFloat(t, ld[0], 20, "LEAD row 1")
	if ld[2] != nil {
		t.Fatalf("LEAD last row: got %v, want nil", ld[2])
	}
	lg2 := col(t, out, "lg2")
	wantFloat(t, lg2[0], 0, "LAG offset 2 default")
	wantFloat(t, lg2[2], 10, "LAG offset 2")
	fv := col(t, out, "fv")
	lv := col(t, out, "lv")
	wantFloat(t, fv[2], 10, "FIRST_VALUE")
	wantFloat(t, lv[0], 30, "LAST_VALUE (whole partition, Polars semantics)")
	wantFloat(t, fv[3], 99, "FIRST_VALUE dept b")
}

func TestSQLFnCatalogErrors(t *testing.T) {
	t.Parallel()
	if err := execSQLErr(t, baseDF(t), "SELECT FROBNICATE(a) FROM t"); err == nil || !strings.Contains(err.Error(), "FROBNICATE") {
		t.Fatalf("unknown fn: err = %v, want error naming FROBNICATE", err)
	}
	if err := execSQLErr(t, baseDF(t), "SELECT ATAN2(a) FROM t"); err == nil || !strings.Contains(err.Error(), "2 argument") {
		t.Fatalf("arity: err = %v, want two-argument error", err)
	}
	// Function names are case-insensitive.
	out := runSQL(t, baseDF(t), "SELECT ends_with(b, 'x') AS m FROM t LIMIT 1")
	if v := col(t, out, "m")[0]; v != true {
		t.Fatalf("case-insensitive lookup: got %v", v)
	}
}
