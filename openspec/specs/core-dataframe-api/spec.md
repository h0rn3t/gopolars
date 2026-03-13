## Purpose
Визначає MVP v0.1 вимоги до публічного DataFrame/LazyFrame/Expr API.

## Requirements

### Requirement: DataFrame API SHALL provide Polars-aligned core operations
Система SHALL надавати публічний DataFrame API з chainable методами `Select`, `Filter`, `WithColumns`, `Sort`, `Limit`, `Join`, `GroupBy().Agg()` з семантикою, сумісною з Polars Python.

#### Scenario: Chainable eager pipeline
- **WHEN** користувач послідовно викликає `Select`, `Filter`, `Sort`, `Limit`
- **THEN** кожний виклик повертає новий DataFrame без мутації попереднього екземпляра

### Requirement: LazyFrame API SHALL build a deferred execution plan
Система SHALL надавати `LazyFrame` з методами `Select`, `Filter`, `WithColumns`, `Join`, `GroupBy().Agg()`, `Sort`, `Limit`, `Collect`, `Explain`, де обчислення виконується тільки під час `Collect`.

#### Scenario: Lazy operations are deferred
- **WHEN** користувач викликає ланцюжок lazy-методів без `Collect`
- **THEN** система створює логічний план без матеріалізації результату

### Requirement: Expression API SHALL be type-safe and composable
Система SHALL підтримувати `Expr` конструктори `Col`, `Lit`, арифметичні, логічні та порівняльні операції, а також `Alias`, `Cast`, `IsNull`, `IsNotNull`.

#### Scenario: Expression composition for filter and projection
- **WHEN** користувач передає комбіновані `Expr` у `Filter` та `Select`
- **THEN** система коректно обчислює вирази та зберігає імена колонок згідно з `Alias`
