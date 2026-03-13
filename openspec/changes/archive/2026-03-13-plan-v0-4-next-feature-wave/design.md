## Context

v0.3 закрив фундаментальні можливості (публічний Series API, source-level lazy scan, production parquet backend, базові quality gates), але наступна хвиля фіч потребує системного розширення у чотирьох напрямах: аналітична виразність (SQL/window), складні типи (list/struct), cloud-lakehouse IO та контроль виконання на production.

Поточні обмеження:
- SQL підтримує лише базовий subset і не покриває CTE/window сценарії.
- Nested type workflows недостатньо формалізовані для стабільного API контракту.
- IO profile орієнтований на single-file сценарії, а не на partitioned datasets.
- Explain/perf signals існують, але не дають достатньої трасованості для release evidence.

## Goals / Non-Goals

**Goals:**
- Зафіксувати v0.4 feature-wave scope з пріоритетними capability-блоками і чіткими release gates.
- Формалізувати advanced SQL/window, nested transforms, cloud-lakehouse IO та execution observability.
- Оновити існуючі parity/conformance/optimizer/io вимоги під v0.4 профіль.
- Визначити керований шлях API evolution і migration policy для minor releases.

**Non-Goals:**
- Повне покриття всього Python Polars surface у межах одного release.
- Впровадження distributed execution або external query engine federation у v0.4.
- Гарантія zero-breaking без migration strategy, якщо parity потребує контрактних змін.

## Decisions

1. Capability-batch delivery з хвилями
- Рішення: реалізовувати v0.4 у хвилях `SQL+window`, `nested`, `cloud IO`, `observability`, `compatibility policy`.
- Чому: це дозволяє незалежно перевіряти кожен блок і знижує інтеграційний ризик.
- Альтернатива: один монолітний delivery. Відхилено через складність рев’ю та високий regression blast radius.

2. SQL/window як driver для planner/expr evolution
- Рішення: розширювати planner і expression engine під CTE/window, а не додавати ізольовані API шорткати.
- Чому: SQL сценарії примушують формалізувати єдину семантику для API та query-планування.
- Альтернатива: фокус лише на DataFrame API методах. Відхилено як недостатнє для production BI сценаріїв.

3. Nested types через explicit contracts
- Рішення: зафіксувати точні контракти для list/struct dtype, explode/flatten/map-подібних трансформацій і null semantics.
- Чому: nested функціональність без strict контракту швидко деградує у несумісні edge-cases.
- Альтернатива: best-effort підтримка без формалізації. Відхилено як джерело semantic drift.

4. Lakehouse IO profile
- Рішення: додати partition-aware scan/read contracts для multi-file dataset layouts на object storage.
- Чому: real-world датасети рідко single-file; без цього неможливо стабільно масштабувати IO сценарії.
- Альтернатива: тримати single-file profile. Відхилено як вузьке для production adoption.

5. Observability і compatibility як release contract
- Рішення: вимагати operator-level diagnostics + формалізовану versioning/deprecation policy як частину DoD.
- Чому: це зменшує ризик непрозорих регресій і неконтрольованих API змін між minor-релізами.
- Альтернатива: ad-hoc release notes. Відхилено як недостатньо перевірюване.

## Risks / Trade-offs

- [Planner complexity growth] SQL/window розширення збільшить складність parser/binder/planner → Мітигація: staged rollout і strict golden tests.
- [Nested semantics ambiguity] list/struct edge-cases можуть трактуватись неоднозначно → Мітигація: normative requirement scenarios + differential tests.
- [IO cost on cloud scans] partition-aware workflows можуть збільшити latency/cost у поганій конфігурації → Мітигація: pruning contracts і observability counters.
- [Regression surface expansion] більше фіч = більше зон регресій → Мітигація: capability coverage matrix + nightly differential suite.
- [Compatibility pressure] API evolution без policy призведе до churn → Мітигація: explicit deprecation windows і migration guarantees.

## Migration Plan

1. Затвердити capability specs і зафіксувати phase gates для v0.4.
2. Реалізувати SQL/window і nested contracts з окремими implementation changes.
3. Додати cloud-lakehouse IO profile та capability-level observability.
4. Оновити conformance/coverage/pipeline rules під v0.4 thresholds.
5. Оприлюднити compatibility policy та migration guides перед RC.

Rollback strategy:
- Якщо capability не проходить quality/perf gates, виключити її з default release profile, залишивши за feature toggle до стабілізації.

## Open Questions

- Який мінімальний SQL/window coverage потрібен для v0.4, щоб не відкладати реліз?
- Чи включати nested joins/unnest у базовий профіль v0.4 або залишити в post-v0.4 backlog?
- Який рівень object-store consistency assumptions вважати обов’язковим для cloud-lakehouse IO?
- Чи потрібен окремий LTS compatibility track після введення strict deprecation policy?
