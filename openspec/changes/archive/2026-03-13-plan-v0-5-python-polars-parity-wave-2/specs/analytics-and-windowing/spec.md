## MODIFIED Requirements

### Requirement: System SHALL support full window function parity
Система SHALL підтримувати window expressions з `over`, ranking, cumulative та partition/order semantics, сумісні з Polars Python, включно з розширеним v0.5 SQL/API покриттям.

#### Scenario: Partitioned window aggregation parity
- **WHEN** користувач виконує window aggregation з partition і order
- **THEN** система повертає ті самі значення та row alignment, що і Polars Python

### Requirement: System SHALL support rolling and dynamic time windows
Система SHALL підтримувати rolling і dynamic group windows для temporal analytics з еквівалентними boundary та closed-interval rules, а також time-aware join сумісність у комбінованих pipeline.

#### Scenario: Dynamic temporal grouping parity
- **WHEN** виконується group-by-dynamic pipeline над часовим рядом
- **THEN** система формує ті самі вікна й агрегати, що і Polars Python

#### Scenario: Rolling analytics with asof join context
- **WHEN** rolling/dynamic обчислення застосовуються після time-aware join
- **THEN** система зберігає коректні часові межі та детерміновані агрегати

### Requirement: System SHALL support advanced aggregation and reshaping analytics
Система SHALL підтримувати pivot, melt, percentile/quantile, correlation та інші аналітичні операції, необхідні для v0.5 parity matrix.

#### Scenario: Reshape and aggregate parity
- **WHEN** користувач виконує pivot/melt з подальшими агрегаціями
- **THEN** результуючі схеми, значення і ordering правила відповідають Polars Python
