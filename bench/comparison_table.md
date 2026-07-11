## Top-30 Operations — gopolars vs Python Polars

> **Hardware:** Apple M4 Pro · Go 1.26 · Python Polars 1.41.2
> **Sizes:** 1 K rows and 1 M rows — min ns/op across calibration rounds

### DataFrame

| operation | size | Go time | Py time | speedup | Go B/op | Go allocs/op |
|-----------|------|---------|---------|---------|---------|--------------|
| `filter` | 1K | 6.0 µs | 94.4 µs | **Go ×15.6** | 27.1 KB | 23 |
| `filter` | 1M | 1.54 ms | 635.3 µs | Py ×2.4 | 23.7 MB | 214 |
| `select` | 1K | 601 ns | 42.3 µs | **Go ×70.4** | 1.5 KB | 10 |
| `select` | 1M | 475 ns | 48.2 µs | **Go ×101.4** | 1.5 KB | 10 |
| `with_columns` | 1K | 584 ns | 9.0 µs | **Go ×15.5** | 1.5 KB | 10 |
| `with_columns` | 1M | 470 ns | 8.8 µs | **Go ×18.7** | 1.5 KB | 10 |
| `sort` | 1K | 23.4 µs | 173.4 µs | **Go ×7.4** | 66.6 KB | 19 |
| `sort` | 1M | 17.42 ms | 14.03 ms | Py ×1.2 | 62.0 MB | 179 |
| `group_by` | 1K | 14.9 µs | 683.1 µs | **Go ×46.0** | 18.6 KB | 41 |
| `group_by` | 1M | 1.83 ms | 1.58 ms | Py ×1.2 | 90.7 KB | 186 |
| `join` | 1K | 88.5 µs | 315.3 µs | **Go ×3.6** | 225.4 KB | 727 |
| `join` | 1M | 11.85 ms | 6.71 ms | Py ×1.8 | 101.4 MB | 1164 |
| `head` | 1K | 464 ns | 633 ns | **Go ×1.4** | 1.6 KB | 10 |
| `head` | 1M | 329 ns | 933 ns | **Go ×2.8** | 1.6 KB | 10 |
| `tail` | 1K | 1.8 µs | 600 ns | Py ×3.0 | 7.0 KB | 18 |
| `tail` | 1M | 1.0 µs | 662 ns | Py ×1.6 | 7.0 KB | 18 |
| `unique` | 1K | 10.6 µs | 129.0 µs | **Go ×12.2** | 9.9 KB | 21 |
| `unique` | 1M | 16.29 ms | 4.45 ms | Py ×3.7 | 7.6 MB | 21 |
| `fill_null` | 1K | 2.2 µs | 146.7 µs | **Go ×65.4** | 10.0 KB | 10 |
| `fill_null` | 1M | 425.3 µs | 1.29 ms | **Go ×3.0** | 8.6 MB | 35 |
| `drop_nulls` | 1K | 10.4 µs | 94.4 µs | **Go ×9.1** | 50.6 KB | 17 |
| `drop_nulls` | 1M | 2.59 ms | 1.64 ms | Py ×1.6 | 42.1 MB | 121 |

### Expr

| operation | size | Go time | Py time | speedup | Go B/op | Go allocs/op |
|-----------|------|---------|---------|---------|---------|--------------|
| `cum_sum` | 1K | 2.6 µs | 13.2 µs | **Go ×5.1** | 10.7 KB | 14 |
| `cum_sum` | 1M | 782.0 µs | 3.18 ms | **Go ×4.1** | 8.6 MB | 14 |
| `rank` | 1K | 17.2 µs | 49.2 µs | **Go ×2.9** | 34.7 KB | 17 |
| `rank` | 1M | 20.71 ms | 14.85 ms | Py ×1.4 | 31.5 MB | 17 |
| `over` | 1K | 16.0 µs | 289.6 µs | **Go ×18.1** | 28.2 KB | 26 |
| `over` | 1M | 18.21 ms | 10.86 ms | Py ×1.7 | 24.8 MB | 26 |
| `fill_null` | 1K | 3.0 µs | 39.0 µs | **Go ×13.1** | 10.9 KB | 16 |
| `fill_null` | 1M | 388.4 µs | 759.3 µs | **Go ×2.0** | 8.6 MB | 41 |
| `fill_nan` | 1K | 1.2 µs | 79.4 µs | **Go ×66.4** | 1.9 KB | 15 |
| `fill_nan` | 1M | 75.7 µs | 894.3 µs | **Go ×11.8** | 2.6 KB | 40 |
| `rolling_mean` | 1K | 6.5 µs | 18.5 µs | **Go ×2.8** | 10.7 KB | 16 |
| `rolling_mean` | 1M | 4.77 ms | 9.23 ms | **Go ×1.9** | 8.6 MB | 16 |
| `rolling_sum` | 1K | 6.7 µs | 18.1 µs | **Go ×2.7** | 10.7 KB | 16 |
| `rolling_sum` | 1M | 4.74 ms | 9.09 ms | **Go ×1.9** | 8.6 MB | 16 |
| `rolling_min` | 1K | 4.8 µs | 15.5 µs | **Go ×3.2** | 11.5 KB | 17 |
| `rolling_min` | 1M | 8.17 ms | 12.24 ms | **Go ×1.5** | 8.9 MB | 28 |
| `rolling_max` | 1K | 5.1 µs | 15.3 µs | **Go ×3.0** | 11.5 KB | 17 |
| `rolling_max` | 1M | 8.06 ms | 12.26 ms | **Go ×1.5** | 8.9 MB | 28 |

