## ADDED Requirements

### Requirement: Reshape layer SHALL support pivot and unpivot/melt workflows
Система SHALL поддерживать операции `pivot` и `unpivot/melt` с контролируемой агрегацией и совместимой schema-семантикой.

#### Scenario: Build pivot table with aggregation
- **WHEN** пользователь выполняет pivot по key/value колонкам с агрегатором
- **THEN** система возвращает корректную сводную таблицу с ожидаемой структурой и значениями

### Requirement: Explode/unnest reshape scenarios SHALL preserve row alignment contracts
Система MUST обеспечивать предсказуемые правила row alignment при explode/unnest-трансформациях nested данных.

#### Scenario: Unnest nested column with sibling columns
- **WHEN** пользователь разворачивает nested колонку в присутствии связанных колонок
- **THEN** система сохраняет детерминированное соответствие строк и null-поведение

### Requirement: Reshape operations SHALL be available in lazy pipelines
Система SHALL поддерживать reshape-операции в lazy execution с корректной интеграцией в optimizer/planner.

#### Scenario: Execute lazy melt pipeline
- **WHEN** melt-пайплайн выполняется через lazy collect
- **THEN** результат эквивалентен eager-варианту и проходит parity-проверку
