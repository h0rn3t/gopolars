## MODIFIED Requirements

### Requirement: Optimizer SHALL implement parity baseline for logical rewrites
Система SHALL підтримувати rule set для projection/predicate pushdown, constant folding, expression simplification, join reordering і plan normalization згідно parity profile, а також adaptive planning рішень для v0.5 pipeline patterns з advanced join/reshape/sql навантаженнями.

#### Scenario: Optimizer rewrite parity
- **WHEN** lazy query має оптимізовні патерни
- **THEN** система застосовує еквівалентні rewrites без зміни результатної семантики відносно Polars Python

#### Scenario: Adaptive planning on mixed workloads
- **WHEN** pipeline поєднує window, nested, reshape, advanced joins і cloud scan оператори
- **THEN** optimizer обирає сумісний adaptive план із прогнозованою продуктивністю та коректною семантикою

### Requirement: Execution engine SHALL support streaming-compatible pipelines
Система SHALL виконувати streaming-friendly pipelines для великих наборів даних з bounded-memory strategy і deterministic outputs, включно з fallback-стратегією для операторів, що не можуть бути виконані в streaming path.

#### Scenario: Streaming pipeline parity
- **WHEN** користувач запускає lazy pipeline у streaming mode
- **THEN** система завершує виконання в межах memory budget і повертає результати, сумісні з non-streaming semantics

#### Scenario: Deterministic fallback behavior
- **WHEN** pipeline містить оператор, несумісний із поточним streaming path
- **THEN** система виконує контрольований fallback без semantic drift і з видимою діагностикою причини

### Requirement: Explainability SHALL expose plan stages and optimizer decisions
Система SHALL надавати explain output, що відображає logical/optimized/physical stages і ключові optimizer decisions у стабільному форматі.

#### Scenario: Explain contract stability
- **WHEN** користувач викликає Explain на еквівалентному запиті
- **THEN** система повертає детермінований і валідуємий план з необхідними stage annotations
