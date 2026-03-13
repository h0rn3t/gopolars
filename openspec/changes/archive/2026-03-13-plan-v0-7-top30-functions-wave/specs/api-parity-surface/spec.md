## MODIFIED Requirements

### Requirement: DataFrame API SHALL match Python Polars core method surface
Система SHALL расширить DataFrame API v0.7 high-priority utility методами: `drop_nans`, `fill_nan`, `is_empty`, `n_unique`, `null_count`, `sample`, `to_dicts`, `with_row_count`, `with_row_index`, `estimated_size`.

#### Scenario: DataFrame utility parity for Top-30
- **WHEN** пользователь применяет DataFrame utility функции из v0.7 Top-30
- **THEN** система возвращает поведение, совместимое с Python Polars в утверждённом профиле

### Requirement: LazyFrame API SHALL preserve Polars lazy semantics
Система SHALL предоставить lazy-функции `collect_async`, `collect_batches`, `inspect`, `profile`, `join_where`, `sink_ndjson`, `sql` с детерминированной семантикой.

#### Scenario: Lazy ergonomics parity
- **WHEN** пользователь запускает Top-30 lazy функции в реальном pipeline
- **THEN** система сохраняет deferred execution semantics и корректную materialization behavior
