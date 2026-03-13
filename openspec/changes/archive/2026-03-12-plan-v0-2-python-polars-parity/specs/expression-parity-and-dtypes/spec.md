## ADDED Requirements

### Requirement: Expression engine SHALL support full Polars expression repertoire
Система SHALL реалізувати expression-паритет для arithmetic, boolean, comparison, conditional, string, datetime, list, struct, null-handling і aggregation expression families.

#### Scenario: Expression family coverage
- **WHEN** conformance suite запускає еталонні вирази з кожної expression family
- **THEN** система успішно виконує всі вирази з результатами, сумісними з Polars Python

### Requirement: DType system SHALL align with Polars logical and physical types
Система SHALL підтримувати dtype модель, сумісну з Polars Python, включно з nested types, categorical/enum, decimal, temporal і nullable semantics.

#### Scenario: DType roundtrip compatibility
- **WHEN** користувач створює, кастить і серіалізує колонки всіх підтримуваних dtype
- **THEN** система зберігає очікувану dtype і значення без semantic loss

### Requirement: Null/NaN behavior SHALL match Polars semantics across operators
Система SHALL гарантувати ідентичну поведінку null/NaN у фільтрах, порівняннях, сортуванні, агрегаціях і join predicates.

#### Scenario: Null and NaN semantic parity
- **WHEN** виконуються оператори над змішаними null/NaN наборами
- **THEN** результати, включно з ordering і aggregations, збігаються з Polars Python
