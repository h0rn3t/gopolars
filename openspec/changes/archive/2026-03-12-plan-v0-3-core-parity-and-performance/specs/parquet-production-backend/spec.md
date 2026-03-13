## ADDED Requirements

### Requirement: Production Parquet Read Path
Система MUST реализовать реальное чтение Parquet через `parquet-go v0.29.0` с поддержкой column projection и корректной типизации.

#### Scenario: Read subset of columns from parquet
- **WHEN** разработчик запрашивает ограниченный набор колонок при чтении parquet
- **THEN** backend считывает и возвращает только указанные колонки

### Requirement: Production Parquet Write Path
Система SHALL записывать parquet-файлы через `parquet-go v0.29.0` с корректной сериализацией поддерживаемых dtype и null значений.

#### Scenario: Write and read back dataset with nulls
- **WHEN** dataset с null-значениями записывается и затем считывается обратно
- **THEN** schema и null-позиции сохраняют эквивалентную семантику

### Requirement: Logical Type and Timestamp Contract
Система MUST иметь документированный и тестируемый контракт маппинга logical types (включая timestamp и decimal) между внутренними dtype и parquet представлением.

#### Scenario: Preserve timestamp semantics
- **WHEN** timestamp-колонка записывается и считывается в parquet
- **THEN** значения времени и порядок элементов сохраняются без семантической деградации

### Requirement: Version-Pinned Dependency Policy
Система SHALL фиксировать целевую версию parquet backend (`v0.29.0`) и иметь процесс контролируемого обновления.

#### Scenario: Validate backend version in build metadata
- **WHEN** выполняется release pipeline
- **THEN** подтверждается соответствие используемой parquet зависимости утверждённой версии
