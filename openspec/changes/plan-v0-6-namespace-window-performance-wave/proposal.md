## Why

После v0.5 проект закрыл критичные join/reshape/sql/sink пробелы, но migration-командам всё ещё не хватает глубины namespace API, продвинутых temporal window сценариев (`group_by_dynamic`/`rolling`) и стабильной performance-предсказуемости на long-running аналитических нагрузках. v0.6 wave нужен сейчас, чтобы перевести gopolars из уровня feature parity в устойчивый production parity профиль.

## What Changes

- Расширить namespace parity для string/datetime/list/struct выражений, включая edge-cases и error contracts.
- Добавить полноценный API/engine контракт для `group_by_dynamic` и расширенного `rolling` с rule-level semantic parity.
- Усилить planner/optimizer/runtimes для window-heavy и mixed workloads с performance hardening gates.
- Расширить observability/telemetry для детального анализа window/stateful execution и memory pressure.
- Обновить conformance/coverage matrix под v0.6 с отдельными треками namespace + temporal analytics + perf budgets.
- Зафиксировать compatibility policy и migration guidance для alignment изменений, включая потенциально **BREAKING** семантические корректировки.

## Capabilities

### New Capabilities
- `temporal-window-parity-v0-6`: единый контракт для `group_by_dynamic`, `rolling` и related temporal analytics semantics.
- `performance-hardening-and-budgets`: performance governance capability для budget gates, regressions detection и benchmark evidence.

### Modified Capabilities
- `expression-parity-and-dtypes`: расширение namespace coverage и null/NaN/dtype edge-cases для v0.6.
- `analytics-and-windowing`: обновление требований для expanded temporal window behavior и mixed-window pipelines.
- `lazy-optimizer-streaming`: усиление adaptive planning и fallback-политик под stateful window-heavy workloads.
- `execution-observability`: расширение execution diagnostics и telemetry под performance hardening сценарии.
- `conformance-quality-suite`: обновление differential coverage, evidence matrix и quality gates для v0.6.
- `compatibility-and-versioning-policy`: уточнение compatibility/deprecation практик для parity-alignment изменений v0.6.

## Impact

- Основной impact на `pkg/expr`, `pkg/frame`, `pkg/exec`, `pkg/plan/*`, `pkg/polars`, test suites и benchmarking pipeline.
- Понадобятся новые fixture datasets для temporal windows и отдельные perf regression сценарии.
- Увеличится объём release evidence и CI checks, но это снизит риск semantic/performance regressions при production adoption.
