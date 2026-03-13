## Context

В `docs/parity/python_polars_full_matrix.md` зафиксирован high-priority Top-30 backlog для v0.7.  
Набор распределён по четырём зонам: DataFrame utility methods, Expr window/null transformations, LazyFrame execution ergonomics и SQLContext registration.

Текущая архитектура gopolars уже имеет рабочее ядро eager/lazy/SQL и observability v2, поэтому v0.7 можно проводить как целевую capability wave без пересмотра базовой архитектуры.

## Goals / Non-Goals

**Goals**
- Реализовать все 30 функций из `docs/parity/v0_7_top30_functions.md`.
- Сохранить единый semantic contract между eager/lazy/sql execution paths.
- Добавить test evidence и quality gates на Top-30 scope.
- Поднять measurable parity coverage и обеспечить удобный migration uplift.

**Non-Goals**
- Полная parity реализация всех `607` нереализованных методов.
- Полный multi-table SQL catalog в пределах одного релиза.
- Глобальный рефакторинг физического планировщика вне потребностей Top-30.

## Delivery Slices

1. **DataFrame utilities slice**  
`drop_nans`, `fill_nan`, `is_empty`, `n_unique`, `null_count`, `sample`, `to_dicts`, `with_row_count`, `with_row_index`, `estimated_size`.

2. **Expr analytic/window slice**  
`cum_sum`, `cum_count`, `over`, `rank`, `replace`, `fill_null`, `fill_nan`, `rolling_min/max/mean/sum/std`.

3. **LazyFrame ergonomics slice**  
`collect_async`, `collect_batches`, `inspect`, `profile`, `join_where`, `sink_ndjson`, `sql`.

4. **SQLContext bootstrap slice**  
`register` как foundation для дальнейшего multi-table parity.

## Risks / Trade-offs

- [Semantic mismatch] Новые Expr/window операции могут расходиться с Polars edge-cases.  
  Митигация: differential fixtures для каждого семейства функций.

- [Runtime regressions] Rolling/rank/over могут повысить затраты памяти и latency.  
  Митигация: benchmark budget и operator-level telemetry checks.

- [API consistency drift] Параллельное расширение DataFrame/LazyFrame/SQLContext может нарушить naming/behavior consistency.  
  Митигация: единые acceptance tests на API equivalence.

## Validation Strategy

- Unit tests по каждой функции из Top-30.
- Conformance subset against Python Polars fixtures.
- Full CI gates: `go test ./...`, `go vet ./...`, `go test -race ./...`.
- Benchmark smoke + regression evidence на новых window/lazy функциях.

## Rollout Plan

1. Реализовать DataFrame и Expr high-value методы.
2. Добавить LazyFrame execution surface.
3. Добавить SQLContext `register` и связанный API flow.
4. Прогнать Top-30 conformance/performance suites.
5. Обновить parity matrix и release evidence.
