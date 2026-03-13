## v0.8 Migration Notes

### DataFrame surface expansion

В v0.8 добавлена волна из 25 DataFrame методов:

- `approx_n_unique`, `bottom_k`, `cast`, `clear`, `clone`, `collect_schema`
- `corr`, `count`, `describe`, `deserialize`, `drop_in_place`, `dtypes`
- `equals`, `extend`, `flags`, `fold`, `gather_every`
- `get_column`, `get_column_index`, `get_columns`, `glimpse`
- `hash_rows`, `hstack`, `insert_column`, `interpolate`

### Adoption guidance

- Для быстрой диагностики структуры используйте `collect_schema`, `dtypes`, `glimpse`.
- Для ingestion/reshape сценариев используйте `hstack`, `insert_column`, `extend`.
- Для статистического анализа используйте `corr`, `describe`, `approx_n_unique`, `hash_rows`.
- Для контроля v0.8 wave запускайте `scripts/check_v08_dataframe_wave.sh`.
