## MODIFIED Requirements

### Requirement: Expression engine SHALL support full Polars expression repertoire
Система SHALL реалізувати expression-паритет для arithmetic, boolean, comparison, conditional, string, datetime, list, struct, null-handling і aggregation expression families, включно з v0.5 namespace edge-cases.

#### Scenario: Expression family coverage
- **WHEN** conformance suite запускає еталонні вирази з кожної expression family
- **THEN** система успішно виконує всі вирази з результатами, сумісними з Polars Python

#### Scenario: Namespace edge-case parity
- **WHEN** користувач виконує складні string/datetime/list/struct комбінації з null/NaN
- **THEN** система повертає детерміновані результати й помилки, сумісні з Polars Python contracts

### Requirement: DType system SHALL align with Polars logical and physical types
Система SHALL підтримувати dtype модель, сумісну з Polars Python, включно з nested types, categorical/enum, decimal, temporal і nullable semantics, та гарантувати узгодженість dtype у reshape/join/sink сценаріях.

#### Scenario: DType roundtrip compatibility
- **WHEN** користувач створює, кастить і серіалізує колонки всіх підтримуваних dtype
- **THEN** система зберігає очікувану dtype і значення без semantic loss

#### Scenario: DType stability in reshape and sink
- **WHEN** дані проходять pivot/melt або sink+readback цикл
- **THEN** система зберігає валідні dtype contracts без непередбачуваного widening/losing semantics

### Requirement: Null/NaN behavior SHALL match Polars semantics across operators
Система SHALL гарантувати ідентичну поведінку null/NaN у фільтрах, порівняннях, сортуванні, агрегаціях і join predicates, включно з advanced join і reshape pipeline.

#### Scenario: Null and NaN semantic parity
- **WHEN** виконуються оператори над змішаними null/NaN наборами
- **THEN** результати, включно з ordering і aggregations, збігаються з Polars Python
