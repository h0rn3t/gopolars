## Top-30 Operations — gopolars vs Python Polars

> **Hardware:** Apple M4 Pro · Go 1.26 · Python Polars 1.41.2
> **Sizes:** 1 K rows and 1 M rows — min ns/op across calibration rounds

### DataFrame

| operation | size | Go time | Py time | speedup | Go B/op | Go allocs/op |
|-----------|------|---------|---------|---------|---------|--------------|
| `filter` | 1K | 9.6 µs | 94.2 µs | **Go ×9.8** | 28.3 KB | 25 |
| `filter` | 1M | 5.80 ms | 543.6 µs | Py ×10.7 | 25.1 MB | 112 |
| `select` | 1K | 291 ns | 48.6 µs | **Go ×167.2** | 864 B | 7 |
| `select` | 1M | 219 ns | 34.7 µs | **Go ×158.5** | 864 B | 7 |
| `with_columns` | 1K | 405 ns | 8.4 µs | **Go ×20.7** | 1.1 KB | 8 |
| `with_columns` | 1M | 308 ns | 8.3 µs | **Go ×26.9** | 1.1 KB | 8 |
| `sort` | 1K | 29.2 µs | 170.0 µs | **Go ×5.8** | 71.2 KB | 22 |
| `sort` | 1M | 35.94 ms | 12.72 ms | Py ×2.8 | 68.1 MB | 78 |
| `group_by` | 1K | 14.6 µs | 690.8 µs | **Go ×47.2** | 18.9 KB | 42 |
| `group_by` | 1M | 2.08 ms | 1.50 ms | Py ×1.4 | 92.8 KB | 187 |
| `join` | 1K | 150.9 µs | 319.2 µs | **Go ×2.1** | 311.1 KB | 1385 |
| `join` | 1M | 47.00 ms | 6.02 ms | Py ×7.8 | 122.5 MB | 2120 |
| `head` | 1K | 2.3 µs | 612 ns | Py ×3.7 | 7.2 KB | 19 |
| `head` | 1M | 1.6 µs | 812 ns | Py ×1.9 | 7.2 KB | 19 |
| `tail` | 1K | 2.3 µs | 758 ns | Py ×3.0 | 7.4 KB | 21 |
| `tail` | 1M | 1.6 µs | 833 ns | Py ×1.9 | 7.4 KB | 21 |
| `unique` | 1K | 10.2 µs | 198.7 µs | **Go ×19.5** | 9.9 KB | 24 |
| `unique` | 1M | 16.23 ms | 4.37 ms | Py ×3.7 | 7.6 MB | 24 |
| `fill_null` | 1K | 1.9 µs | 126.6 µs | **Go ×67.8** | 9.9 KB | 9 |
| `fill_null` | 1M | 1.15 ms | 902.2 µs | Py ×1.3 | 8.6 MB | 9 |
| `drop_nulls` | 1K | 49.6 µs | 59.7 µs | **Go ×1.2** | 53.6 KB | 20 |
| `drop_nulls` | 1M | 44.48 ms | 1.29 ms | Py ×34.4 | 49.6 MB | 20 |

### Expr

| operation | size | Go time | Py time | speedup | Go B/op | Go allocs/op |
|-----------|------|---------|---------|---------|---------|--------------|
| `cum_sum` | 1K | 1.9 µs | 12.9 µs | **Go ×6.7** | 10.0 KB | 10 |
| `cum_sum` | 1M | 774.6 µs | 2.92 ms | **Go ×3.8** | 8.6 MB | 10 |
| `rank` | 1K | 17.3 µs | 56.4 µs | **Go ×3.3** | 34.8 KB | 13 |
| `rank` | 1M | 22.74 ms | 16.24 ms | Py ×1.4 | 33.0 MB | 13 |
| `over` | 1K | 14.3 µs | 262.5 µs | **Go ×18.3** | 27.5 KB | 22 |
| `over` | 1M | 18.37 ms | 9.25 ms | Py ×2.0 | 24.8 MB | 22 |
| `fill_null` | 1K | 2.1 µs | 44.2 µs | **Go ×21.3** | 10.1 KB | 11 |
| `fill_null` | 1M | 1.20 ms | 689.8 µs | Py ×1.7 | 8.6 MB | 11 |
| `fill_nan` | 1K | 2.1 µs | 62.1 µs | **Go ×29.3** | 10.1 KB | 11 |
| `fill_nan` | 1M | 1.27 ms | 784.3 µs | Py ×1.6 | 8.6 MB | 11 |
| `rolling_mean` | 1K | 5.9 µs | 18.6 µs | **Go ×3.2** | 10.0 KB | 12 |
| `rolling_mean` | 1M | 4.79 ms | 8.86 ms | **Go ×1.8** | 8.6 MB | 12 |
| `rolling_sum` | 1K | 5.9 µs | 16.7 µs | **Go ×2.8** | 10.0 KB | 12 |
| `rolling_sum` | 1M | 4.77 ms | 8.78 ms | **Go ×1.8** | 8.6 MB | 12 |
| `rolling_min` | 1K | 4.3 µs | 15.0 µs | **Go ×3.5** | 10.8 KB | 13 |
| `rolling_min` | 1M | 8.07 ms | 12.06 ms | **Go ×1.5** | 8.9 MB | 24 |
| `rolling_max` | 1K | 4.5 µs | 16.7 µs | **Go ×3.7** | 10.8 KB | 13 |
| `rolling_max` | 1M | 8.17 ms | 12.36 ms | **Go ×1.5** | 8.9 MB | 24 |

