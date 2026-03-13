## Context

v0.7 закрыл high-priority Top-30, но DataFrame API всё ещё имеет заметные разрывы в utility/stat/inspection методах.  
Этот wave из 25 методов нужен для выравнивания повседневного DataFrame ergonomics с Python Polars.

## Goals / Non-Goals

**Goals**
- Реализовать все 25 перечисленных DataFrame методов.
- Сохранить immutability/chainability контракт текущего API.
- Обеспечить тестируемую семантическую совместимость с Python Polars.
- Обновить parity evidence по завершению wave.

**Non-Goals**
- Расширение LazyFrame/Expr/SQL scope вне зависимостей DataFrame методов.
- Полное закрытие всех remaining методов вне списка wave.

## Delivery Slices

1. **Schema/metadata/inspection**  
`collect_schema`, `dtypes`, `flags`, `get_column`, `get_columns`, `get_column_index`, `glimpse`.

2. **Mutation-like utility semantics**  
`clear`, `clone`, `drop_in_place`, `extend`, `hstack`, `insert_column`.

3. **Statistics and aggregation helpers**  
`approx_n_unique`, `corr`, `count`, `describe`, `hash_rows`.

4. **Transform/selection helpers**  
`bottom_k`, `cast`, `equals`, `fold`, `gather_every`, `interpolate`.

5. **Serialization and parity contracts**  
`deserialize` + обновление матрицы/отчётов/quality gates.

## Validation Strategy

- Unit tests на каждый метод в DataFrame wave.
- Differential/conformance fixtures для edge-case поведения.
- CI gates: `go test ./...`, `go vet ./...`, `go test -race ./...`.
- Обновление parity matrix после реализации wave.
