## 1. DataFrame Top-30 Utility Methods

- [x] 1.1 Реализовать `drop_nans`, `fill_nan`, `is_empty`, `n_unique`, `null_count`
- [x] 1.2 Реализовать `sample`, `to_dicts`, `with_row_count`, `with_row_index`, `estimated_size`
- [x] 1.3 Добавить unit/conformance тесты для DataFrame Top-30 методов

## 2. Expr Top-30 Analytic and Window Methods

- [x] 2.1 Реализовать `cum_sum`, `cum_count`, `rank`, `over`
- [x] 2.2 Реализовать `replace`, `fill_null`, `fill_nan`, `rolling_min/max/mean/sum/std`
- [x] 2.3 Добавить semantic parity тесты для Expr Top-30 методов

## 3. LazyFrame Top-30 Execution Methods

- [x] 3.1 Реализовать `collect_async`, `collect_batches`, `inspect`, `profile`
- [x] 3.2 Реализовать `join_where`, `sink_ndjson`, `sql`
- [x] 3.3 Добавить end-to-end тесты lazy semantics и materialization

## 4. SQLContext Bootstrap

- [x] 4.1 Реализовать `SQLContext.register` для DataFrame регистрации
- [x] 4.2 Обеспечить интеграцию регистрации с SQL execution path
- [x] 4.3 Добавить SQLContext conformance тесты

## 5. Quality Gates and Release Evidence

- [x] 5.1 Добавить Top-30 differential conformance профиль в CI
- [x] 5.2 Ввести coverage gate `30/30` для v0.7 release candidate
- [x] 5.3 Обновить parity matrix и release evidence документы
