# lazy-scan-and-pushdown Specification

## Purpose
TBD - synced from change plan-v0-3-core-parity-and-performance. Update Purpose if needed.

## Requirements
### Requirement: Source-Level Lazy Scan
Система SHALL реализовывать `ScanCSV/ScanJSON/ScanParquet/ScanIPC` как source-level lazy nodes без немедленного materialize полного DataFrame.

#### Scenario: Build lazy plan without eager read
- **WHEN** разработчик вызывает `ScanParquet` и добавляет фильтр/проекцию
- **THEN** чтение данных откладывается до вызова `Collect` или `CollectStreaming`

### Requirement: Projection Pushdown
Система MUST выполнять projection pushdown на уровне scan, чтобы считывать только столбцы, требуемые в downstream выражениях.

#### Scenario: Scan reads only selected columns
- **WHEN** lazy pipeline использует подмножество колонок из источника
- **THEN** физический план чтения включает только это подмножество колонок

### Requirement: Predicate Pushdown
Система MUST применять поддерживаемые фильтры на уровне источника до materialization, когда это возможно для данного формата.

#### Scenario: Push down simple comparison predicate
- **WHEN** фильтр содержит сравнение по исходной колонке, поддерживаемое источником
- **THEN** фильтрация выполняется на этапе scan до передачи строк в последующие операторы

### Requirement: Stream-Aware Collect Semantics
Система SHALL поддерживать `CollectStreaming` для lazy pipeline с bounded-memory политикой и корректным fallback для stateful операторов.

#### Scenario: Fallback for stateful pipeline
- **WHEN** pipeline содержит stateful оператор, несовместимый с текущим streaming path
- **THEN** система выполняет безопасный fallback на стандартный execution path без потери корректности результата