### Series

| operation | size | Go time | Py time | speedup | Go B/op | Go allocs/op |
|-----------|------|---------|---------|---------|---------|--------------|
| `null_count` | 1K | 2 ns | 470 ns | **Go ×235.4** | — | 0 |
| `null_count` | 1M | 2 ns | 529 ns | **Go ×264.6** | — | 0 |
| `drop_nans` | 1K | 1.9 µs | 12.0 µs | **Go ×6.2** | 9.2 KB | 4 |
| `drop_nans` | 1M | 1.13 ms | 98.6 µs | Py ×11.4 | 8.6 MB | 4 |
| `to_list` | 1K | 9.4 µs | 10.2 µs | **Go ×1.1** | 23.8 KB | 1001 |
| `to_list` | 1M | 7.32 ms | 12.16 ms | **Go ×1.7** | 22.9 MB | 1000001 |
| `is_null` | 1K | 381 ns | 11.5 µs | **Go ×30.1** | 2.2 KB | 4 |
| `is_null` | 1M | 55.9 µs | 11.7 µs | Py ×4.8 | 1.9 MB | 4 |
| `is_not_null` | 1K | 659 ns | 11.5 µs | **Go ×17.4** | 2.2 KB | 4 |
| `is_not_null` | 1M | 379.7 µs | 12.8 µs | Py ×29.7 | 1.9 MB | 4 |
| `fill_nan` | 1K | 1.6 µs | 73.2 µs | **Go ×46.7** | 9.2 KB | 4 |
| `fill_nan` | 1M | 698.3 µs | 779.9 µs | **Go ×1.1** | 8.6 MB | 4 |

### LazyFrame

| operation | size | Go time | Py time | speedup | Go B/op | Go allocs/op |
|-----------|------|---------|---------|---------|---------|--------------|
| `collect` | 1K | 122 ns | 3.6 µs | **Go ×29.6** | 304 B | 4 |
| `collect` | 1M | 97 ns | 4.4 µs | **Go ×45.2** | 304 B | 4 |
| `sql` | 1K | 26.6 µs | 11.4 µs | Py ×2.3 | 36.9 KB | 76 |
| `sql` | 1M | 6.55 ms | 11.4 µs | Py ×575.1 | 25.1 MB | 163 |
| `inspect` | 1K | 27 ns | 1.1 µs | **Go ×41.8** | 96 B | 1 |
| `inspect` | 1M | 18 ns | 1.4 µs | **Go ×75.9** | 96 B | 1 |

### SQLContext

| operation | size | Go time | Py time | speedup | Go B/op | Go allocs/op |
|-----------|------|---------|---------|---------|---------|--------------|
| `execute` | 1K | 1.7 µs | 7.2 µs | **Go ×4.3** | 3.7 KB | 31 |
| `execute` | 1M | 1.4 µs | 6.6 µs | **Go ×4.8** | 3.7 KB | 31 |
| `register` | 1K | 170 ns | 2.2 µs | **Go ×13.0** | 800 B | 3 |
| `register` | 1M | 158 ns | 2.3 µs | **Go ×14.4** | 800 B | 3 |
| `tables` | 1K | 246 ns | 1.8 µs | **Go ×7.4** | 816 B | 4 |
| `tables` | 1M | 282 ns | 1.9 µs | **Go ×6.6** | 816 B | 4 |


## Filter + Sum Pipeline — Detailed Comparison

> Three execution paths vs Python Polars eager and lazy across two selectivity profiles.

> **Go B/op** = heap bytes allocated per operation (Go runtime).  
> **Py peak RSS** = peak resident-set-size growth Polars drove for the operation.

### 0% selectivity (threshold=50, no rows pass)

