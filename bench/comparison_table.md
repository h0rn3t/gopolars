## Top-30 Operations — gopolars vs Python Polars

> **Hardware:** Apple M4 Pro · Go 1.26 · Python Polars 1.40.1
> **Sizes:** 1 K rows and 1 M rows — min ns/op across calibration rounds

### DataFrame

| operation | size | Go time | Py time | speedup |
|-----------|------|---------|---------|---------|
| `filter` | 1K | 5.3 µs | 65.1 µs | **Go ×12.3** |
| `filter` | 1M | 2.18 ms | 576.4 µs | Py ×3.8 |
| `select` | 1K | 493 ns | 51.4 µs | **Go ×104.2** |
| `select` | 1M | 459 ns | 31.8 µs | **Go ×69.3** |
| `with_columns` | 1K | 488 ns | 8.1 µs | **Go ×16.5** |
| `with_columns` | 1M | 433 ns | 8.0 µs | **Go ×18.5** |
| `sort` | 1K | 21.0 µs | 159.5 µs | **Go ×7.6** |
| `sort` | 1M | 18.41 ms | 11.36 ms | Py ×1.6 |
| `group_by` | 1K | 13.8 µs | 584.5 µs | **Go ×42.3** |
| `group_by` | 1M | 1.52 ms | 1.67 ms | **Go ×1.1** |
| `join` | 1K | 88.0 µs | 362.0 µs | **Go ×4.1** |
| `join` | 1M | 25.89 ms | 5.47 ms | Py ×4.7 |
| `head` | 1K | 1.2 µs | 562 ns | Py ×2.2 |
| `head` | 1M | 899 ns | 795 ns | Py ×1.1 |
| `tail` | 1K | 1.4 µs | 579 ns | Py ×2.4 |
| `tail` | 1M | 1.0 µs | 587 ns | Py ×1.7 |
| `unique` | 1K | 9.4 µs | 162.0 µs | **Go ×17.3** |
| `unique` | 1M | 16.13 ms | 5.16 ms | Py ×3.1 |
| `fill_null` | 1K | 1.9 µs | 144.0 µs | **Go ×76.9** |
| `fill_null` | 1M | 1.20 ms | 314.0 µs | Py ×3.8 |
| `drop_nulls` | 1K | 734 ns | 88.4 µs | **Go ×120.4** |
| `drop_nulls` | 1M | 925.1 µs | 1.57 ms | **Go ×1.7** |

### Expr

| operation | size | Go time | Py time | speedup |
|-----------|------|---------|---------|---------|
| `cum_sum` | 1K | 2.2 µs | 12.0 µs | **Go ×5.5** |
| `cum_sum` | 1M | 885.0 µs | 2.33 ms | **Go ×2.6** |
| `rank` | 1K | 16.3 µs | 55.2 µs | **Go ×3.4** |
| `rank` | 1M | 19.94 ms | 14.78 ms | Py ×1.3 |
| `over` | 1K | 13.7 µs | 366.3 µs | **Go ×26.8** |
| `over` | 1M | 18.11 ms | 6.92 ms | Py ×2.6 |
| `fill_null` | 1K | 2.3 µs | 48.5 µs | **Go ×21.2** |
| `fill_null` | 1M | 1.12 ms | 213.1 µs | Py ×5.3 |
| `fill_nan` | 1K | 2.3 µs | 76.1 µs | **Go ×33.0** |
| `fill_nan` | 1M | 1.27 ms | 375.7 µs | Py ×3.4 |
| `rolling_mean` | 1K | 5.9 µs | 17.9 µs | **Go ×3.0** |
| `rolling_mean` | 1M | 5.15 ms | 8.54 ms | **Go ×1.7** |
| `rolling_sum` | 1K | 5.9 µs | 17.8 µs | **Go ×3.0** |
| `rolling_sum` | 1M | 4.89 ms | 8.30 ms | **Go ×1.7** |
| `rolling_min` | 1K | 4.3 µs | 14.5 µs | **Go ×3.4** |
| `rolling_min` | 1M | 8.03 ms | 11.77 ms | **Go ×1.5** |
| `rolling_max` | 1K | 4.4 µs | 14.7 µs | **Go ×3.3** |
| `rolling_max` | 1M | 8.16 ms | 12.05 ms | **Go ×1.5** |

