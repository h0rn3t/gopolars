## ADDED Requirements

### Requirement: DataFrame API SHALL match Python Polars core method surface
Система SHALL підтримувати повний набір core DataFrame методів Polars Python для вибірки, трансформацій, сортування, об’єднання, reshaping і статистичних операцій у v0.2 profile.

#### Scenario: Core method availability parity
- **WHEN** користувач перевіряє supported DataFrame methods проти еталонного списку Polars Python
- **THEN** система повертає повне покриття без пропусків для затвердженого v0.2 scope

### Requirement: LazyFrame API SHALL preserve Polars lazy semantics
Система SHALL надавати LazyFrame API з паритетною семантикою побудови плану, deferred execution, collect/sink операцій і explain контрактів.

#### Scenario: Deferred plan parity
- **WHEN** користувач будує еквівалентний lazy pipeline у gopolars і Polars Python
- **THEN** обидві системи виконують обчислення лише на materialization step та повертають еквівалентні результати

### Requirement: Series API SHALL provide parity for scalar, vector and namespace ops
Система SHALL підтримувати parity операцій Series, включно з numeric/string/datetime/list/struct namespaces у межах v0.2 capability matrix.

#### Scenario: Series namespace parity
- **WHEN** користувач виконує namespace-операції Series у відповідних типах
- **THEN** система повертає результати й помилки, сумісні з Polars Python semantics
