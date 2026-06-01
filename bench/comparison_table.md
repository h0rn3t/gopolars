## Top-30 Operations — gopolars vs Python Polars

> **Hardware:** Apple M4 Pro · Go 1.26 · Python Polars 1.41.2
> **Sizes:** 1 K rows and 1 M rows — min ns/op across calibration rounds

### DataFrame

| operation | size | Go time | Py time | speedup | Go B/op | Go allocs/op |
|-----------|------|---------|---------|---------|---------|--------------|
| `filter` | 1K | 9.3 µs | 98.1 µs | **Go ×10.5** | 28.3 KB | 25 |
| `filter` | 1M | 5.81 ms | 564.1 µs | Py ×10.3 | 25.1 MB | 112 |
| `select` | 1K | 279 ns | 43.9 µs | **Go ×157.4** | 864 B | 7 |
| `select` | 1M | 220 ns | 35.9 µs | **Go ×163.4** | 864 B | 7 |
| `with_columns` | 1K | 403 ns | 8.5 µs | **Go ×21.0** | 1.1 KB | 8 |
| `with_columns` | 1M | 315 ns | 8.0 µs | **Go ×25.5** | 1.1 KB | 8 |
| `sort` | 1K | 28.4 µs | 173.8 µs | **Go ×6.1** | 69.6 KB | 22 |
| `sort` | 1M | 38.92 ms | 11.51 ms | Py ×3.4 | 64.9 MB | 22 |
| `group_by` | 1K | 12.0 µs | 625.7 µs | **Go ×52.4** | 17.1 KB | 20 |
| `group_by` | 1M | 18.07 ms | 1.57 ms | Py ×11.5 | 15.3 MB | 21 |
| `join` | 1K | 148.5 µs | 349.2 µs | **Go ×2.4** | 311.1 KB | 1385 |
| `join` | 1M | 47.40 ms | 6.40 ms | Py ×7.4 | 122.5 MB | 2120 |
| `head` | 1K | 2.1 µs | 620 ns | Py ×3.4 | 7.2 KB | 19 |
| `head` | 1M | 1.6 µs | 612 ns | Py ×2.5 | 7.2 KB | 19 |
| `tail` | 1K | 2.2 µs | 604 ns | Py ×3.7 | 7.4 KB | 21 |
| `tail` | 1M | 1.6 µs | 629 ns | Py ×2.5 | 7.4 KB | 21 |
| `unique` | 1K | 9.6 µs | 161.1 µs | **Go ×16.8** | 9.9 KB | 24 |
| `unique` | 1M | 16.44 ms | 4.92 ms | Py ×3.3 | 7.6 MB | 24 |
| `fill_null` | 1K | 1.9 µs | 132.8 µs | **Go ×69.6** | 9.9 KB | 9 |
| `fill_null` | 1M | 1.16 ms | 870.8 µs | Py ×1.3 | 8.6 MB | 9 |
| `drop_nulls` | 1K | 48.1 µs | 113.9 µs | **Go ×2.4** | 53.6 KB | 20 |
| `drop_nulls` | 1M | 43.87 ms | 1.36 ms | Py ×32.4 | 49.6 MB | 20 |

### Expr

| operation | size | Go time | Py time | speedup | Go B/op | Go allocs/op |
|-----------|------|---------|---------|---------|---------|--------------|
| `cum_sum` | 1K | 1.9 µs | 13.1 µs | **Go ×6.8** | 10.0 KB | 10 |
| `cum_sum` | 1M | 768.1 µs | 2.87 ms | **Go ×3.7** | 8.6 MB | 10 |
| `rank` | 1K | 69.8 µs | 61.0 µs | Py ×1.1 | 18.0 KB | 13 |
| `rank` | 1M | 336.64 ms | 13.29 ms | Py ×25.3 | 16.2 MB | 13 |
| `over` | 1K | 13.4 µs | 224.0 µs | **Go ×16.7** | 27.5 KB | 22 |
| `over` | 1M | 18.07 ms | 9.10 ms | Py ×2.0 | 24.8 MB | 22 |
| `fill_null` | 1K | 2.1 µs | 53.3 µs | **Go ×25.3** | 10.1 KB | 11 |
| `fill_null` | 1M | 1.14 ms | 659.0 µs | Py ×1.7 | 8.6 MB | 11 |
| `fill_nan` | 1K | 2.2 µs | 99.5 µs | **Go ×45.9** | 10.1 KB | 11 |
| `fill_nan` | 1M | 1.22 ms | 798.1 µs | Py ×1.5 | 8.6 MB | 11 |
| `rolling_mean` | 1K | 5.9 µs | 17.7 µs | **Go ×3.0** | 10.0 KB | 12 |
| `rolling_mean` | 1M | 4.97 ms | 8.89 ms | **Go ×1.8** | 8.6 MB | 12 |
| `rolling_sum` | 1K | 5.8 µs | 19.0 µs | **Go ×3.3** | 10.0 KB | 12 |
| `rolling_sum` | 1M | 5.13 ms | 8.80 ms | **Go ×1.7** | 8.6 MB | 12 |
| `rolling_min` | 1K | 4.3 µs | 15.9 µs | **Go ×3.7** | 10.8 KB | 13 |
| `rolling_min` | 1M | 8.05 ms | 11.91 ms | **Go ×1.5** | 8.9 MB | 24 |
| `rolling_max` | 1K | 4.5 µs | 14.8 µs | **Go ×3.3** | 10.8 KB | 13 |
| `rolling_max` | 1M | 8.04 ms | 11.96 ms | **Go ×1.5** | 8.9 MB | 24 |

