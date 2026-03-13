## 1. Wave A — High Priority Closure

- [x] 1.1 Реализовать `SQLContext.register_many`
- [x] 1.2 Закрыть high-priority Series методы (`drop_nans`, `fill_nan`, `null_count`, `rolling_*`, `to_list`)
- [x] 1.3 Добавить unit/conformance профиль для high-priority closure

## 2. Wave B — Medium Priority Closure

- [x] 2.1 Закрыть medium backlog для DataFrame
- [x] 2.2 Закрыть medium backlog для Expr
- [x] 2.3 Закрыть medium backlog для LazyFrame, Series и SQLContext (`execute_global`, `register_globals`)

## 3. Wave C — Low Priority Structural APIs

- [x] 3.1 Закрыть low backlog для DataFrame (structural/metadata/io helpers)
- [x] 3.2 Закрыть low backlog для LazyFrame (plan/materialization/helpers)
- [x] 3.3 Добавить regression тесты для structural low-priority wave

## 4. Wave D — Low Priority Compute APIs

- [ ] 4.1 Закрыть low backlog для Expr
- [ ] 4.2 Закрыть low backlog для Series
- [ ] 4.3 Добавить семантические conformance тесты для compute wave

## 5. Wave E — Final Stabilization and Evidence

- [ ] 5.1 Ввести финальный gate `remaining_methods == 0`
- [ ] 5.2 Обновить parity matrix, release evidence и migration docs
- [ ] 5.3 Прогнать полную валидацию (tests, vet, race, closure checks)
