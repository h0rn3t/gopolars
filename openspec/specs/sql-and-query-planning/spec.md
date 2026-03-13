## Purpose
Фіксує вимоги SQL frontend, планування запитів та оптимізації lazy execution для MVP v0.1.

## Requirements

### Requirement: SQL frontend SHALL support baseline analytical query subset
Система SHALL підтримувати SQL-запити класу `SELECT`, `FROM`, `WHERE`, `GROUP BY`, `HAVING`, `ORDER BY`, `LIMIT` для v0.1.

#### Scenario: Execute grouped SQL query
- **WHEN** користувач виконує SQL запит із `GROUP BY` та агрегатами
- **THEN** система повертає еквівалентний результат до API-пайплайна на `LazyFrame`

### Requirement: Planner SHALL map SQL and API to a unified logical plan
Система SHALL перетворювати SQL і API-вирази в єдиний logical plan формат перед фізичним виконанням.

#### Scenario: Equal semantics for SQL and API
- **WHEN** однакова трансформація виражена через SQL і через `Expr` API
- **THEN** система будує еквівалентні logical план-структури з однаковою семантикою

### Requirement: Optimizer SHALL apply baseline rule-based rewrites
Система SHALL застосовувати правила `predicate pushdown`, `projection pruning` і `constant folding` до lazy logical плану перед фізичним виконанням.

#### Scenario: Predicate pushdown on lazy scan
- **WHEN** lazy план містить `Filter` після `Scan`
- **THEN** оптимізатор переміщує предикат максимально близько до scan-оператора без зміни результату
