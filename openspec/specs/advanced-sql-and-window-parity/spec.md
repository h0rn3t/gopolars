# advanced-sql-and-window-parity Specification

## Purpose
TBD - synced from change plan-v0-4-next-feature-wave. Update Purpose if needed.

## Requirements
### Requirement: SQL Layer SHALL support advanced analytical constructs
Система SHALL підтримувати v0.4 SQL-профіль із CTE, розширеними aggregate expressions і валідаційно-сумісним плануванням для аналітичних сценаріїв.

#### Scenario: Execute query with common table expression
- **WHEN** користувач виконує запит із `WITH`-блоком і подальшою агрегацією
- **THEN** планер будує коректний logical plan і повертає семантично валідний результат

### Requirement: Window expressions SHALL be available in SQL and API pipelines
Система MUST підтримувати window expressions з `PARTITION BY` та `ORDER BY` семантикою у SQL шарі та еквівалентних API сценаріях.

#### Scenario: Compute partitioned running aggregate
- **WHEN** користувач виконує віконну агрегацію по partition ключу з порядком
- **THEN** результат містить коректні значення для кожного рядка в межах partition

### Requirement: SQL parity evidence SHALL be testable
Система SHALL мати набір перевірок, що доводить semantic parity SQL/window поведінки відносно затвердженого profile.

#### Scenario: Validate SQL/window fixtures in conformance suite
- **WHEN** CI запускає SQL/window conformance fixtures
- **THEN** будь-яке відхилення від затвердженої семантики детектується як failure
