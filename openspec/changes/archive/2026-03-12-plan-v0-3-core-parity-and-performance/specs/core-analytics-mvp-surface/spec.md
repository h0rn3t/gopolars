## ADDED Requirements

### Requirement: DataFrame MVP Surface Completion
Система SHALL поддерживать MVP-критичный срез DataFrame API: `slice`, `unique/distinct`, `concat/vstack/hstack`, `fill_null`, `drop_nulls` и стабильный контракт `join` для `inner/left/right/full`.

#### Scenario: Apply unique on selected columns
- **WHEN** разработчик вызывает `unique` по подмножеству колонок
- **THEN** результат содержит уникальные строки по заданному ключу с детерминированным контрактом

### Requirement: LazyFrame Surface Alignment
Система MUST предоставлять LazyFrame-эквиваленты для MVP DataFrame операций, совместимые с optimizer/collect semantics.

#### Scenario: Build lazy plan with slice and fill-null
- **WHEN** разработчик добавляет `slice` и `fill_null` в lazy pipeline
- **THEN** операции корректно представлены в logical/physical плане и применяются при collect

### Requirement: MVP Aggregate and Join Correctness
Система SHALL обеспечивать корректность ключевых аналитических операций для MVP-сценариев: group-by aggregates, joins и сортировки с null-aware поведением.

#### Scenario: Group-by aggregate after join
- **WHEN** pipeline включает join и последующую агрегацию
- **THEN** итоговые значения и количество строк соответствуют специфицированной семантике операций
