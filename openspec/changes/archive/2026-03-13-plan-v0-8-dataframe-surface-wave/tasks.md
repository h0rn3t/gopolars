## 1. Schema, Metadata and Inspection Methods

- [x] 1.1 Реализовать `collect_schema`, `dtypes`, `flags`
- [x] 1.2 Реализовать `get_column`, `get_columns`, `get_column_index`, `glimpse`
- [x] 1.3 Добавить unit/conformance тесты для inspection методов

## 2. Mutation-like Utility Methods

- [x] 2.1 Реализовать `clear`, `clone`, `drop_in_place`
- [x] 2.2 Реализовать `extend`, `hstack`, `insert_column`
- [x] 2.3 Добавить тесты на immutability и корректность мутационных сценариев

## 3. Statistics and Aggregation Helpers

- [x] 3.1 Реализовать `approx_n_unique`, `corr`, `count`, `describe`
- [x] 3.2 Реализовать `hash_rows`
- [x] 3.3 Добавить parity тесты для статистических методов

## 4. Transform and Selection Helpers

- [x] 4.1 Реализовать `bottom_k`, `cast`, `equals`, `fold`
- [x] 4.2 Реализовать `gather_every`, `interpolate`
- [x] 4.3 Добавить тесты edge-cases для transform методов

## 5. Serialization and Evidence

- [x] 5.1 Реализовать `deserialize`
- [x] 5.2 Добавить DataFrame v0.8 conformance профиль и coverage gate
- [x] 5.3 Обновить parity matrix и release evidence документы
