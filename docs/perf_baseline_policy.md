## Benchmark Baseline Policy

### Command profile

- `go test ./bench/... -run '^$' -bench . -benchmem -count=3 -benchtime=3s`

### Regression budget

- `ns/op`: fail on regression above 10%
- `allocs/op`: fail on regression above 15%
- `B/op`: fail on regression above 15%

### Baseline lifecycle

- Baseline is refreshed after intentional performance-affecting merges.
- Refresh requires benchmark report and reviewer sign-off.
- Release workflow uses the same benchmark command profile as CI.