### Series

| operation | size | Go time | Py time | speedup | Go B/op | Go allocs/op |
|-----------|------|---------|---------|---------|---------|--------------|
| `null_count` | 1K | 2 ns | 462 ns | **Go ×231.3** | — | 0 |
| `null_count` | 1M | 2 ns | 795 ns | **Go ×397.9** | — | 0 |
| `drop_nans` | 1K | 2.0 µs | 12.1 µs | **Go ×6.1** | 9.2 KB | 4 |
| `drop_nans` | 1M | 1.15 ms | 97.8 µs | Py ×11.7 | 8.6 MB | 4 |
| `to_list` | 1K | 9.3 µs | 9.7 µs | **Go ×1.0** | 23.8 KB | 1001 |
| `to_list` | 1M | 7.11 ms | 13.19 ms | **Go ×1.9** | 22.9 MB | 1000001 |
| `is_null` | 1K | 382 ns | 11.2 µs | **Go ×29.3** | 2.2 KB | 4 |
| `is_null` | 1M | 69.4 µs | 11.6 µs | Py ×6.0 | 1.9 MB | 4 |
| `is_not_null` | 1K | 665 ns | 11.2 µs | **Go ×16.9** | 2.2 KB | 4 |
| `is_not_null` | 1M | 369.4 µs | 12.9 µs | Py ×28.6 | 1.9 MB | 4 |
| `fill_nan` | 1K | 1.6 µs | 95.6 µs | **Go ×60.7** | 9.2 KB | 4 |
| `fill_nan` | 1M | 664.6 µs | 735.9 µs | **Go ×1.1** | 8.6 MB | 4 |

### LazyFrame

| operation | size | Go time | Py time | speedup | Go B/op | Go allocs/op |
|-----------|------|---------|---------|---------|---------|--------------|
| `collect` | 1K | 120 ns | 3.7 µs | **Go ×30.5** | 304 B | 4 |
| `collect` | 1M | 98 ns | 4.3 µs | **Go ×43.4** | 304 B | 4 |
| `sql` | 1K | 27.0 µs | 11.4 µs | Py ×2.4 | 36.9 KB | 76 |
| `sql` | 1M | 6.45 ms | 11.6 µs | Py ×557.0 | 25.1 MB | 163 |
| `inspect` | 1K | 26 ns | 1.1 µs | **Go ×44.1** | 96 B | 1 |
| `inspect` | 1M | 18 ns | 1.1 µs | **Go ×62.5** | 96 B | 1 |

### SQLContext

| operation | size | Go time | Py time | speedup | Go B/op | Go allocs/op |
|-----------|------|---------|---------|---------|---------|--------------|
| `execute` | 1K | 1.6 µs | 6.3 µs | **Go ×3.9** | 3.7 KB | 31 |
| `execute` | 1M | 1.4 µs | 6.3 µs | **Go ×4.6** | 3.7 KB | 31 |
| `register` | 1K | 175 ns | 2.3 µs | **Go ×12.9** | 800 B | 3 |
| `register` | 1M | 155 ns | 2.5 µs | **Go ×16.4** | 800 B | 3 |
| `tables` | 1K | 235 ns | 1.9 µs | **Go ×7.9** | 816 B | 4 |
| `tables` | 1M | 276 ns | 1.8 µs | **Go ×6.6** | 816 B | 4 |


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
