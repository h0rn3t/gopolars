## v0.5 Release Checklist

- Advanced joins (`semi`, `anti`, `cross`, `asof`) покрыты unit и conformance тестами.
- Reshape операции (`melt`, `pivot`) проходят eager/lazy parity проверки.
- SQL subqueries и set-операции проходят differential fixtures.
- Lazy sink materialization проверена для parquet/csv/ipc.
- Explain diagnostics schema включает set/reshape indicators и стабильна в CI.
- Coverage matrix для v0.5 достигла утверждённого порога.
- Migration notes и compatibility evidence опубликованы.
