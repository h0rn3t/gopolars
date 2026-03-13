## MODIFIED Requirements

### Requirement: Explain diagnostics SHALL expose operator-level execution signals
Система SHALL надавати explain diagnostics із інформацією про оператори, pushdown decisions, streaming fallback і performance markers для window-heavy pipelines.

#### Scenario: Inspect execution diagnostics for optimized plan
- **WHEN** користувач запитує explain у діагностичному режимі
- **THEN** система повертає повний перелік operator stages і прийнятих оптимізаційних рішень

### Requirement: Runtime telemetry SHALL track performance budget indicators
Система MUST збирати runtime signals для latency, memory/allocations і volume метрик на рівні pipeline/операторів, включно з temporal-window execution traces.

#### Scenario: Record metrics for benchmark pipeline
- **WHEN** запускається еталонний benchmark pipeline
- **THEN** система публікує метрики, придатні для порівняння з baseline budget

### Requirement: Observability output SHALL be stable for automation
Система SHALL гарантувати стабільний формат observability output, який може використовуватись у CI і release evidence, включно з v0.6 performance-hardening gates.

#### Scenario: Validate diagnostics schema in CI
- **WHEN** CI виконує перевірку observability contract
- **THEN** формат diagnostics залишається сумісним із автоматичною валідацією