### Series

| operation | size | Go time | Py time | speedup |
|-----------|------|---------|---------|---------|
| `null_count` | 1K | 2 ns | 449 ns | **Go ×225.0** |
| `null_count` | 1M | 2 ns | 466 ns | **Go ×233.3** |
| `drop_nans` | 1K | 1.9 µs | 11.3 µs | **Go ×5.9** |
| `drop_nans` | 1M | 1.14 ms | 102.4 µs | Py ×11.1 |
| `to_list` | 1K | 8.4 µs | 10.1 µs | **Go ×1.2** |
| `to_list` | 1M | 7.43 ms | 14.19 ms | **Go ×1.9** |
| `is_null` | 1K | 330 ns | 11.1 µs | **Go ×33.6** |
| `is_null` | 1M | 43.3 µs | 10.8 µs | Py ×4.0 |
| `is_not_null` | 1K | 332 ns | 10.8 µs | **Go ×32.6** |
| `is_not_null` | 1M | 59.2 µs | 12.3 µs | Py ×4.8 |
| `fill_nan` | 1K | 1.5 µs | 69.1 µs | **Go ×46.0** |
| `fill_nan` | 1M | 657.8 µs | 301.7 µs | Py ×2.2 |

### LazyFrame

| operation | size | Go time | Py time | speedup |
|-----------|------|---------|---------|---------|
| `collect` | 1K | 119 ns | 3.4 µs | **Go ×29.0** |
| `collect` | 1M | 112 ns | 3.7 µs | **Go ×33.3** |
| `sql` | 1K | 24.5 µs | 11.3 µs | Py ×2.2 |
| `sql` | 1M | 2.77 ms | 11.0 µs | Py ×251.6 |
| `inspect` | 1K | 23 ns | 991 ns | **Go ×43.1** |
| `inspect` | 1M | 19 ns | 1.7 µs | **Go ×90.1** |

### SQLContext

| operation | size | Go time | Py time | speedup |
|-----------|------|---------|---------|---------|
| `execute` | 1K | 2.1 µs | 6.2 µs | **Go ×2.9** |
| `execute` | 1M | 2.0 µs | 6.4 µs | **Go ×3.2** |
| `register` | 1K | 180 ns | 2.2 µs | **Go ×12.2** |
| `register` | 1M | 167 ns | 2.1 µs | **Go ×12.8** |
| `tables` | 1K | 227 ns | 1.8 µs | **Go ×7.7** |
| `tables` | 1M | 273 ns | 1.8 µs | **Go ×6.5** |


## Filter + Sum Pipeline — Detailed Comparison

> Three execution paths vs Python Polars eager and lazy across two selectivity profiles.

> **Go B/op** = heap bytes allocated per operation (Go runtime).  
> **Py peak RSS** = peak resident-set-size growth Polars drove for the operation.

### 0% selectivity (threshold=50, no rows pass)

