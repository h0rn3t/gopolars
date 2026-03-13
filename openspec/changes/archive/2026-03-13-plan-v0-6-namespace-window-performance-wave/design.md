## Context

v0.5 закрыл большую часть high-impact API parity, но для устойчивой замены Python Polars в production остаются три системные зоны: (1) глубокое namespace parity, (2) temporal analytics parity (`group_by_dynamic`, расширенный rolling), (3) performance predictability под stateful и mixed workloads.

Текущая архитектура уже поддерживает lazy/explain/diagnostics и adaptive элементы, однако поведение temporal windows и нагрузочное масштабирование требуют более жёстких контрактов и расширенных quality gates.

## Goals / Non-Goals

**Goals:**
- Довести namespace parity до уровня v0.6 compatibility profile с формализованными edge-case контрактами.
- Ввести полноценный temporal-window контракт для `group_by_dynamic`/`rolling` в eager/lazy/sql путях.
- Усилить performance hardening через budget gates, regression detection и runtime diagnostics.
- Обеспечить проверяемую release evidence цепочку для v0.6.

**Non-Goals:**
- Полная 100% parity по всему Python Polars API в рамках одного wave.
- Реализация distributed execution/cluster runtime.
- Введение нестабильных experimental API без compatibility governance.

## Decisions

1. Capability-driven v0.6 scope
- Решение: сфокусировать wave на трёх производственных осях (namespace, temporal windows, performance), вместо широкого long-tail расширения.
- Альтернатива: распылить усилия на много среднеприоритетных API. Отклонено как менее эффективно для migration readiness.

2. Temporal windows как отдельный capability
- Решение: вынести temporal window parity в отдельную capability и связать её со стандартным analytics/windowing spec.
- Альтернатива: добавить требования фрагментарно в текущий analytics spec. Отклонено из-за снижения трассируемости.

3. Performance governance как first-class contract
- Решение: ввести capability `performance-hardening-and-budgets` с критериями budget, telemetry, regression gates.
- Альтернатива: оставить performance в рамках ad-hoc benchmark практик. Отклонено как недостаточно надёжно для release gating.

4. Diagnostics-first delivery
- Решение: расширять observability параллельно с функциональностью, чтобы каждое stateful изменение имело explain/telemetry evidence.
- Альтернатива: диагностировать постфактум. Отклонено как риск late regression discovery.

## Risks / Trade-offs

- [Рост сложности планировщика] Больше temporal/window правил → Митигация: staged rollout и capability-level differential fixtures.
- [Performance regressions] Stateful windows могут ухудшать latency/memory → Митигация: обязательные perf budgets и telemetry gates.
- [Semantic drift между API/SQL/lazy] Разные execution-пути могут расходиться → Митигация: cross-path parity suites.
- [Migration friction] Alignment fixes могут менять edge semantics → Митигация: deprecation windows + migration docs.

## Migration Plan

1. Утвердить proposal/specs/tasks для v0.6.
2. Реализовать namespace parity пакет с edge-case conformance.
3. Реализовать temporal window контракты (`group_by_dynamic`, rolling).
4. Ввести performance hardening capability и CI gates.
5. Обновить observability, compatibility evidence и release documentation.

Rollback strategy:
- Любая capability, не проходящая conformance/performance gates, исключается из default v0.6 profile и остаётся behind flag до стабилизации.

## Open Questions

- Какой минимальный performance budget threshold считать приемлемым для v0.6 RC?
- Какие temporal-window edge-cases являются blocking для release, а какие можно вынести в v0.7?
- Нужна ли отдельная degraded-mode политика для heavy window workloads при memory pressure?
