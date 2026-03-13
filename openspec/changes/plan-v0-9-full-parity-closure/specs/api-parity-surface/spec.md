## MODIFIED Requirements

### Requirement: DataFrame, LazyFrame, Expr, Series, SQLContext SHALL achieve complete parity closure
Система SHALL закрыть весь remaining методный backlog по объектам `DataFrame`, `LazyFrame`, `Expr`, `Series`, `SQLContext` из parity матрицы.

#### Scenario: Remaining methods become zero
- **WHEN** выполняется финальный parity gate
- **THEN** количество нереализованных методов в актуальном отчёте равно 0

### Requirement: Public API growth SHALL preserve compatibility contracts
Система MUST сохранять обратную совместимость и устойчивые контракты поведения при массовом расширении surface.

#### Scenario: Existing workloads after parity expansion
- **WHEN** пользователи обновляются на релиз с полным parity closure
- **THEN** ранее рабочие сценарии продолжают выполняться без регрессий
