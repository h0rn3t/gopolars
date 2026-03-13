## ADDED Requirements

### Requirement: Public Series Type in Polars API
Система SHALL предоставлять публичный `Series` API в пакете `polars` с контрактом, пригодным для column-first операций и совместимым с текущей DataFrame моделью.

#### Scenario: Construct series from typed values
- **WHEN** разработчик создаёт `Series` с именем, dtype и набором значений
- **THEN** система возвращает валидный series-объект с корректной длиной и типом

#### Scenario: Reject invalid dtype/value combinations
- **WHEN** разработчик передаёт значения, не соответствующие объявленному dtype
- **THEN** система MUST вернуть диагностируемую ошибку валидации

### Requirement: Null-Aware Series Operations
Система MUST поддерживать null-aware базовые операции серии: `is_null`, `is_not_null`, `fill_null`, `drop_nulls` с предсказуемым поведением.

#### Scenario: Fill null values with scalar
- **WHEN** в серии присутствуют null и вызывается `fill_null` со скалярным значением
- **THEN** все null-элементы заменяются значением, а остальные элементы сохраняются

#### Scenario: Drop null values from series
- **WHEN** вызывается `drop_nulls` для серии с null-элементами
- **THEN** результат содержит только ненулевые элементы и корректно пересчитанную длину

### Requirement: Series Arithmetic and Comparison Surface
Система SHALL поддерживать MVP-набор векторных операций `add/sub/mul/div` и сравнений `eq/ne/gt/ge/lt/le` на уровне публичного Series API.

#### Scenario: Execute element-wise arithmetic on numeric series
- **WHEN** разработчик применяет арифметическую операцию к двум числовым сериям одинаковой длины
- **THEN** система возвращает серию результата с типом и значениями согласно операции

#### Scenario: Error on length mismatch
- **WHEN** операция выполняется между сериями разной длины
- **THEN** система MUST вернуть ошибку несоответствия размеров
