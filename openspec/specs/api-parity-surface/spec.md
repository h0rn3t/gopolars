# api-parity-surface Specification

## Purpose
TBD - created by archiving change plan-v0-2-python-polars-parity. Update Purpose after archive.
## Requirements
### Requirement: DataFrame API SHALL match Python Polars core method surface
Система SHALL підтримувати повний набір core DataFrame методів Polars Python для вибірки, трансформацій, сортування, об’єднання, reshaping і статистичних операцій у v0.5 profile, включно з nested/list/struct workflow контрактами та advanced join/reshape операціями.

#### Scenario: Core method availability parity
- **WHEN** користувач перевіряє supported DataFrame methods проти еталонного списку Polars Python
- **THEN** система повертає повне покриття без пропусків для затвердженого v0.5 scope

#### Scenario: Nested workflow parity coverage
- **WHEN** користувач виконує nested/list/struct сценарії з трансформаціями у DataFrame API
- **THEN** система повертає сумісну семантику результатів і помилок у межах v0.5 capability profile

#### Scenario: Advanced join and reshape API parity
- **WHEN** користувач застосовує asof/semi/anti/cross join та pivot/melt у DataFrame API
- **THEN** система повертає поведінку, сумісну з Python Polars для затвердженого v0.5 scope

### Requirement: LazyFrame API SHALL preserve Polars lazy semantics
Система SHALL надавати LazyFrame API з паритетною семантикою побудови плану, deferred execution, collect/sink операцій і explain контрактів, включно з window/nested/cloud pipeline use-cases і reshape/materialization сценаріями.

#### Scenario: Deferred plan parity
- **WHEN** користувач будує еквівалентний lazy pipeline у gopolars і Polars Python
- **THEN** обидві системи виконують обчислення лише на materialization step та повертають еквівалентні результати

#### Scenario: Window and cloud lazy parity
- **WHEN** lazy pipeline містить window-операції та cloud-backed scan джерела
- **THEN** система зберігає deferred execution semantics і сумісність результатів у затвердженому v0.5 профілі

#### Scenario: Sink materialization parity
- **WHEN** lazy pipeline materialize через sink у підтримуваний формат
- **THEN** readback результат еквівалентний collect-семантиці в межах parity профілю

### Requirement: Series API SHALL provide parity for scalar, vector and namespace ops
Система SHALL підтримувати parity операцій Series, включно з numeric/string/datetime/list/struct namespaces у межах v0.5 capability matrix.

#### Scenario: Series namespace parity
- **WHEN** користувач виконує namespace-операції Series у відповідних типах
- **THEN** система повертає результати й помилки, сумісні з Polars Python semantics
