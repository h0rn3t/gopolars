## MODIFIED Requirements

### Requirement: Expression engine SHALL support full Polars expression repertoire
Система SHALL добавить Expr-функции из v0.7 Top-30: `over`, `rank`, `cum_sum`, `cum_count`, `replace`, `fill_null`, `fill_nan`, `rolling_min`, `rolling_max`, `rolling_mean`, `rolling_sum`, `rolling_std`.

#### Scenario: Expr Top-30 semantic parity
- **WHEN** conformance suite выполняет эталонные выражения из v0.7 Top-30
- **THEN** результаты и ошибки согласованы с Python Polars contracts

### Requirement: Null/NaN behavior SHALL match Polars semantics across operators
Система MUST обеспечивать корректный null/NaN contract для новых Expr операций `fill_null/fill_nan/rolling*`.

#### Scenario: Null/NaN behavior in Top-30 Expr
- **WHEN** операции выполняются над смешанными null/NaN наборами
- **THEN** система возвращает значения, соответствующие Polars semantics
