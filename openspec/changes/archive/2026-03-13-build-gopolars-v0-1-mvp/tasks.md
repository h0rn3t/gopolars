## 1. Архітектурний каркас і контракти API

- [x] 1.1 Створити структуру пакетів `pkg/polars`, `pkg/dtypes`, `pkg/series`, `pkg/frame`, `pkg/expr`, `pkg/plan`, `pkg/exec`, `pkg/io`, `pkg/sql`, `pkg/cache`
- [x] 1.2 Описати публічні контракти `DataFrame`, `LazyFrame`, `Expr`, `IO` з Polars-aligned неймінгом
- [x] 1.3 Додати input-struct контракти для `Join`, `Sort`, IO read/scan/write операцій

## 2. Columnar ядро і базові оператори

- [x] 2.1 Реалізувати columnar серії з типами `int64`, `float64`, `string`, `bool`, `datetime` і null bitmap
- [x] 2.2 Реалізувати векторизовані оператори `Select`, `Filter`, `WithColumns`, `Sort`, `Limit`
- [x] 2.3 Реалізувати `GroupBy().Agg()` з `sum`, `min`, `max`, `mean`, `count`, `n_unique`
- [x] 2.4 Реалізувати `Join` режими `inner` і `left` для ключів однакового типу

## 3. Lazy execution і оптимізатор

- [x] 3.1 Реалізувати logical plan вузли для scan/projection/filter/join/aggregate/sort/limit
- [x] 3.2 Реалізувати physical plan і scheduler паралельного виконання через goroutines
- [x] 3.3 Додати оптимізації `predicate pushdown`, `projection pruning`, `constant folding`
- [x] 3.4 Реалізувати `Collect` та `Explain` для LazyFrame

## 4. IO і Apache Arrow інтеграція

- [x] 4.1 Додати eager read/write для CSV і JSON/NDJSON з schema inference та schema override
- [x] 4.2 Додати eager `ReadParquet` і lazy `ScanParquet` з column projection
- [x] 4.3 Додати експорт/імпорт Arrow table для сумісності екосистеми
- [x] 4.4 Додати інтеграційні тести roundtrip для CSV/JSON/Parquet/Arrow

## 5. SQL frontend і уніфікація семантики

- [x] 5.1 Реалізувати parser/binder/planner для `SELECT/FROM/WHERE/GROUP BY/HAVING/ORDER BY/LIMIT`
- [x] 5.2 Забезпечити перетворення SQL у той самий Expr AST і logical plan, що використовує API
- [x] 5.3 Додати golden-тести на еквівалентність SQL і API сценаріїв

## 6. Якість, продуктивність і постачання

- [x] 6.1 Додати unit та integration тести для ядра, IO, lazy/eager і SQL сценаріїв
- [x] 6.2 Додати benchmark-набір micro і macro з метриками часу та алокацій
- [x] 6.3 Налаштувати CI workflow з `gofmt`, `go vet`, `go test`, `go test -race`, benchmark smoke
- [x] 6.4 Підготувати релізний workflow модуля, README з прикладами та MVP матрицю підтримки