### Series

| operation | size | Go time | Py time | speedup | Go B/op | Go allocs/op |
|-----------|------|---------|---------|---------|---------|--------------|
| `null_count` | 1K | 2 ns | 475 ns | **Go ×237.5** | — | 0 |
| `null_count` | 1M | 2 ns | 662 ns | **Go ×331.3** | — | 0 |
| `drop_nans` | 1K | 410 ns | 12.4 µs | **Go ×30.2** | 300 B | 4 |
| `drop_nans` | 1M | 85.5 µs | 100.8 µs | **Go ×1.2** | 988 B | 29 |
| `to_list` | 1K | 9.5 µs | 9.8 µs | **Go ×1.0** | 23.8 KB | 1001 |
| `to_list` | 1M | 7.36 ms | 14.14 ms | **Go ×1.9** | 22.9 MB | 1000001 |
| `is_null` | 1K | 395 ns | 11.5 µs | **Go ×29.2** | 2.2 KB | 4 |
| `is_null` | 1M | 42.0 µs | 11.8 µs | Py ×3.6 | 1.9 MB | 4 |
| `is_not_null` | 1K | 391 ns | 11.7 µs | **Go ×30.0** | 2.2 KB | 4 |
| `is_not_null` | 1M | 53.9 µs | 13.8 µs | Py ×3.9 | 1.9 MB | 4 |
| `fill_nan` | 1K | 396 ns | 84.3 µs | **Go ×213.0** | 300 B | 4 |
| `fill_nan` | 1M | 74.3 µs | 793.3 µs | **Go ×10.7** | 988 B | 29 |

### LazyFrame

| operation | size | Go time | Py time | speedup | Go B/op | Go allocs/op |
|-----------|------|---------|---------|---------|---------|--------------|
| `collect` | 1K | 83 ns | 4.1 µs | **Go ×49.2** | 192 B | 2 |
| `collect` | 1M | 70 ns | 4.1 µs | **Go ×58.9** | 192 B | 2 |
| `inspect` | 1K | 31 ns | 1.1 µs | **Go ×37.0** | 112 B | 1 |
| `inspect` | 1M | 21 ns | 1.1 µs | **Go ×53.0** | 112 B | 1 |

### SQLContext

| operation | size | Go time | Py time | speedup | Go B/op | Go allocs/op |
|-----------|------|---------|---------|---------|---------|--------------|


## Filter + Sum Pipeline — Detailed Comparison

> Three execution paths vs Python Polars eager and lazy across two selectivity profiles.

> **Go B/op** = heap bytes allocated per operation (Go runtime).  
> **Py peak RSS** = peak resident-set-size growth Polars drove for the operation.

### 0% selectivity (threshold=50, no rows pass)

