## Why

Текущий этап закрыл v0.2 parity-срез, но для практического MVP уровня “developer-ready” остаются критические блоки: публичный Series API, truly-lazy Scan pipeline, production-ready Parquet backend и более строгие quality/perf gates. Без этого Go-версия Polars остаётся частично совместимой и ограниченной в производительности на реальных ETL/analytics сценариях.

## What Changes

- Сформировать целевой v0.3 план реализации core parity для `DataFrame`, `Series`, `LazyFrame` с фокусом на MVP-критичный API.
- Зафиксировать архитектурные решения для производительности: source-level lazy execution, pushdown, bounded-memory streaming, оптимизаторные правила.
- Зафиксировать интеграционные требования для Arrow `23.0.1` и Parquet (`parquet-go v0.29.0`) с чёткими контрактами совместимости.
- Определить roadmap с фазами, контрольными точками и критериями “go/no-go” для release candidate.
- Ввести измеримые quality/performance метрики и release gates (coverage, conformance parity, race/stability, benchmark regression budget).
- Специфицировать тестовую стратегию (unit/integration/benchmark/differential) и минимальные требования к документации/примером для разработчиков.

## Capabilities

### New Capabilities
- `series-public-api`: публичный контракт Series-уровня и column-first операций для parity с Polars.
- `lazy-scan-and-pushdown`: source-level scan nodes, projection/predicate pushdown и stream-aware lazy execution.
- `parquet-production-backend`: production-ready parquet read/write с `parquet-go v0.29.0` и schema/logical type mapping.
- `core-analytics-mvp-surface`: обязательный MVP-срез DataFrame/LazyFrame методов для прикладных аналитических сценариев.
- `quality-and-performance-gates-v0-3`: обязательные CI/release метрики качества, стабильности и производительности.
- `developer-docs-and-examples-v0-3`: обязательный набор developer-facing документации, migration notes и runnable examples.

### Modified Capabilities

- *(none)*

## Impact

- Затрагивает `pkg/polars`, `pkg/frame`, `pkg/series`, `pkg/expr`, `pkg/exec`, `pkg/plan/optimizer`, `pkg/io/parquet`, `pkg/io/arrow`, `test/unit`, `test/conformance`, `bench/*`, `.github/workflows/*`, `docs/*`.
- Добавляет или обновляет внешние зависимости вокруг Arrow/Parquet stack и требует version pinning policy.
- Может включать **BREAKING** API-изменения для выравнивания с контрактами Polars (прежде всего в области Series и Lazy Scan).
