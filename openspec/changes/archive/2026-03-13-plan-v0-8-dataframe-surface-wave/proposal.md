## Why

После v0.7 в DataFrame surface остаётся набор нереализованных методов, который напрямую влияет на практический DX parity с Python Polars.  
Запрошен целевой wave для 25 методов DataFrame из parity отчёта.

## What Changes

- Реализовать DataFrame методы:
  - `approx_n_unique`
  - `bottom_k`
  - `cast`
  - `clear`
  - `clone`
  - `collect_schema`
  - `corr`
  - `count`
  - `describe`
  - `deserialize`
  - `drop_in_place`
  - `dtypes`
  - `equals`
  - `extend`
  - `flags`
  - `fold`
  - `gather_every`
  - `get_column`
  - `get_column_index`
  - `get_columns`
  - `glimpse`
  - `hash_rows`
  - `hstack`
  - `insert_column`
  - `interpolate`
- Добавить conformance и regression checks по каждому методу.
- Зафиксировать parity evidence в matrix/report артефактах.

## Capabilities

### New Capabilities
- `v0-8-dataframe-surface-delivery`: delivery capability для закрытия DataFrame method wave.

### Modified Capabilities
- `api-parity-surface`: расширение DataFrame API на 25 методов.
- `conformance-quality-suite`: отдельный conformance профиль для v0.8 DataFrame wave.

## Impact

- Изменения затронут `pkg/polars`, `pkg/frame`, `pkg/series`, `test/unit`, `test/conformance`, parity docs.
- Увеличится набор API и тестовых фикстур для DataFrame compatibility.