| engine | size | Go time | Py time | speedup | Go B/op | Go allocs | Py peak RSS |
|--------|------|---------|---------|---------|---------|-----------|-------------|
| Eager (filter then sum) | 1K | 1.5 µs | 230.5 µs | **Go ×148.9** | 1.5 KB | 13 | 8.2 MB |
| Lazy fused (filter+sum single pass) | 1K | 2.3 µs | 88.8 µs | **Go ×38.0** | 5.6 KB | 31 | 8.8 MB |
| Eager-direct (fused, no plan) | 1K | 1.1 µs | 124.6 µs | **Go ×110.5** | 720 B | 7 | 8.3 MB |
| Eager (filter then sum) | 10K | 9.8 µs | 63.0 µs | **Go ×6.4** | 2.6 KB | 13 | 8.2 MB |
| Lazy fused (filter+sum single pass) | 10K | 9.4 µs | 82.1 µs | **Go ×8.8** | 5.6 KB | 31 | 8.9 MB |
| Eager-direct (fused, no plan) | 10K | 8.0 µs | 88.8 µs | **Go ×11.1** | 720 B | 7 | 8.3 MB |
| Eager (filter then sum) | 100K | 49.7 µs | 79.2 µs | **Go ×1.6** | 32.4 KB | 64 | 8.9 MB |
| Lazy fused (filter+sum single pass) | 100K | 40.5 µs | 71.1 µs | **Go ×1.8** | 8.2 KB | 58 | 9.6 MB |
| Eager-direct (fused, no plan) | 100K | 35.3 µs | 96.5 µs | **Go ×2.7** | 3.3 KB | 34 | 9.0 MB |
| Eager (filter then sum) | 1M | 239.3 µs | 287.2 µs | **Go ×1.2** | 261.2 KB | 64 | 16.0 MB |
| Lazy fused (filter+sum single pass) | 1M | 173.4 µs | 212.4 µs | **Go ×1.2** | 8.2 KB | 58 | 16.6 MB |
| Eager-direct (fused, no plan) | 1M | 172.1 µs | 203.6 µs | **Go ×1.2** | 3.3 KB | 34 | 16.0 MB |
| Eager (filter then sum) | 10M | 1.60 ms | 1.25 ms | Py ×1.3 | 2.4 MB | 64 | 85.7 MB |
| Lazy fused (filter+sum single pass) | 10M | 1.42 ms | 1.25 ms | Py ×1.1 | 8.2 KB | 58 | 86.3 MB |
| Eager-direct (fused, no plan) | 10M | 1.42 ms | 1.28 ms | Py ×1.1 | 3.3 KB | 34 | 85.8 MB |

### 50% selectivity (threshold=0, half rows pass)

| engine | size | Go time | Py time | speedup | Go B/op | Go allocs | Py peak RSS |
|--------|------|---------|---------|---------|---------|-----------|-------------|
| Eager (filter then sum) | 1K | 4.4 µs | 112.2 µs | **Go ×25.5** | 15.7 KB | 16 | 8.3 MB |
| Lazy fused (filter+sum single pass) | 1K | 2.6 µs | 97.7 µs | **Go ×37.7** | 5.6 KB | 32 | 8.9 MB |
| Eager-direct (fused, no plan) | 1K | 1.1 µs | 84.4 µs | **Go ×74.2** | 720 B | 7 | 8.3 MB |
| Eager (filter then sum) | 10K | 30.6 µs | 73.9 µs | **Go ×2.4** | 122.6 KB | 16 | 8.4 MB |
| Lazy fused (filter+sum single pass) | 10K | 9.9 µs | 101.2 µs | **Go ×10.2** | 5.6 KB | 32 | 9.0 MB |
| Eager-direct (fused, no plan) | 10K | 8.1 µs | 90.7 µs | **Go ×11.2** | 720 B | 7 | 8.4 MB |
| Eager (filter then sum) | 100K | 227.2 µs | 99.8 µs | Py ×2.3 | 1.2 MB | 67 | 9.4 MB |
| Lazy fused (filter+sum single pass) | 100K | 43.5 µs | 107.8 µs | **Go ×2.5** | 8.2 KB | 59 | 10.0 MB |
| Eager-direct (fused, no plan) | 100K | 36.2 µs | 92.1 µs | **Go ×2.5** | 3.3 KB | 34 | 9.4 MB |
| Eager (filter then sum) | 1M | 1.38 ms | 399.2 µs | Py ×3.4 | 11.7 MB | 67 | 19.9 MB |
| Lazy fused (filter+sum single pass) | 1M | 182.8 µs | 518.9 µs | **Go ×2.8** | 8.2 KB | 59 | 21.5 MB |
| Eager-direct (fused, no plan) | 1M | 172.1 µs | 394.4 µs | **Go ×2.3** | 3.3 KB | 34 | 19.9 MB |
| Eager (filter then sum) | 10M | 12.99 ms | 7.04 ms | Py ×1.8 | 116.9 MB | 67 | 124.1 MB |
| Lazy fused (filter+sum single pass) | 10M | 1.43 ms | 6.57 ms | **Go ×4.6** | 8.2 KB | 59 | 125.7 MB |
| Eager-direct (fused, no plan) | 10M | 1.43 ms | 6.12 ms | **Go ×4.3** | 3.3 KB | 34 | 124.1 MB |
