## ADDED Requirements

### Requirement: IO SHALL support CSV and JSON read/write in eager mode
Система SHALL надавати `ReadCSV`, `ReadJSON`, `WriteCSV`, `WriteJSON` для eager DataFrame з підтримкою schema inference і явного перевизначення схеми.

#### Scenario: Read and write CSV roundtrip
- **WHEN** користувач читає CSV у DataFrame і записує результат назад у CSV
- **THEN** система зберігає структуру колонок і типи у межах підтримки v0.1

### Requirement: IO SHALL support Parquet read and lazy scan
Система SHALL підтримувати eager `ReadParquet` і lazy `ScanParquet` з column projection для обмеження читання даних.

#### Scenario: Parquet scan with selected columns
- **WHEN** користувач виконує `ScanParquet` і вибирає підмножину колонок
- **THEN** система читає тільки потрібні колонки з джерела

### Requirement: System SHALL provide Apache Arrow interoperability
Система SHALL підтримувати експорт DataFrame у Arrow табличний формат і імпорт із Arrow структури у DataFrame для міжсистемної сумісності.

#### Scenario: Arrow export/import compatibility
- **WHEN** користувач експортує DataFrame у Arrow і повторно імпортує його
- **THEN** система відтворює значення, типи та nullable-стан у межах підтримуваних типів
