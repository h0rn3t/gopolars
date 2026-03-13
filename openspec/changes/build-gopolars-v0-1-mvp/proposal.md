## Why

Потрібна високопродуктивна бібліотека обробки даних на Go 1.26 з API, сумісним за неймінгом із Polars Python, щоб закрити потреби команд, яким потрібен native Go стек без Rust runtime у продакшн-середовищі. Зараз у проєкті відсутній цілісний фреймворк із lazy/eager виконанням, columnar зберіганням та Arrow-сумісністю.

## What Changes

- Додати MVP v0.1 DataFrame/LazyFrame/Expr API з chainable методами, де назви операцій повторюють Polars Python семантику.
- Реалізувати columnar ядро з базовими типами `int64`, `float64`, `string`, `bool`, `datetime` і null bitmap.
- Додати eager і lazy виконання з логічним/фізичним планом та базовим rule-based оптимізатором.
- Реалізувати базові операції: `select`, `filter`, `with_columns`, `group_by + agg`, `join (inner/left)`, `sort`, `limit`.
- Додати IO для `CSV`, `JSON/NDJSON`, `Parquet` та сумісність з Apache Arrow (імпорт/експорт таблиць).
- Додати SQL-like шар для базових запитів (`SELECT/FROM/WHERE/GROUP BY/HAVING/ORDER BY/LIMIT`).
- Додати тестовий та benchmark контур, CI pipeline і підготовку до публікації модуля.

## Capabilities

### New Capabilities
- `core-dataframe-api`: Публічний API DataFrame/LazyFrame/Expr/IO з chainable та type-safe підходом.
- `columnar-execution-engine`: Columnar представлення даних, фізичні оператори та паралельне виконання.
- `data-io-and-arrow`: Читання/запис CSV, JSON, Parquet та інтеграція з Apache Arrow.
- `sql-and-query-planning`: SQL frontend, побудова планів і базові оптимізації lazy execution.
- `quality-and-delivery`: Unit/integration/benchmark тести, CI/CD і релізний контур модуля.

### Modified Capabilities

Немає.

## Impact

- Кодова база: нові пакети `pkg/polars`, `pkg/series`, `pkg/frame`, `pkg/expr`, `pkg/plan`, `pkg/exec`, `pkg/io`, `pkg/sql`, `pkg/cache`.
- API: поява публічних типів `DataFrame`, `LazyFrame`, `Expr`, конфігураційних input-struct для IO/Join/Sort.
- Залежності: додавання Arrow/Parquet бібліотек, SQL parser, інструментів benchmark.
- Процеси: CI з `gofmt`, `go vet`, `go test`, `-race`, smoke benchmarks; підготовка publish workflow.
