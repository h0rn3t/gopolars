## v0.7 Migration Notes

### Highlights

- Добавлены DataFrame utility методы: `drop_nans`, `fill_nan`, `is_empty`, `n_unique`, `null_count`, `sample`, `to_dicts`, `with_row_count`, `with_row_index`, `estimated_size`.
- Добавлены Expr методы аналитики и window-вычислений: `cum_sum`, `cum_count`, `rank`, `over`, `replace`, `fill_null`, `fill_nan`, `rolling_*`.
- Добавлены LazyFrame методы: `collect_async`, `collect_batches`, `inspect`, `profile`, `join_where`, `sink_ndjson`, `sql`.
- Добавлен SQLContext с регистрацией таблиц (`register`) и выполнением SQL.

### Migration guidance

- Для DataFrame pipeline с ручной нумерацией строк используйте `with_row_count/with_row_index`.
- Для lazy materialization сценариев, где нужен стрим батчей, переходите на `collect_batches`.
- Для mixed SQL/API сценариев используйте `SQLContext.register` как единый вход в SQL execution.
- Для release readiness запускайте `scripts/check_top30_coverage.sh` вместе с общими quality gates.
