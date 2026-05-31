package expr

import "testing"

func TestExprSurfaceBuilders(t *testing.T) {
	t.Parallel()

	v := Col("x")
	by := Col("g")

	_ = v.RollingMeanBy(by, 2)
	_ = v.RollingMinBy(by, 2)
	_ = v.RollingSumBy(by, 2)
	_ = v.RollingStdBy(by, 2)
	_ = v.RollingVarBy(by, 2)
	_ = v.RollingMedian(2)
	_ = v.RollingMedianBy(by, 2)
	_ = v.RollingQuantile(2, 0.5)
	_ = v.RollingQuantileBy(by, 2, 0.5)
	_ = v.RollingSkew(2)
	_ = v.RollingKurtosis(2)
	_ = v.RollingMap(2)
	_ = v.Rolling(2)
	_ = v.RollingRank(2)
	_ = v.RollingRankBy(by, 2)
	_ = v.Sort(true)
	_ = v.SortBy(by, true)
	_ = v.UniqueCounts()
	_ = v.Rechunk()
	_ = v.Reinterpret()
	_ = v.RepeatBy(2)
	_ = v.ReplaceStrict(Lit("a"), Lit("b"))
	_ = v.Reshape(2, 2)
	_ = v.Rle()
	_ = v.RleId()
	_ = v.Sample(1, 42)
	_ = v.SearchSorted(Lit(int64(1)))
	_ = v.SetSorted(true)
	_ = v.Shift(1)
	_ = v.ShrinkDtype()
	_ = v.Shuffle(7)
	_ = v.Truncate()
}
