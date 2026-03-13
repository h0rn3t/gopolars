## MODIFIED Requirements

### Requirement: IO layer SHALL provide parity across primary tabular formats
Система SHALL підтримувати parity-read/write для CSV, JSON/NDJSON, Parquet, IPC/Feather з projection, schema control і performant scan paths, включно з multi-file dataset режимом для v0.4.

#### Scenario: Multi-format roundtrip parity
- **WHEN** користувач читає і записує один датасет у різних форматах
- **THEN** система зберігає сумісні значення, dtype і nullable semantics відносно Polars Python

#### Scenario: Multi-file dataset parity
- **WHEN** користувач працює з dataset, що складається з множини файлів одного формату
- **THEN** система виконує узгоджене читання/запис із семантично стабільним результатом у v0.4 профілі

### Requirement: System SHALL support cloud object-store data access
Система SHALL підтримувати роботу з object storage backends (S3/GCS/Azure) для scan/read/write сценаріїв з credential-safe configuration і partition-aware pruning contracts.

#### Scenario: Cloud parquet scan parity
- **WHEN** користувач виконує ScanParquet з object storage URI
- **THEN** система читає дані з еквівалентною поведінкою projection/predicate pushdown

#### Scenario: Partition-aware cloud pruning
- **WHEN** cloud dataset має partition layout і запит містить відповідний предикат
- **THEN** система читає лише релевантні partition сегменти та надає діагностику pruning рішень
