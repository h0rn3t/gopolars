## Why

После завершения v0.4 волны gopolars покрывает базовый и расширенный core, но всё ещё имеет заметные пробелы относительно Python Polars в advanced joins, reshape-сценариях, expression namespaces и SQL surface. Следующий этап нужен сейчас, чтобы сократить migration friction для команд, которые хотят перейти с Python Polars на Go-стек без потери аналитической выразительности.

## What Changes

- Добавить parity-возможности для advanced joins: `asof`, `semi`, `anti`, `cross`, а также time-aware join сценарии.
- Добавить reshape-срез уровня Python Polars: `pivot`, `unpivot/melt`, расширенные explode/unnest шаблоны.
- Расширить expression namespaces для строк, дат/времени, list/struct edge-cases и null/NaN-семантики.
- Расширить SQL surface parity: subqueries, set-операции, дополнительные аналитические выражения и согласованные error-контракты.
- Формализовать lazy sink/materialization сценарии (`sink parquet/csv/ipc`) и стабильный контракт collect/materialize путей.
- Расширить conformance matrix и parity evidence под v0.5 с акцентом на миграционные сценарии с Python Polars.
- Внести **BREAKING** alignment изменения только при необходимости семантической совместимости, с обязательной migration документацией.

## Capabilities

### New Capabilities
- `advanced-joins-and-time-series`: поддержка asof/semi/anti/cross join и time-series join паттернов с детерминированной семантикой.
- `reshape-and-pivot-operations`: поддержка pivot/unpivot/melt и связанных reshape-паттернов для аналитических витрин.
- `sql-surface-parity-v0-5`: расширение SQL compatibility слоя для subqueries, set-операций и richer analytical queries.
- `lazy-sinks-and-materialization`: формализация sink-операций и materialization контрактов для lazy execution.

### Modified Capabilities
- `api-parity-surface`: расширение требований по API parity для reshape/advanced joins и namespace-покрытия.
- `expression-parity-and-dtypes`: расширение namespace операций и edge-case семантики dtype/null/NaN.
- `analytics-and-windowing`: расширение требований для rolling/dynamic/window аналитики и time-aware сценариев.
- `lazy-optimizer-streaming`: расширение optimizer/streaming контрактов под новые join/reshape/sql нагрузки.
- `io-connectors-and-interoperability`: обновление требований под sink-пути и более широкие dataset interoperability сценарии.
- `conformance-quality-suite`: обновление coverage/evidence gates и differential suite для v0.5 parity scope.
- `compatibility-and-versioning-policy`: усиление migration/deprecation правил для новых parity alignment изменений.

## Impact

- Затрагивает слои `pkg/frame`, `pkg/expr`, `pkg/sql`, `pkg/plan`, `pkg/exec`, `pkg/polars`, `pkg/io/*`, а также conformance/unit тесты, benchmarks, документацию и release-gates.
- Потребуются новые test fixtures и differential сценарии, ориентированные на реальные Python Polars migration pipelines.
- Увеличивается сложность planner/execution и требований к perf-регрессиям, что требует усиленной telemetry и стабильных quality gates.
