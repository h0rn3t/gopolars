# Top30 Benchmark Results

## DataFrame

| operation | size | go_ns/op | python_ms/op | ratio |
|-----------|------|----------|--------------|-------|
| fill_null | 1M | 2111667 | 0.504958 | 0.2391 |
| fill_null | 1M | 457472 | 0.400479 | 0.8754 |
| fill_null | 1M | 387462 | 0.367317 | 0.9480 |

## Expr

| operation | size | go_ns/op | python_ms/op | ratio |
|-----------|------|----------|--------------|-------|
| fill_null | 1M | 1437417 | 0.231767 | 0.1612 |
| fill_null | 1M | 473975 | 0.269837 | 0.5693 |
| fill_null | 1M | 399251 | 0.233825 | 0.5857 |
| fill_nan | 1M | 165667 | 0.893225 | 5.3917 |
| fill_nan | 1M | 88036 | 0.874696 | 9.9357 |
| fill_nan | 1M | 80830 | 0.895179 | 11.0748 |
| fill_nan | 1M | 78115 | 0.915583 | 11.7210 |

## Series

| operation | size | go_ns/op | python_ms/op | ratio |
|-----------|------|----------|--------------|-------|

## LazyFrame

| operation | size | go_ns/op | python_ms/op | ratio |
|-----------|------|----------|--------------|-------|

## SQLContext

| operation | size | go_ns/op | python_ms/op | ratio |
|-----------|------|----------|--------------|-------|

