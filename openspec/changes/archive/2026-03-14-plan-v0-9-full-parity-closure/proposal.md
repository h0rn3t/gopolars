## Why

По отчёту `FULL_REPORT=1 ./scripts/compare_python_polars_functions.sh ...` остаётся **550 нереализованных методов**:

- DataFrame: 80
- LazyFrame: 64
- Expr: 190
- Series: 213
- SQLContext: 3

Приоритетный хвост:
- high: 9
- medium: 27
- low: 514

Нужна программная инициатива, которая закроет не только high/medium, но и весь remaining low backlog до полного функционального покрытия по матрице.

## What Changes

- Запустить программу `v0.9 full parity closure` для реализации всех 550 remaining методов.
- Разбить внедрение на волны:
  1) high-priority closure,
  2) medium-priority closure,
  3) low-priority DataFrame/LazyFrame,
  4) low-priority Expr/Series,
  5) hard edges и стабилизация.
- Добавить пообъектные conformance профили и единый финальный parity gate `0 remaining`.
- Расширить compare script evidence до release-grade уровня (summary + benchmark + deterministic snapshots).

## Capabilities

### New Capabilities
- `v0-9-full-parity-program` — программа полного закрытия parity матрицы.

### Modified Capabilities
- `api-parity-surface` — движение от partial parity к complete parity по всем публичным объектам.
- `conformance-quality-suite` — финальный quality gate с требованием `remaining_methods == 0`.

## Impact

- Масштабные изменения в `pkg/polars`, `pkg/frame`, `pkg/expr`, `pkg/series`, `pkg/lazy`, `pkg/sql`.
- Существенное расширение unit/conformance тестов и CI профилей.
- Обновление parity документации, релизных чеклистов и migration guidance.
