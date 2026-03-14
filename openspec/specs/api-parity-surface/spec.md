# api-parity-surface Specification

## Purpose
TBD - created by archiving change plan-v0-2-python-polars-parity. Update Purpose after archive.
## Requirements
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

### Requirement: Series API SHALL provide parity for scalar, vector and namespace ops
Система SHALL підтримувати parity операцій Series, включно з numeric/string/datetime/list/struct namespaces у межах v0.5 capability matrix.

#### Scenario: Series namespace parity
- **WHEN** користувач виконує namespace-операції Series у відповідних типах
- **THEN** система повертає результати й помилки, сумісні з Polars Python semantics

### Requirement: DataFrame, LazyFrame, Expr, Series, SQLContext SHALL achieve complete parity closure
Система SHALL закрыть весь remaining методный backlog по объектам `DataFrame`, `LazyFrame`, `Expr`, `Series`, `SQLContext` из parity матрицы.

#### Scenario: Remaining methods become zero
- **WHEN** выполняется финальный parity gate
- **THEN** количество нереализованных методов в актуальном отчёте равно 0

### Requirement: Public API growth SHALL preserve compatibility contracts
Система MUST сохранять обратную совместимость и устойчивые контракты поведения при массовом расширении surface.

#### Scenario: Existing workloads after parity expansion
- **WHEN** пользователи обновляются на релиз с полным parity closure
- **THEN** ранее рабочие сценарии продолжают выполняться без регрессий
