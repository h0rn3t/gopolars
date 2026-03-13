# advanced-joins-and-time-series Specification

## Purpose
TBD - synced from change plan-v0-5-python-polars-parity-wave-2. Update Purpose if needed.

## Requirements
### Requirement: Join engine SHALL support advanced relational join modes
Система SHALL поддерживать `semi`, `anti`, `cross` и расширенные join-паттерны на уровне DataFrame/LazyFrame API с семантикой, совместимой с Python Polars.

#### Scenario: Execute semi and anti join
- **WHEN** пользователь выполняет `semi` или `anti` join по ключам
- **THEN** результат содержит корректный набор строк согласно семантике фильтрации Python Polars

### Requirement: System SHALL support asof joins for ordered time-series
Система MUST поддерживать `asof` join с правилами направления и tolerance для time-aware аналитики.

#### Scenario: Match nearest timestamp with tolerance
- **WHEN** пользователь выполняет `asof` join по упорядоченной временной колонке с tolerance
- **THEN** система выбирает корректное ближайшее соответствие или возвращает null при отсутствии допустимой пары

### Requirement: Advanced join behavior SHALL remain deterministic across eager and lazy paths
Система SHALL обеспечивать эквивалентность результатов advanced joins между eager и lazy execution путями.

#### Scenario: Compare eager and lazy advanced join outputs
- **WHEN** один и тот же advanced join pipeline запускается в eager и lazy режимах
- **THEN** обе версии возвращают эквивалентные строки, schema и null-семантику
