## 1. Series Public API

- [x] 1.1 Спроєктувати публічний `polars.Series` контракт і узгодити його з поточним `pkg/series`
- [x] 1.2 Реалізувати null-aware MVP операції Series (`is_null`, `fill_null`, `drop_nulls`, cast)
- [x] 1.3 Додати векторні арифметичні та порівняльні операції Series з валідацією довжини

## 2. Lazy Scan and Pushdown

- [x] 2.1 Перевести `Scan*` на source-level lazy nodes без eager materialization
- [x] 2.2 Реалізувати projection pushdown для CSV/JSON/Parquet/IPC scan шляхів
- [x] 2.3 Реалізувати predicate pushdown для підтримуваних умов і fallback при unsupported predicates

## 3. Parquet Production Backend

- [x] 3.1 Інтегрувати `parquet-go v0.29.0` у read/write шлях замість поточного stub формату
- [x] 3.2 Додати schema/logical type mapping для timestamp/decimal/null semantics
- [x] 3.3 Додати parquet roundtrip та selective-column integration тести

## 4. Core DataFrame and LazyFrame MVP Surface

- [x] 4.1 Додати MVP-методи DataFrame (`slice`, `unique`, `concat`, `fill_null`, `drop_nulls`)
- [x] 4.2 Вирівняти LazyFrame surface для MVP-операцій і їх планування в optimizer
- [x] 4.3 Розширити join parity до `right/full` і зафіксувати null-aware sorting/grouping семантику

## 5. Quality and Performance Gates

- [x] 5.1 Посилити CI/release gates: `gofmt -l`, `go vet`, `go test`, `-race`, parity thresholds
- [x] 5.2 Додати benchmark regression budget (`ns/op`, `allocs/op`, `B/op`) і baseline policy
- [x] 5.3 Додати повторювані stability прогонки для детекту flaky тестів

## 6. Documentation and Developer Examples

- [x] 6.1 Оновити developer docs для DataFrame/Series/LazyFrame та IO/semantics
- [x] 6.2 Додати runnable examples для parquet scan, pushdown, joins, analytics і streaming collect
- [x] 6.3 Підготувати migration notes для потенційних **BREAKING** API alignment змін v0.3