| engine | size | Go time | Py time | speedup | Go B/op | Go allocs | Py peak RSS |
|--------|------|---------|---------|---------|---------|-----------|-------------|
| Eager (filter then sum) | 1K | 1.6 µs | 69.6 µs | **Go ×44.8** | 1.4 KB | 12 | 8.2 MB |
| Lazy fused (filter+sum single pass) | 1K | 7.2 µs | 451.9 µs | **Go ×62.5** | 6.4 KB | 40 | 9.0 MB |
| Eager-direct (fused, no plan) | 1K | 1.4 µs | 73.0 µs | **Go ×53.2** | 848 B | 8 | 8.3 MB |
| Eager (filter then sum) | 10K | 10.9 µs | 58.6 µs | **Go ×5.4** | 2.6 KB | 12 | 8.3 MB |
| Lazy fused (filter+sum single pass) | 10K | 17.9 µs | 84.3 µs | **Go ×4.7** | 7.5 KB | 40 | 9.0 MB |
| Eager-direct (fused, no plan) | 10K | 10.7 µs | 79.2 µs | **Go ×7.4** | 2.0 KB | 8 | 8.2 MB |
| Eager (filter then sum) | 100K | 53.0 µs | 97.4 µs | **Go ×1.8** | 32.4 KB | 63 | 9.0 MB |
| Lazy fused (filter+sum single pass) | 100K | 54.7 µs | 87.3 µs | **Go ×1.6** | 24.9 KB | 102 | 9.6 MB |
| Eager-direct (fused, no plan) | 100K | 49.2 µs | 71.1 µs | **Go ×1.4** | 19.3 KB | 70 | 9.0 MB |
| Eager (filter then sum) | 1M | 215.6 µs | 179.6 µs | Py ×1.2 | 261.2 KB | 63 | 16.0 MB |
| Lazy fused (filter+sum single pass) | 1M | 218.9 µs | 192.1 µs | Py ×1.1 | 138.9 KB | 102 | 16.6 MB |
| Eager-direct (fused, no plan) | 1M | 203.2 µs | 173.2 µs | Py ×1.2 | 133.3 KB | 70 | 16.0 MB |
| Eager (filter then sum) | 10M | 1.68 ms | 1.26 ms | Py ×1.3 | 2.4 MB | 63 | 85.7 MB |
| Lazy fused (filter+sum single pass) | 10M | 1.50 ms | 1.31 ms | Py ×1.1 | 1.2 MB | 102 | 86.4 MB |
| Eager-direct (fused, no plan) | 10M | 1.51 ms | 1.68 ms | **Go ×1.1** | 1.2 MB | 70 | 85.7 MB |

### 50% selectivity (threshold=0, half rows pass)

| engine | size | Go time | Py time | speedup | Go B/op | Go allocs | Py peak RSS |
|--------|------|---------|---------|---------|---------|-----------|-------------|
| Eager (filter then sum) | 1K | 4.0 µs | 94.6 µs | **Go ×23.9** | 15.7 KB | 15 | 8.3 MB |
| Lazy fused (filter+sum single pass) | 1K | 8.7 µs | 68.5 µs | **Go ×7.8** | 6.4 KB | 41 | 9.0 MB |
| Eager-direct (fused, no plan) | 1K | 2.6 µs | 100.0 µs | **Go ×38.8** | 848 B | 8 | 8.4 MB |
| Eager (filter then sum) | 10K | 33.4 µs | 65.7 µs | **Go ×2.0** | 122.6 KB | 15 | 8.4 MB |
| Lazy fused (filter+sum single pass) | 10K | 30.1 µs | 66.0 µs | **Go ×2.2** | 7.6 KB | 41 | 9.0 MB |
| Eager-direct (fused, no plan) | 10K | 22.4 µs | 95.7 µs | **Go ×4.3** | 2.0 KB | 8 | 8.4 MB |
| Eager (filter then sum) | 100K | 299.7 µs | 81.5 µs | Py ×3.7 | 1.2 MB | 66 | 9.4 MB |
| Lazy fused (filter+sum single pass) | 100K | 101.9 µs | 121.1 µs | **Go ×1.2** | 24.9 KB | 103 | 10.1 MB |
| Eager-direct (fused, no plan) | 100K | 93.3 µs | 80.3 µs | Py ×1.2 | 19.3 KB | 70 | 9.4 MB |
| Eager (filter then sum) | 1M | 1.64 ms | 401.2 µs | Py ×4.1 | 11.7 MB | 66 | 19.9 MB |
| Lazy fused (filter+sum single pass) | 1M | 690.3 µs | 549.6 µs | Py ×1.3 | 138.9 KB | 103 | 21.5 MB |
| Eager-direct (fused, no plan) | 1M | 675.5 µs | 455.3 µs | Py ×1.5 | 133.3 KB | 70 | 19.9 MB |
| Eager (filter then sum) | 10M | 15.30 ms | 7.37 ms | Py ×2.1 | 116.9 MB | 66 | 124.1 MB |
| Lazy fused (filter+sum single pass) | 10M | 6.07 ms | 6.20 ms | **Go ×1.0** | 1.2 MB | 103 | 125.7 MB |
| Eager-direct (fused, no plan) | 10M | 6.98 ms | 5.92 ms | Py ×1.2 | 1.2 MB | 70 | 124.1 MB |

