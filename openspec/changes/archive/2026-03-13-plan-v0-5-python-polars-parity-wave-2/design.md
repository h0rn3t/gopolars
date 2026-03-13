## Context

gopolars прошёл фундаментальные этапы v0.1–v0.4 и уже покрывает значимую часть DataFrame/Lazy/IO функциональности. Однако при миграции production-пайплайнов с Python Polars остаются пробелы: advanced joins (включая time-aware asof), reshape-паттерны (pivot/unpivot), более глубокий SQL surface, а также стандартизация lazy sink/materialize контрактов.

Текущий стек архитектурно готов к расширению, но следующие parity-фичи затрагивают сразу несколько слоёв (API, planner, optimizer, execution, IO, conformance). Это требует согласованного feature-wave дизайна, чтобы избежать semantic drift и сохранить стабильность релизов.

## Goals / Non-Goals

**Goals:**
- Расширить parity-coverage к Python Polars по high-impact сценариям: joins, reshape, namespaces и SQL.
- Формализовать lazy sink/materialization поведение в едином контракте для eager/lazy/streaming путей.
- Усилить conformance evidence и compatibility governance для контролируемых alignment изменений.
- Сохранить проверяемую и детерминированную семантику при расширении optimizer/execution.

**Non-Goals:**
- Достичь 100% полную parity по всем API Python Polars за один release wave.
- Вводить distributed execution/runtime federation в рамках v0.5.
- Поддерживать нестабильные или экспериментальные API без compatibility policy и migration guidance.

## Decisions

1. Feature-wave с фокусом на migration-critical сценарии
- Решение: приоритизировать advanced joins, reshape, SQL parity и sink/materialization раньше long-tail API additions.
- Почему: это даёт максимальный эффект для реальных migration pipeline.
- Альтернатива: равномерно расширять все namespaces понемногу. Отклонено из-за размытой ценности и сложного контроля регрессий.

2. Parity через единые семантические контракты
- Решение: для каждого блока формализовать normative behavior (result semantics + error contract + null/NaN rules).
- Почему: без strict-контрактов parity быстро расходится между eager/lazy/sql путями.
- Альтернатива: best-effort compatibility. Отклонено как непредсказуемое для production adoption.

3. SQL surface как отдельный capability-трек
- Решение: вынести SQL parity v0.5 в отдельную capability, но синхронизировать с API/expr контрактами.
- Почему: SQL и API должны сходиться на одном планировщике и общей execution семантике.
- Альтернатива: развивать SQL только через побочные API улучшения. Отклонено как недостаточно управляемое.

4. Lazy sink/materialization как release-критичный контракт
- Решение: определить единые правила для sink parquet/csv/ipc и collect/materialize paths.
- Почему: без этого сложно гарантировать predictable behavior и reproducible outputs в batch workflows.
- Альтернатива: оставить sink как ad-hoc реализацию на уровне IO. Отклонено из-за риска несовместимых путей.

5. Compatibility-first governance
- Решение: для потенциально breaking alignment изменений обязательно требовать deprecation windows, migration notes и release evidence.
- Почему: parity прогресс не должен ломать существующих пользователей без управляемого перехода.
- Альтернатива: фиксировать несовместимости только в changelog. Отклонено как слабый процесс контроля.

## Risks / Trade-offs

- [Рост сложности planner/optimizer] Больше SQL/join/reshape правил → Митигация: staged rollout, golden plans и differential fixtures.
- [Semantic drift между eager/lazy/sql] Разные execution-пути могут расходиться → Митигация: capability-level parity tests для всех трёх путей.
- [Регрессии производительности] Advanced joins и reshape увеличивают cost сложных запросов → Митигация: benchmark budget gates + operator telemetry.
- [Migration churn] Новые alignment контракты могут потребовать адаптацию кода → Митигация: deprecation windows и обязательные migration notes.
- [Cloud IO edge-cases] Sink/dataset сценарии могут быть нестабильны на разных storage-профилях → Митигация: integration matrix и diagnostics evidence.

## Migration Plan

1. Утвердить capability specs и определить v0.5 release gates.
2. Реализовать advanced joins + reshape контракты с unit/conformance покрытием.
3. Расширить SQL surface parity и синхронизировать с planner/optimizer behavior.
4. Добавить lazy sink/materialization контракты и интеграционные проверки по форматам.
5. Обновить compatibility/conformance gates и выпустить migration documentation.

Rollback strategy:
- Если отдельная capability не проходит quality/perf/compatibility gates, она исключается из default v0.5 profile и остаётся за feature-toggle до стабилизации.

## Open Questions

- Какой минимальный coverage threshold для advanced joins/reshape достаточен для v0.5 RC?
- Какие SQL конструкции считать обязательными для parity milestone v0.5, а какие вынести в v0.6?
- Нужен ли единый sink transaction contract для всех форматов или достаточно format-specific guarantees?
- Какой acceptable performance budget по asof/pivot сценариям относительно Python Polars baseline?
