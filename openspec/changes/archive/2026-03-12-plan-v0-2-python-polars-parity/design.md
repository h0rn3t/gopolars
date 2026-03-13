## Context

v0.1 закрив базову аналітичну модель (DataFrame/LazyFrame, базовий optimizer, CSV/JSON/Parquet/Arrow, SQL subset), але Python Polars має значно ширший API surface і складнішу семантику виконання. Для v0.2 потрібен єдиний технічний план, який формалізує full-parity scope, порядок реалізації і контроль сумісності.

Ключові обмеження:
- Go-реалізація має залишатися idiomatic і продуктивною без прямого копіювання внутрішньої Rust-архітектури Polars.
- Потрібна сумісність поведінки на рівні observable semantics (результати, null/NaN, sorting/grouping rules), а не тільки назви API.
- Scope “100% parity” вимагає конформанс-тестів проти еталонного Python Polars.

## Goals / Non-Goals

**Goals:**
- Визначити повну карту capability-рівня для паритету з Python Polars.
- Зафіксувати технічні рішення щодо API-покриття, execution semantics, IO/interchange, optimizer/streaming.
- Визначити фази delivery та release gates, які можна перевіряти автоматично.
- Визначити стратегію differential testing проти Python Polars.

**Non-Goals:**
- Реалізація всіх parity-функцій у межах цього change.
- Byte-level сумісність внутрішніх планів або приватних implementation details Polars Rust.
- Гарантія повної backwards compatibility з pre-v0.2 API, якщо для parity потрібні breaking alignment зміни.

## Decisions

1. Capability-driven decomposition
- Рішення: розбити parity на 6 capability-блоків із власними вимогами.
- Чому: зменшує ризик “монолітного” плану і дозволяє незалежно рухати реалізацію.
- Альтернатива: один umbrella spec. Відхилено через низьку керованість і погану трасованість прогресу.

2. Semantics-first parity contract
- Рішення: визначати parity через тестовані SHALL-вимоги і сценарії поведінки.
- Чому: однакові результати важливіші за ідентичність внутрішньої архітектури.
- Альтернатива: API-name parity only. Відхилено як недостатньо для реальної сумісності.

3. Differential conformance testing as release gate
- Рішення: додати обов’язковий набір порівняльних тестів gopolars vs python polars.
- Чому: це єдиний практичний доказ “100% parity”.
- Альтернатива: лише локальні unit tests. Відхилено через високий ризик semantic drift.

4. Phased execution roadmap
- Рішення: виконувати v0.2 у фазах: API+Expr → Analytics/Windowing → IO/Interop → Optimizer/Streaming → Conformance hardening.
- Чому: дозволяє incremental delivery без втрати цілісності.
- Альтернатива: big-bang delivery. Відхилено через ризики інтеграції та регресій.

## Risks / Trade-offs

- [Scope explosion] Повне parity-покриття має великий обсяг → Мітигація: capability milestones, strict DoD per capability.
- [Semantic mismatches] Нюанси null/NaN/order semantics можуть розходитись → Мітигація: differential tests + golden fixtures.
- [Performance regressions] Розширення функціоналу може погіршити latency/allocs → Мітигація: benchmark gates для критичних операторів.
- [Dependency pressure] IO/connectors можуть вимагати нових залежностей → Мітигація: dependency review і optional build flags.
- [Breaking alignment] Частина API може потребувати зміни контрактів v0.1 → Мітигація: explicit migration notes і compatibility shims де можливо.

## Migration Plan

1. Затвердити capability-specs і tasks як базовий v0.2 roadmap.
2. Створити серію implementation changes по capability-блоках з власними delta specs.
3. Для кожного блоку додавати conformance tests і benchmark smoke до CI.
4. Ввести pre-release matrix: API parity %, semantic parity %, perf regression budget.
5. Перед релізом v0.2 виконати повний differential suite і зафіксувати coverage report.

Rollback strategy:
- Якщо capability не досягає quality gates, виключити її з release profile і залишити за feature flag до стабілізації.

## Open Questions

- Який мінімальний перелік Polars API вважати “100%” (core + optional namespaces)?
- Чи потрібна повна SQL parity з Polars SQLContext у v0.2 або частковий профіль?
- Який рівень cloud IO support обов’язковий (S3/GCS/Azure parity matrix)?
- Яка policy для expression plugins/UDF: strict parity чи idiomatic Go extension model?
