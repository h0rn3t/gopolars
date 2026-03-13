# lazy-sinks-and-materialization Specification

## Purpose
TBD - synced from change plan-v0-5-python-polars-parity-wave-2. Update Purpose if needed.

## Requirements
### Requirement: Lazy execution SHALL support sink operations for primary formats
Система SHALL поддерживать lazy sink в Parquet/CSV/IPC с предсказуемым контрактом записи.

#### Scenario: Sink lazy pipeline to parquet
- **WHEN** пользователь строит lazy pipeline и вызывает sink в parquet
- **THEN** система записывает корректный output dataset с согласованной schema и значениями

### Requirement: Materialization contracts SHALL be consistent across collect and sink paths
Система MUST обеспечивать согласованность materialization semantics между `collect` и `sink` путями.

#### Scenario: Compare collect and sink-readback results
- **WHEN** один pipeline materialize через collect и через sink+readback
- **THEN** результаты эквивалентны по значениям, schema и null/NaN поведению

### Requirement: Sink execution SHALL publish diagnostics and failure reasons
Система SHALL предоставлять диагностируемые статусы sink-операций для CI и production observability.

#### Scenario: Capture sink failure diagnostics
- **WHEN** sink завершается ошибкой записи
- **THEN** система возвращает структурированную причину сбоя без потери контекста execution stage
