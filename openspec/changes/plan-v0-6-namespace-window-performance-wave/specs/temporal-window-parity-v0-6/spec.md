## ADDED Requirements

### Requirement: System SHALL support group_by_dynamic with Polars-compatible temporal semantics
Система SHALL поддерживать `group_by_dynamic` с корректными boundary/offset/closed-интервалами и детерминированной window классификацией.

#### Scenario: Build dynamic windows on event stream
- **WHEN** пользователь запускает `group_by_dynamic` на временном ряду с заданными параметрами окна
- **THEN** система формирует окна и агрегаты, совместимые с Python Polars semantics

### Requirement: Rolling analytics SHALL align across eager and lazy execution paths
Система MUST обеспечивать эквивалентную rolling-семантику в eager и lazy режимах, включая null-handling и ordering behavior.

#### Scenario: Compare eager and lazy rolling outputs
- **WHEN** эквивалентный rolling pipeline выполняется в eager и lazy режимах
- **THEN** результаты совпадают по значениям, schema и edge-case поведению

### Requirement: Temporal window operations SHALL expose explainable diagnostics
Система SHALL публиковать explain/diagnostics сигналы по temporal window execution (window cardinality, fallback path, stateful markers).

#### Scenario: Inspect temporal execution diagnostics
- **WHEN** пользователь запрашивает diagnostics для temporal pipeline
- **THEN** система возвращает структурированные метрики и признаки выбранного execution path
