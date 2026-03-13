## MODIFIED Requirements

### Requirement: DataFrame API SHALL match Python Polars core method surface
Система SHALL підтримувати повний набір core DataFrame методів Polars Python для вибірки, трансформацій, сортування, об’єднання, reshaping і статистичних операцій у v0.4 profile, включно з nested/list/struct workflow контрактами.

#### Scenario: Core method availability parity
- **WHEN** користувач перевіряє supported DataFrame methods проти еталонного списку Polars Python
- **THEN** система повертає повне покриття без пропусків для затвердженого v0.4 scope

#### Scenario: Nested workflow parity coverage
- **WHEN** користувач виконує nested/list/struct сценарії з трансформаціями у DataFrame API
- **THEN** система повертає сумісну семантику результатів і помилок у межах v0.4 capability profile

### Requirement: LazyFrame API SHALL preserve Polars lazy semantics
Система SHALL надавати LazyFrame API з паритетною семантикою побудови плану, deferred execution, collect/sink операцій і explain контрактів, включно з window/nested/cloud pipeline use-cases.

#### Scenario: Deferred plan parity
- **WHEN** користувач будує еквівалентний lazy pipeline у gopolars і Polars Python
- **THEN** обидві системи виконують обчислення лише на materialization step та повертають еквівалентні результати

#### Scenario: Window and cloud lazy parity
- **WHEN** lazy pipeline містить window-операції та cloud-backed scan джерела
- **THEN** система зберігає deferred execution semantics і сумісність результатів у затвердженому v0.4 профілі
