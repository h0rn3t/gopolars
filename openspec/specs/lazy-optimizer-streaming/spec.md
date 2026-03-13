# lazy-optimizer-streaming Specification

## Purpose
TBD - created by archiving change plan-v0-2-python-polars-parity. Update Purpose after archive.
## Requirements
### Requirement: Optimizer SHALL implement parity baseline for logical rewrites
Система SHALL підтримувати rule set для projection/predicate pushdown, constant folding, expression simplification, join reordering і plan normalization згідно parity profile, а також adaptive planning рішень для v0.6 window-heavy workloads.

#### Scenario: Optimizer rewrite parity
- **WHEN** lazy query має оптимізовні патерни
- **THEN** система застосовує еквівалентні rewrites без зміни результатної семантики відносно Polars Python

#### Scenario: Adaptive planning on temporal-heavy pipelines
- **WHEN** pipeline поєднує dynamic/rolling windows, joins і stateful операції
- **THEN** optimizer обирає сумісний план із передбачуваною продуктивністю та коректною семантикою

### Requirement: Execution engine SHALL support streaming-compatible pipelines
Система SHALL виконувати streaming-friendly pipelines для великих наборів даних з bounded-memory strategy і deterministic outputs, включно з fallback-поведінкою для new stateful window operators.

#### Scenario: Streaming pipeline parity
- **WHEN** користувач запускає lazy pipeline у streaming mode
- **THEN** система завершує виконання в межах memory budget і повертає результати, сумісні з non-streaming semantics

### Requirement: Explainability SHALL expose plan stages and optimizer decisions
Система SHALL надавати explain output, що відображає logical/optimized/physical stages і ключові optimizer decisions у стабільному форматі, включно з performance/telemetry markers.

#### Scenario: Explain contract stability
- **WHEN** користувач викликає Explain на еквівалентному запиті
- **THEN** система повертає детермінований і валідуємий план з необхідними stage annotations

