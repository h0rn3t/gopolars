## MODIFIED Requirements

### Requirement: DataFrame API SHALL match Python Polars core method surface
Система SHALL добавить в DataFrame API методы:
`approx_n_unique`, `bottom_k`, `cast`, `clear`, `clone`, `collect_schema`, `corr`, `count`, `describe`, `deserialize`, `drop_in_place`, `dtypes`, `equals`, `extend`, `flags`, `fold`, `gather_every`, `get_column`, `get_column_index`, `get_columns`, `glimpse`, `hash_rows`, `hstack`, `insert_column`, `interpolate`.

#### Scenario: DataFrame method parity for v0.8 wave
- **WHEN** пользователь вызывает любой метод из v0.8 списка
- **THEN** система возвращает валидный результат или диагностируемую ошибку согласно контракту
