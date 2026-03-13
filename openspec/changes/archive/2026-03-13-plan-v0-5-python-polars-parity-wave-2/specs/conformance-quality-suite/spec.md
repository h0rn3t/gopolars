## MODIFIED Requirements

### Requirement: Project SHALL maintain automated parity conformance suite
Система SHALL містити автоматизований набір parity-тестів, що порівнює результати gopolars і Python Polars на однакових fixture datasets, включно з nightly differential профілем для SQL/window/nested/cloud, advanced join і reshape сценаріїв.

#### Scenario: Differential result validation
- **WHEN** CI запускає differential conformance job
- **THEN** система виявляє будь-які semantic deviations і маркує pipeline як failed

#### Scenario: Nightly differential expansion
- **WHEN** запускається nightly conformance прогін
- **THEN** система покриває розширені v0.5 capability fixtures і публікує детальний звіт відхилень

### Requirement: Project SHALL define objective parity coverage metrics
Система SHALL публікувати coverage matrix за API namespaces, expressions, IO features і optimizer rules з числовим відсотком покриття, доповненим capability-level evidence для release.

#### Scenario: Coverage gate on release candidate
- **WHEN** формується release candidate v0.5
- **THEN** pipeline дозволяє реліз лише якщо parity coverage досягає затвердженого порогу

#### Scenario: Capability evidence for v0.5 release
- **WHEN** готується v0.5 release candidate
- **THEN** система надає coverage/evidence matrix по SQL, advanced joins, reshape, sink/materialization і namespace parity

### Requirement: Project SHALL enforce performance and stability gates
Система SHALL запускати micro/macro benchmarks, memory regression checks і race/stress suites як обов’язкові quality gates для v0.5.

#### Scenario: Benchmark and stability enforcement
- **WHEN** зміна впливає на execution core, SQL planner або IO
- **THEN** CI перевіряє performance budget і стабільність перед merge
