## Context

Текущий статус после v0.8:
- реализовано: 130/680
- remaining: 550/680

На этом этапе нужны не точечные waves, а управляемая multi-wave программа с прозрачной метрикой прогресса и обязательной проверкой семантической совместимости.

## Goals / Non-Goals

**Goals**
- Закрыть все remaining методы из parity матрицы.
- Ввести объективный финальный gate: `remaining_methods == 0`.
- Сохранить стабильность API и производительности при расширении surface.
- Получить проверяемый release evidence пакет.

**Non-Goals**
- Переписывание внутреннего движка без связи с parity задачами.
- Изменение семантики уже реализованных методов без необходимости.

## Delivery Architecture

### Wave A — High Priority Closure
- `SQLContext.register_many`
- Series: `drop_nans`, `fill_nan`, `null_count`, `rolling_max`, `rolling_mean`, `rolling_min`, `rolling_sum`, `to_list`

### Wave B — Medium Priority Closure
- DataFrame medium backlog
- Expr medium backlog
- LazyFrame medium backlog
- Series medium backlog
- SQLContext: `execute_global`, `register_globals`

### Wave C — Low Priority Structural APIs (DataFrame/LazyFrame)
- DataFrame low backlog
- LazyFrame low backlog

### Wave D — Low Priority Compute APIs (Expr/Series)
- Expr low backlog
- Series low backlog

### Wave E — Hard Edge Stabilization
- Сложные методы с внешними зависимостями/графами/interop
- Повторная валидация parity + regression + perf budgets

## Validation Strategy

- Для каждой волны:
  - unit + conformance профиль
  - compare script snapshot update
  - regression + race checks
- Для финала:
  - `check_full_parity.sh` с `remaining == 0`
  - полная валидация `go test ./...`, `go vet ./...`, `go test -race ./...`
  - обновленные migration/release docs
