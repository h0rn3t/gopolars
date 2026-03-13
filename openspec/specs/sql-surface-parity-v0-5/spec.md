# sql-surface-parity-v0-5 Specification

## Purpose
TBD - synced from change plan-v0-5-python-polars-parity-wave-2. Update Purpose if needed.

## Requirements
### Requirement: SQL engine SHALL support subqueries and set operations
Система SHALL поддерживать SQL subqueries и set-операции (`UNION`, `INTERSECT`, `EXCEPT`) в пределах v0.5 parity profile.

#### Scenario: Execute query with subquery and union
- **WHEN** пользователь выполняет SQL-запрос с подзапросом и `UNION`
- **THEN** система возвращает корректный результат с ожидаемыми правилами dedup и schema alignment

### Requirement: SQL analytical surface SHALL align with expression and window semantics
Система MUST обеспечивать согласованность SQL аналитических выражений с API expression/window контрактами.

#### Scenario: Validate SQL and API semantic equivalence
- **WHEN** эквивалентная аналитика выполняется через SQL и через API pipeline
- **THEN** результаты и error-поведение совпадают в рамках утверждённого parity scope

### Requirement: SQL parity evidence SHALL be tracked by conformance fixtures
Система SHALL иметь отдельный conformance набор для SQL v0.5 surface с дифференциальной проверкой против Python Polars.

#### Scenario: Run SQL differential suite
- **WHEN** CI запускает SQL differential fixtures
- **THEN** любое расхождение семантики фиксируется как failure с отчётом по несовпадениям
