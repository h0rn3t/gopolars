## Why

Після закриття v0.3 core-базису бібліотеці потрібен наступний feature-wave для прикладної аналітики на production-навантаженнях: глибший SQL/windowing, nested типи, cloud-native IO та краща керованість виконання. Без цього gopolars залишається сильним на базових pipeline, але обмеженим для складних сценаріїв і довгих release-циклів команд.

## What Changes

- Визначити roadmap v0.4 з пріоритетом на аналітичні можливості верхнього рівня: advanced SQL, window engine, nested/list/struct обчислення.
- Додати cloud-native орієнтири для IO: partition-aware scan, multi-file datasets, стабільні object-store сценарії.
- Розширити execution observability: explain diagnostics, operator metrics, профілювання регресій.
- Ввести release policy для сумісності API і семантичної стабільності між minor-версіями.
- Посилити conformance контур: matrix по нових фічах, nightly differential набір, стабільні quality gates.

## Capabilities

### New Capabilities
- `advanced-sql-and-window-parity`: розширення SQL surface (CTE, window expressions, richer aggregations) з parity-поведінкою.
- `nested-types-and-transforms`: підтримка list/struct pipeline-операцій, flatten/explode/map-подібних трансформацій і сумісної типізації.
- `cloud-lakehouse-io`: partition-aware scan/read для dataset-структур і стабільний cloud object-store execution profile.
- `execution-observability`: explain diagnostics, operator-level telemetry, traceable performance budget signals.
- `compatibility-and-versioning-policy`: формалізована policy для API evolution, deprecation windows і migration guarantees.

### Modified Capabilities
- `api-parity-surface`: розширення вимог щодо SQL/window та nested namespace parity для v0.4 profile.
- `lazy-optimizer-streaming`: оновлення вимог для adaptive planning, pushdown coverage і streaming fallback semantics.
- `io-connectors-and-interoperability`: оновлення вимог для partitioned datasets, multi-file IO і cloud interop contracts.
- `conformance-quality-suite`: оновлення вимог для nightly differential, capability coverage matrix і release evidence.

## Impact

- Зачіпає ключові шари `pkg/sql`, `pkg/plan`, `pkg/exec`, `pkg/expr`, `pkg/frame`, `pkg/io/*`, `pkg/polars`, а також `test/conformance`, `test/unit`, `bench/*`, `docs/*`, CI workflows.
- Вимагає розширення специфікацій і тестових фікстур для SQL/window/nested/cloud сценаріїв.
- Може включати **BREAKING** API-alignment зміни для узгодження namespace і behavior контрактів у v0.4.
