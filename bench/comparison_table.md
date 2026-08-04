## Top-30 Operations — gopolars vs Python Polars

> **Hardware:** Apple M4 Pro · Go 1.26 · Python Polars 1.41.2
> **Sizes:** 1 K rows and 1 M rows — min ns/op across calibration rounds

### DataFrame

| operation | size | Go time | Py time | speedup | Go B/op | Go allocs/op |
|-----------|------|---------|---------|---------|---------|--------------|
| `filter` | 1K | 5.8 µs | 109.8 µs | **Go ×18.9** | 23.1 KB | 22 |
| `filter` | 1M | 1.09 ms | 589.5 µs | Py ×1.8 | 19.8 MB | 116 |
| `select` | 1K | 588 ns | 30.9 µs | **Go ×52.5** | 1.5 KB | 10 |
| `select` | 1M | 464 ns | 33.1 µs | **Go ×71.3** | 1.5 KB | 10 |
| `with_columns` | 1K | 568 ns | 8.4 µs | **Go ×14.8** | 1.5 KB | 10 |
| `with_columns` | 1M | 449 ns | 9.1 µs | **Go ×20.2** | 1.5 KB | 10 |
| `sort` | 1K | 24.2 µs | 201.3 µs | **Go ×8.3** | 66.6 KB | 19 |
| `sort` | 1M | 15.89 ms | 11.27 ms | Py ×1.4 | 62.0 MB | 153 |
| `group_by` | 1K | 14.9 µs | 706.5 µs | **Go ×47.5** | 18.6 KB | 41 |
| `group_by` | 1M | 1.48 ms | 1.63 ms | **Go ×1.1** | 90.7 KB | 186 |
| `join` | 1K | 89.2 µs | 341.5 µs | **Go ×3.8** | 217.4 KB | 727 |
| `join` | 1M | 6.93 ms | 5.94 ms | Py ×1.2 | 95.2 MB | 1590 |
| `head` | 1K | 466 ns | 620 ns | **Go ×1.3** | 1.6 KB | 10 |
| `head` | 1M | 326 ns | 624 ns | **Go ×1.9** | 1.6 KB | 10 |
| `tail` | 1K | 466 ns | 612 ns | **Go ×1.3** | 1.6 KB | 10 |
| `tail` | 1M | 327 ns | 658 ns | **Go ×2.0** | 1.6 KB | 10 |
| `unique` | 1K | 8.8 µs | 146.3 µs | **Go ×16.6** | 1.9 KB | 21 |
| `unique` | 1M | 992.2 µs | 4.35 ms | **Go ×4.4** | 70.6 KB | 147 |
| `fill_null` | 1K | 2.5 µs | 116.0 µs | **Go ×46.1** | 10.0 KB | 10 |
| `fill_null` | 1M | 412.0 µs | 718.1 µs | **Go ×1.7** | 8.6 MB | 35 |
| `drop_nulls` | 1K | 10.4 µs | 58.7 µs | **Go ×5.6** | 42.7 KB | 17 |
| `drop_nulls` | 1M | 1.63 ms | 1.31 ms | Py ×1.2 | 35.4 MB | 64 |

### Expr

