# v0.2 Python Polars Parity Inventory

## DataFrame

| Namespace | Status | Notes |
|---|---|---|
| Construction (`DataFrame`, schema infer) | Partial | Basic construction is available, advanced constructors are missing |
| Projection/selection | Partial | `Select` exists, selectors/regex/selectors namespace is missing |
| Filtering | Partial | Boolean predicate filtering exists |
| Sorting | Partial | Multi-column sort exists, advanced null/order controls are limited |
| GroupBy/Agg | Partial | Core aggs exist (`sum/min/max/mean/count/n_unique`) |
| Join | Partial | `inner` and `left` exist, advanced join strategies/types are missing |
| Reshaping (`pivot/melt`) | Missing | Not implemented |
| Window ops | Missing | Not implemented |
| Rolling/dynamic groups | Missing | Not implemented |
| SQL context integrations | Partial | Baseline SQL subset is implemented |
| IO methods | Partial | CSV/JSON/Parquet and Arrow roundtrip are present; IPC/cloud are missing |

## LazyFrame

| Namespace | Status | Notes |
|---|---|---|
| Plan building (`Select/Filter/WithColumns`) | Present | Implemented |
| Join/sort/limit/groupby | Present | Implemented for baseline scenarios |
| Optimizer rewrites | Partial | Baseline rewrite rules exist |
| Explain contracts | Partial | Basic explain output exists, staged contract improved in v0.2 apply |
| Streaming execution | Missing | Dedicated streaming path is not implemented |

## Expr

| Namespace | Status | Notes |
|---|---|---|
| Arithmetic/comparison/boolean | Present | Core operators are implemented |
| Cast/alias/null checks | Present | Baseline support exists |
| Aggregation exprs | Present | `sum/min/max/mean/count/n_unique` |
| Conditional (`when/then/otherwise`) | Missing | Not implemented |
| String namespace | Missing | Namespace ops are not implemented |
| Datetime namespace | Missing | Namespace ops are not implemented |
| List/Struct namespace | Missing | Nested expression namespaces are not implemented |

## DTypes

| Type Family | Status | Notes |
|---|---|---|
| Primitive (`int64/float64/string/bool`) | Present | Implemented |
| Temporal (`datetime`) | Present | Implemented |
| Decimal | Missing | Not implemented |
| Categorical/Enum | Missing | Not implemented |
| List/Struct | Missing | Not implemented |

## Execution/Quality

| Area | Status | Notes |
|---|---|---|
| Physical plan/scheduler | Partial | Present with baseline operator execution |
| Benchmarks | Partial | Micro and macro benchmark smoke exist |
| Differential conformance | Partial | Introduced in v0.2 apply as optional suite vs Python Polars |
| Coverage gating | Partial | Introduced in v0.2 apply with threshold checks |
