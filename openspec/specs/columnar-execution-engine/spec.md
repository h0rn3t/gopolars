## Purpose
Визначає вимоги до columnar виконання, векторизації та паралелізму для MVP v0.1.

## Requirements

### Requirement: Engine SHALL use columnar in-memory representation
Система SHALL зберігати дані у columnar форматі з підтримкою типів `int64`, `float64`, `string`, `bool`, `datetime` та окремим null bitmap для nullable значень.

#### Scenario: Nullable values in columnar storage
- **WHEN** вхідний набір даних містить null у підтримуваних типах
- **THEN** система зберігає null-стан у bitmap без втрати значень не-null елементів

### Requirement: Engine SHALL execute vectorized operators
Система SHALL виконувати фільтрацію, арифметику та агрегації на рівні векторизованих chunk-операцій, а не поелементно через row-ітерацію.

#### Scenario: Vectorized filter execution
- **WHEN** застосовується `Filter` до DataFrame з великим обсягом рядків
- **THEN** система обробляє дані chunk-партіями через векторні kernels

### Requirement: Engine SHALL support parallel execution for core operators
Система SHALL розпаралелювати `Filter`, `GroupBy().Agg()` і `Join` через goroutines з детермінованим злиттям результатів.

#### Scenario: Parallel group-by aggregation
- **WHEN** користувач виконує `GroupBy().Agg()` на багаточанковому наборі
- **THEN** система обчислює часткові агрегації паралельно та повертає коректний фінальний результат
