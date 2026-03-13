## 1. Namespace Parity Expansion

- [x] 1.1 Расширить string/datetime/list/struct namespace API до v0.6 профиля
- [x] 1.2 Уточнить null/NaN и dtype edge-case поведение в expression evaluator
- [x] 1.3 Добавить differential fixtures для namespace parity и error-контрактов

## 2. Temporal Window Parity

- [x] 2.1 Реализовать `group_by_dynamic` с boundary/offset/closed параметрами
- [x] 2.2 Расширить rolling analytics контракт для eager/lazy/sql путей
- [x] 2.3 Добавить conformance тесты для temporal window edge-cases

## 3. Optimizer and Execution Hardening

- [x] 3.1 Обновить adaptive planning под window-heavy и mixed workloads
- [x] 3.2 Усилить streaming fallback правила для stateful temporal операторов
- [x] 3.3 Обновить explain diagnostics и stateful/perf markers

## 4. Performance Governance

- [x] 4.1 Ввести benchmark budget thresholds для parity-critical сценариев
- [x] 4.2 Добавить regression detection отчёты относительно baseline
- [x] 4.3 Интегрировать performance evidence в release quality gates

## 5. Observability and Telemetry

- [x] 5.1 Расширить runtime telemetry для temporal-window execution paths
- [x] 5.2 Стабилизировать diagnostics schema для CI-парсинга
- [x] 5.3 Добавить telemetry проверки в nightly pipelines

## 6. Conformance, Compatibility and Release Readiness

- [x] 6.1 Обновить coverage matrix и nightly differential suite под v0.6
- [x] 6.2 Усилить compatibility policy checks для namespace/temporal alignment
- [x] 6.3 Подготовить migration notes и release evidence для возможных **BREAKING** изменений
