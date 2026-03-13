# expression-parity-and-dtypes Specification

## Purpose
TBD - created by archiving change plan-v0-2-python-polars-parity. Update Purpose after archive.
## Requirements
### Requirement: Expression engine SHALL support full Polars expression repertoire
Система SHALL реалізувати expression-паритет для arithmetic, boolean, comparison, conditional, string, datetime, list, struct, null-handling і aggregation expression families, включно з розширеним v0.6 namespace surface.

#### Scenario: Expression family coverage
- **WHEN** conformance suite запускає еталонні вирази з кожної expression family
- **THEN** система успішно виконує всі вирази з результатами, сумісними з Polars Python

#### Scenario: Extended namespace parity
- **WHEN** користувач застосовує розширені string/datetime/list/struct namespace операції
- **THEN** система повертає результати та error contracts, сумісні з затвердженим v0.6 профілем

### Requirement: DType system SHALL align with Polars logical and physical types
Система SHALL підтримувати dtype модель, сумісну з Polars Python, включно з nested types, categorical/enum, decimal, temporal і nullable semantics, та забезпечувати стабільність dtype behavior у temporal-window і performance-тестових сценаріях.

#### Scenario: DType roundtrip compatibility
- **WHEN** користувач створює, кастить і серіалізує колонки всіх підтримуваних dtype
- **THEN** система зберігає очікувану dtype і значення без semantic loss

### Requirement: Null/NaN behavior SHALL match Polars semantics across operators
Система SHALL гарантувати ідентичну поведінку null/NaN у фільтрах, порівняннях, сортуванні, агрегаціях і join predicates, включно з temporal-window pipeline.

#### Scenario: Null and NaN semantic parity
- **WHEN** виконуються оператори над змішаними null/NaN наборами
- **THEN** результати, включно з ordering і aggregations, збігаються з Polars Python

