## Context

После завершения v0.2 у проекта есть базовый parity-контур, но остаются системные ограничения для production-использования:
- отсутствует публичный `Series` API на уровне `pkg/polars`;
- `Scan*` остаётся псевдо-lazy (чтение происходит до построения lazy-плана);
- parquet слой не соответствует production-ожиданиям и требует перехода на `parquet-go v0.29.0`;
- quality/performance gates недостаточно строгие для устойчивого развития API parity.

Текущая архитектура уже имеет правильный каркас (`logical -> optimizer -> physical -> scheduler`), поэтому основной дизайн-фокус — расширение источников данных, API-поверхности и governance метрик без слома существующего контракта.

Внешние ограничения:
- совместимость с Arrow `23.0.1` и корректный type/null mapping;
- idiomatic Go реализация без копирования внутренних деталей Rust Polars;
- сохранение наблюдаемой semantic parity (поведение, а не внутренние структуры).

## Goals / Non-Goals

**Goals:**
- Определить технические решения для MVP v0.3 по DataFrame/Series/LazyFrame.
- Зафиксировать архитектуру source-level lazy scan + pushdown.
- Спроектировать production-ready parquet backend на `parquet-go v0.29.0`.
- Установить измеримые quality/performance/conformance gates для CI/release.
- Зафиксировать обязательный стандарт developer-документации и примеров.

**Non-Goals:**
- Полная реализация всех Polars namespaces в рамках одного change.
- Байт-в-байт совместимость с внутренними планами/типами Rust Polars.
- Миграция всех исторических API без потенциальных breaking alignment изменений.

## Decisions

1. Capability-first delivery для v0.3
- Решение: делить реализацию на отдельные capability-потоки (`series-public-api`, `lazy-scan-and-pushdown`, `parquet-production-backend`, `core-analytics-mvp-surface`, `quality-and-performance-gates-v0-3`, `developer-docs-and-examples-v0-3`).
- Почему: уменьшает интеграционный риск и ускоряет независимую проверку прогресса.
- Альтернатива: единый монолитный implementation change. Отклонено из-за слабой трассируемости.

2. Source-level lazy execution как основной путь
- Решение: `Scan*` должны формировать source node в логическом плане, а чтение данных переносится на этап исполнения.
- Почему: это prerequisite для projection/predicate pushdown и stream-aware исполнения.
- Альтернатива: сохранить eager `Read* -> Lazy()`. Отклонено как непаритетное и неэффективное.

3. Явная Arrow/Parquet compatibility policy
- Решение: закрепить stack-версии (`Arrow 23.0.1`, `parquet-go v0.29.0`) и единый контракт типизации/nullability.
- Почему: устраняет семантическую дрожь между форматами и упрощает differential tests.
- Альтернатива: “latest available” без pinning. Отклонено из-за нестабильности для CI и релизов.

4. Quality gates как release contract
- Решение: добавить обязательные пороги для coverage, parity coverage, race/stability, benchmark regression.
- Почему: превращает качество из “best effort” в проверяемый контракт.
- Альтернатива: ручная quality-проверка перед релизом. Отклонено как нереплицируемо.

5. Docs-as-feature для developer readiness
- Решение: считать документацию и runnable-примеры частью definition-of-done capability, а не post-factum задачей.
- Почему: снижает стоимость внедрения библиотеки и число интеграционных ошибок.
- Альтернатива: писать docs после реализации. Отклонено как источник задержек и деградации DX.

## Risks / Trade-offs

- [API churn] Потенциальные breaking alignment изменения в Series/Lazy API → Митигация: migration notes, compatibility aliases, фиксированная deprecation policy.
- [Perf regressions] Новые возможности могут увеличивать allocs/latency → Митигация: benchmark budget + CI regression gates.
- [Format mismatch] Различия в трактовке logical types между Arrow/Parquet → Митигация: единый mapping layer и roundtrip tests.
- [Scope creep] MVP может разрастись до “почти full parity” → Митигация: жёсткий MVP-срез и phase exit criteria.
- [Operational complexity] Ужесточение CI увеличит время pipeline → Митигация: split-jobs, smoke/full режимы, nightly расширенные прогоны.

## Migration Plan

1. Утвердить capability specs и сформировать implementation backlog по фазам.
2. Внедрить source-level lazy scan и parquet backend как инфраструктурный фундамент.
3. Расширить публичные DataFrame/Series/LazyFrame контракты до MVP v0.3 surface.
4. Включить quality/perf gates в CI и release workflow с обязательными порогами.
5. Подготовить migration notes, docs и runnable examples, затем выполнить release-candidate audit.

Rollback strategy:
- При провале ключевых quality/perf/parity gate релизный профиль сужается до стабильного подмножества через feature toggle и документированный fallback behavior.

## Open Questions

- Какой минимальный набор Series namespace функций считается обязательным для “developer-ready” v0.3?
- Нужно ли включать `asof` join в MVP или оставить в post-MVP roadmap?
- Какая стратегия приоритета у streaming path при конфликте с strict ordering semantics?
- Должен ли differential suite с Python Polars быть обязательным в каждом PR или только в nightly/release?
