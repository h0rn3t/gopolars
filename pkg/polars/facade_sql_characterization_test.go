//go:build duckdb && duckdb_arrow

package polars

import (
	"context"
	"reflect"
	"testing"
)

func TestLazySQLCharacterization(t *testing.T) {
	ctx := context.Background()
	res, err := sqlFrame(t).Lazy().Filter(Col("v").Gt(Lit(int64(10)))).SQL(ctx, "SELECT id FROM t WHERE g = 'b' ORDER BY id", "t")
	out := collect(t, res, err)
	if got, want := out.ToDict(), map[string][]any{"id": {int64(3), int64(4)}}; !reflect.DeepEqual(got, want) {
		t.Errorf("LazyFrame.SQL = %v, want %v", got, want)
	}
	if _, err := sqlFrame(t).Lazy().Select(Col("missing")).SQL(ctx, "SELECT * FROM t", "t"); err == nil {
		t.Errorf("LazyFrame.SQL on a failing plan: want error")
	}
}
