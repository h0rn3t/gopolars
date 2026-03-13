## 1. Advanced Join Engine

- [x] 1.1 Реализовать `semi`, `anti` и `cross` join в DataFrame и LazyFrame API
- [x] 1.2 Реализовать `asof` join с direction/tolerance контрактом для time-series
- [x] 1.3 Добавить unit и differential тесты для advanced join семантики

## 2. Reshape and Pivot Surface

- [x] 2.1 Реализовать `pivot` операции с контролируемой агрегацией
- [x] 2.2 Реализовать `unpivot/melt` и расширить explode/unnest контракты
- [x] 2.3 Добавить eager/lazy parity тесты для reshape сценариев

## 3. Expression Namespace Expansion

- [x] 3.1 Расширить string/datetime/list/struct namespace операции до v0.5 профиля
- [x] 3.2 Зафиксировать dtype/null/NaN edge-case семантику в expression evaluator
- [x] 3.3 Добавить conformance fixtures для namespace parity и error-контрактов

## 4. SQL Surface Parity v0.5

- [x] 4.1 Добавить поддержку subqueries в SQL parser/binder/planner
- [x] 4.2 Реализовать set-операции `UNION`, `INTERSECT`, `EXCEPT`
- [x] 4.3 Провести SQL differential валидацию против Python Polars сценариев

## 5. Lazy Sinks and Materialization

- [x] 5.1 Реализовать lazy sink для Parquet, CSV и IPC
- [x] 5.2 Согласовать collect и sink материализацию по schema/value семантике
- [x] 5.3 Добавить диагностируемые sink-ошибки и execution-stage metadata

## 6. Optimizer and Streaming Hardening

- [x] 6.1 Расширить adaptive planning под advanced join/reshape/sql нагрузки
- [x] 6.2 Обновить streaming fallback правила для новых stateful операторов
- [x] 6.3 Обновить explain/diagnostics контракт для новых execution path

## 7. IO and Interoperability Enhancements

- [x] 7.1 Расширить IO read/write/sink контракты для v0.5 multi-format parity
- [x] 7.2 Усилить object-store dataset сценарии и pruning diagnostics
- [x] 7.3 Проверить Arrow/dataframe interchange после reshape и advanced joins

## 8. Conformance, Compatibility and Release Gates

- [x] 8.1 Обновить coverage matrix и nightly differential suite под v0.5 scope
- [x] 8.2 Усилить compatibility policy checks для parity alignment изменений
- [x] 8.3 Подготовить migration notes и release evidence для возможных **BREAKING** изменений
