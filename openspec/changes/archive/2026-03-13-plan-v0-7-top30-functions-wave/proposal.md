## Why

Матрица parity показала, что при покрытии `73/680` наибольший практический эффект даст реализация целевого `Top-30` high-priority функций из `docs/parity/v0_7_top30_functions.md`.  
Этот набор закрывает критичные пробелы в DataFrame utility API, window/analytic Expr-паттернах, lazy materialization ergonomics и SQLContext registration flow.

## What Changes

- Реализовать `Top-30` методов v0.7 как согласованный delivery wave по DataFrame/Expr/LazyFrame/SQLContext.
- Сфокусировать первую очередь на runtime-значимых функциях: `drop_nans`, `fill_nan`, `sample`, `n_unique`, `null_count`, `with_row_index`, `with_row_count`.
- Добавить Expr-контур для `over`, `rank`, `cum_sum`, `cum_count`, `replace`, `fill_null`, `fill_nan` и rolling* family.
- Расширить LazyFrame surface для `collect_async`, `collect_batches`, `inspect`, `profile`, `join_where`, `sink_ndjson`, `sql`.
- Ввести SQLContext registration flow (`register`) как старт к multi-table SQL parity.
- Закрепить differential/conformance/performance evidence именно на Top-30 scope.

## Capabilities

### New Capabilities
- `v0-7-top30-function-delivery`: capability для целевого выполнения Top-30 функций с трассируемыми acceptance criteria.

### Modified Capabilities
- `api-parity-surface`: расширение DataFrame/LazyFrame API по Top-30.
- `expression-parity-and-dtypes`: расширение Expr namespace/window/null handling.
- `advanced-sql-and-window-parity`: добавление SQLContext registration baseline.
- `conformance-quality-suite`: отдельный Top-30 conformance profile и coverage gate.
- `execution-observability`: расширенные diagnostics для новых lazy и window execution paths.

## Impact

- Изменения затронут `pkg/polars`, `pkg/frame`, `pkg/expr`, `pkg/sql`, `pkg/exec`, `test/unit`, `test/conformance`, benchmark и docs.
- Увеличится объём CI checks за счёт Top-30 regression fixtures.
- Ожидаемое ускорение migration path для пользователей Python Polars в high-impact сценариях.
