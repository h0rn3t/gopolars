## Why

Поточний MVP v0.1 покриває базові сценарії DataFrame, але не досягає функціонального паритету з Python Polars. Для версії 0.2 потрібен формалізований план, який визначить повне покриття API, семантики та якості сумісності.

## What Changes

- Визначити цільовий обсяг “100% feature parity” з Python Polars на рівні capability-specs і вимог.
- Зафіксувати повний API surface для eager/lazy, expressions, joins, windowing, reshaping, nested types, SQL і streaming.
- Додати вимоги до IO-конекторів, cloud/object-store сценаріїв, interchange з Arrow/Pandas і сумісності семантики null/NaN.
- Визначити обов’язковий conformance test suite проти еталонних сценаріїв Polars.
- Додати roadmap задач із фазуванням робіт, залежностями і release gates для v0.2.

## Capabilities

### New Capabilities
- `api-parity-surface`: повний контракт методів DataFrame/LazyFrame/Series до рівня Python Polars.
- `expression-parity-and-dtypes`: повне покриття expression engine, dtype system, casting, null/NaN semantics.
- `analytics-and-windowing`: window functions, rolling/dynamic groups, temporal analytics, advanced aggregations.
- `io-connectors-and-interoperability`: повний набір IO форматів, cloud storage, Arrow/Pandas/IPC interoperability.
- `lazy-optimizer-streaming`: rule/cost optimizer, streaming execution, join strategies, plan explainability.
- `conformance-quality-suite`: parity tests, differential tests against Polars Python, benchmarks and stability gates.

### Modified Capabilities

- *(none)*

## Impact

- Зачіпає всі ключові пакети (`pkg/polars`, `pkg/expr`, `pkg/frame`, `pkg/plan`, `pkg/exec`, `pkg/io`, `pkg/sql`).
- Вимагає розширення тестової та benchmark інфраструктури і CI policy.
- Формує основу для наступних implementation changes v0.2 та потенційно містить **BREAKING** API-alignment зміни.