| operation | size | Go time | Py time | speedup | Go B/op | Go allocs/op |
|-----------|------|---------|---------|---------|---------|--------------|
| `cum_sum` | 1K | 2.6 µs | 13.1 µs | **Go ×5.1** | 10.7 KB | 14 |
| `cum_sum` | 1M | 779.3 µs | 2.94 ms | **Go ×3.8** | 8.6 MB | 14 |
| `rank` | 1K | 18.4 µs | 67.2 µs | **Go ×3.7** | 34.7 KB | 17 |
| `rank` | 1M | 14.08 ms | 13.71 ms | Py ×1.0 | 31.5 MB | 47 |
| `over` | 1K | 14.9 µs | 315.8 µs | **Go ×21.1** | 28.2 KB | 28 |
| `over` | 1M | 4.66 ms | 9.53 ms | **Go ×2.0** | 25.2 MB | 137 |
| `fill_null` | 1K | 3.2 µs | 40.1 µs | **Go ×12.6** | 10.9 KB | 16 |
| `fill_null` | 1M | 402.3 µs | 675.0 µs | **Go ×1.7** | 8.6 MB | 41 |
| `fill_nan` | 1K | 1.2 µs | 73.1 µs | **Go ×60.7** | 1.9 KB | 15 |
| `fill_nan` | 1M | 75.5 µs | 871.1 µs | **Go ×11.5** | 2.6 KB | 40 |
| `rolling_mean` | 1K | 6.4 µs | 18.3 µs | **Go ×2.8** | 10.7 KB | 16 |
| `rolling_mean` | 1M | 4.51 ms | 9.22 ms | **Go ×2.0** | 8.6 MB | 16 |
| `rolling_sum` | 1K | 6.6 µs | 20.0 µs | **Go ×3.0** | 10.7 KB | 16 |
| `rolling_sum` | 1M | 4.39 ms | 9.05 ms | **Go ×2.1** | 8.6 MB | 16 |
| `rolling_min` | 1K | 4.9 µs | 15.2 µs | **Go ×3.1** | 11.5 KB | 17 |
| `rolling_min` | 1M | 8.11 ms | 12.05 ms | **Go ×1.5** | 8.9 MB | 28 |
| `rolling_max` | 1K | 5.3 µs | 14.7 µs | **Go ×2.8** | 11.5 KB | 17 |
| `rolling_max` | 1M | 8.12 ms | 12.15 ms | **Go ×1.5** | 8.9 MB | 28 |

### Series

| operation | size | Go time | Py time | speedup | Go B/op | Go allocs/op |
|-----------|------|---------|---------|---------|---------|--------------|
| `null_count` | 1K | 2 ns | 487 ns | **Go ×243.8** | — | 0 |
| `null_count` | 1M | 2 ns | 550 ns | **Go ×275.0** | — | 0 |
| `drop_nans` | 1K | 394 ns | 12.3 µs | **Go ×31.2** | 300 B | 4 |
| `drop_nans` | 1M | 73.7 µs | 99.7 µs | **Go ×1.4** | 988 B | 29 |
| `to_list` | 1K | 9.7 µs | 10.4 µs | **Go ×1.1** | 23.8 KB | 1001 |
| `to_list` | 1M | 7.42 ms | 13.26 ms | **Go ×1.8** | 22.9 MB | 1000001 |
| `is_null` | 1K | 414 ns | 11.5 µs | **Go ×27.8** | 2.2 KB | 4 |
| `is_null` | 1M | 45.1 µs | 11.8 µs | Py ×3.8 | 1.9 MB | 4 |
| `is_not_null` | 1K | 409 ns | 11.5 µs | **Go ×28.1** | 2.2 KB | 4 |
| `is_not_null` | 1M | 68.7 µs | 13.6 µs | Py ×5.0 | 1.9 MB | 4 |
| `fill_nan` | 1K | 393 ns | 95.9 µs | **Go ×244.0** | 300 B | 4 |
| `fill_nan` | 1M | 73.9 µs | 852.7 µs | **Go ×11.5** | 988 B | 29 |

### LazyFrame

| operation | size | Go time | Py time | speedup | Go B/op | Go allocs/op |
|-----------|------|---------|---------|---------|---------|--------------|
| `collect` | 1K | 86 ns | 4.0 µs | **Go ×47.0** | 192 B | 2 |
| `collect` | 1M | 69 ns | 4.8 µs | **Go ×70.1** | 192 B | 2 |
| `inspect` | 1K | 31 ns | 1.2 µs | **Go ×38.3** | 112 B | 1 |
| `inspect` | 1M | 20 ns | 1.1 µs | **Go ×56.9** | 112 B | 1 |

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
