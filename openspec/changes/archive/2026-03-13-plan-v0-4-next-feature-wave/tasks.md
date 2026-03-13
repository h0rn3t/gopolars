## 1. Advanced SQL and Window Foundation

- [x] 1.1 Розширити SQL parser/binder для CTE і window expression конструкцій
- [x] 1.2 Додати planner підтримку window logical nodes і валідацію frame semantics
- [x] 1.3 Реалізувати conformance тести для SQL/window parity fixtures

## 2. Nested Types and Transform Operations

- [x] 2.1 Зафіксувати list/struct dtype contracts у frame/series/expr шарах
- [x] 2.2 Реалізувати MVP nested трансформації для explode/flatten сценаріїв
- [x] 2.3 Додати eager/lazy parity тести для nested pipeline поведінки

## 3. Cloud Lakehouse IO Profile

- [x] 3.1 Реалізувати partition-aware dataset scan/read для multi-file layout
- [x] 3.2 Додати pruning логіку за partition key predicates для cloud scans
- [x] 3.3 Покрити object-store integration тести з diagnostics перевірками

## 4. Lazy Optimizer and Streaming Evolution

- [x] 4.1 Додати adaptive planning rules для mixed window/nested/cloud workloads
- [x] 4.2 Розширити streaming fallback diagnostics і deterministic behavior checks
- [x] 4.3 Валідувати explain контракти для нових optimizer/execution path рішень

## 5. Execution Observability

- [x] 5.1 Додати operator-level telemetry signals у execution lifecycle
- [x] 5.2 Зафіксувати стабільний diagnostics output schema для CI automation
- [x] 5.3 Інтегрувати performance budget evidence у benchmark pipeline звіти

## 6. Compatibility and Versioning Governance

- [x] 6.1 Впровадити policy класифікацію compatible/deprecating/breaking змін
- [x] 6.2 Додати deprecation windows і migration contract у release процес
- [x] 6.3 Підключити release gate для migration evidence при breaking changes

## 7. Conformance and Release Readiness

- [x] 7.1 Розширити capability coverage matrix для v0.4 feature wave
- [x] 7.2 Додати nightly differential suite для SQL/window/nested/cloud сценаріїв
- [x] 7.3 Оновити release checklist з quality/perf/compatibility доказами
