# series-public-api Specification

## Purpose
TBD - synced from change plan-v0-3-core-parity-and-performance. Update Purpose if needed.

## Requirements
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

### Requirement: Series SHALL close or explicitly classify each remaining full-matrix low-priority row
Система SHALL для кожного методу `Series`, що залишається у статусі `не реализовано` / `low` у `docs/parity/python_polars_full_matrix.md`, або (a) надати публічний Go-еквівалент з документованою семантикою Polars у межах підтримуваних dtype, або (b) оновити матрицю на **deferred** / **non-goal** з текстовим обґрунтуванням у відповідній OpenSpec-зміні або `design.md`.

#### Scenario: Matrix row reflects implementation
- **WHEN** метод реалізовано на `Series` або її namespace
- **THEN** відповідний рядок матриці SHALL показувати `реализовано`, еквівалентний Go символ і пріоритет `—`

#### Scenario: Matrix row reflects explicit deferral
- **WHEN** метод свідомо не реалізовується в межах зміни
- **THEN** матриця або `design.md` SHALL містити категорію **deferred** або **non-goal** і посилання на обмеження зберігання / ecosystem

#### Scenario: No silent gap
- **WHEN** закривається зміна, що покриває low-priority хвіст
- **THEN** Review у `tasks.md` або матриця SHALL пояснювати залишковий зліч `не реализовано` / `low` відносно цільового порогу

### Requirement: Series rolling-by and ewm-by family SHALL be deterministic or explicitly unsupported
Для методів сімейства `rolling_*_by`, `ewm_mean_by`, `rolling_rank*` система SHALL забезпечити детерміновану поведінку на тестових dtype або стабільну помилку `not supported` до першої невалідної комбінації dtype/вікна.

#### Scenario: Unsupported dtype for rolling_by
- **WHEN** користувач викликає rolling-by на dtype без дорожньої карти реалізації
- **THEN** система SHALL повернути діагностичну помилку без паніки