| engine | size | Go time | Py time | speedup | Go B/op | Go allocs | Py peak RSS |
|--------|------|---------|---------|---------|---------|-----------|-------------|
| Eager (filter then sum) | 1K | 86.6 µs | 87.5 µs | **Go ×1.0** | 1.2 KB | 11 | 8.2 MB |
| Lazy fused (filter+sum single pass) | 1K | 201.2 µs | 95.3 µs | Py ×2.1 | 6.8 KB | 42 | 8.8 MB |
| Eager-direct (fused, no plan) | 1K | 7.2 µs | 76.9 µs | **Go ×10.7** | 816 B | 8 | 8.4 MB |
| Eager (filter then sum) | 10K | 38.0 µs | 70.6 µs | **Go ×1.9** | 2.3 KB | 11 | 8.2 MB |
| Lazy fused (filter+sum single pass) | 10K | 42.0 µs | 93.4 µs | **Go ×2.2** | 7.3 KB | 40 | 9.0 MB |
| Eager-direct (fused, no plan) | 10K | 42.0 µs | 67.2 µs | **Go ×1.6** | 1.9 KB | 8 | 8.3 MB |
| Eager (filter then sum) | 100K | 263.5 µs | 119.3 µs | Py ×2.2 | 37.1 KB | 73 | 8.9 MB |
| Lazy fused (filter+sum single pass) | 100K | 130.5 µs | 111.8 µs | Py ×1.2 | 29.9 KB | 115 | 9.6 MB |
| Eager-direct (fused, no plan) | 100K | 94.4 µs | 100.5 µs | **Go ×1.1** | 22.4 KB | 77 | 9.2 MB |
| Eager (filter then sum) | 1M | 508.8 µs | 218.3 µs | Py ×2.3 | 266.6 KB | 73 | 16.0 MB |
| Lazy fused (filter+sum single pass) | 1M | 338.9 µs | 212.9 µs | Py ×1.6 | 143.0 KB | 113 | 16.6 MB |
| Eager-direct (fused, no plan) | 1M | 244.6 µs | 236.5 µs | Py ×1.0 | 134.2 KB | 73 | 16.0 MB |
| Eager (filter then sum) | 10M | 3.17 ms | 1.29 ms | Py ×2.5 | 2.4 MB | 72 | 85.8 MB |
| Lazy fused (filter+sum single pass) | 10M | 1.87 ms | 1.31 ms | Py ×1.4 | 1.2 MB | 117 | 86.3 MB |
| Eager-direct (fused, no plan) | 10M | 1.61 ms | 1.33 ms | Py ×1.2 | 1.2 MB | 80 | 85.7 MB |

### 50% selectivity (threshold=0, half rows pass)

| engine | size | Go time | Py time | speedup | Go B/op | Go allocs | Py peak RSS |
|--------|------|---------|---------|---------|---------|-----------|-------------|
| Eager (filter then sum) | 1K | 26.4 µs | 92.9 µs | **Go ×3.5** | 16.0 KB | 15 | 8.3 MB |
| Lazy fused (filter+sum single pass) | 1K | 39.7 µs | 94.4 µs | **Go ×2.4** | 6.3 KB | 42 | 9.0 MB |
| Eager-direct (fused, no plan) | 1K | 13.4 µs | 71.2 µs | **Go ×5.3** | 816 B | 8 | 8.3 MB |
| Eager (filter then sum) | 10K | 88.4 µs | 71.4 µs | Py ×1.2 | 127.6 KB | 15 | 8.4 MB |
| Lazy fused (filter+sum single pass) | 10K | 86.0 µs | 101.9 µs | **Go ×1.2** | 7.3 KB | 41 | 9.0 MB |
| Eager-direct (fused, no plan) | 10K | 72.3 µs | 75.1 µs | **Go ×1.0** | 1.9 KB | 8 | 8.5 MB |
| Eager (filter then sum) | 100K | 471.0 µs | 101.3 µs | Py ×4.6 | 1.2 MB | 74 | 9.5 MB |
| Lazy fused (filter+sum single pass) | 100K | 192.5 µs | 84.1 µs | Py ×2.3 | 24.5 KB | 103 | 10.1 MB |
| Eager-direct (fused, no plan) | 100K | 179.4 µs | 89.0 µs | Py ×2.0 | 22.9 KB | 78 | 9.5 MB |
| Eager (filter then sum) | 1M | 3.75 ms | 391.1 µs | Py ×9.6 | 12.2 MB | 77 | 19.9 MB |
| Lazy fused (filter+sum single pass) | 1M | 913.0 µs | 602.4 µs | Py ×1.5 | 141.4 KB | 110 | 25.4 MB |
| Eager-direct (fused, no plan) | 1M | 661.5 µs | 410.1 µs | Py ×1.6 | 133.1 KB | 70 | 19.9 MB |
| Eager (filter then sum) | 10M | 32.30 ms | 8.18 ms | Py ×3.9 | 121.6 MB | 77 | 124.1 MB |
| Lazy fused (filter+sum single pass) | 10M | 6.09 ms | 6.78 ms | **Go ×1.1** | 1.2 MB | 114 | 125.7 MB |
| Eager-direct (fused, no plan) | 10M | 7.94 ms | 9.08 ms | **Go ×1.1** | 1.2 MB | 72 | 124.1 MB |
