## MODIFIED Requirements

### Requirement: SQL Layer SHALL support advanced analytical constructs
Система SHALL добавить SQLContext `register` flow как обязательный baseline для v0.7 Top-30.

#### Scenario: Register DataFrame in SQLContext
- **WHEN** пользователь регистрирует DataFrame в SQLContext и запускает запрос
- **THEN** система корректно выполняет запрос и соблюдает контракт привязки таблицы

### Requirement: Window expressions SHALL be available in SQL and API pipelines
Система SHALL синхронизировать новые Expr window функции (`over`, `rank`, `cum_*`, `rolling*`) между SQL и API semantics.

#### Scenario: SQL/API window equivalence for Top-30
- **WHEN** эквивалентная аналитика выполняется через SQL и через Expr API
- **THEN** результаты и semantics остаются совместимыми
