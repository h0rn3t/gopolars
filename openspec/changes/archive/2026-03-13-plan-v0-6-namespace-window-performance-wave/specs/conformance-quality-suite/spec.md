## MODIFIED Requirements

### Requirement: Project SHALL maintain automated parity conformance suite
Система SHALL містити автоматизований набір parity-тестів, що порівнює результати gopolars і Python Polars на однакових fixture datasets, включно з nightly differential профілем для namespace + temporal-window + performance-sensitive сценаріїв.

#### Scenario: Differential result validation
- **WHEN** CI запускає differential conformance job
- **THEN** система виявляє будь-які semantic deviations і маркує pipeline як failed

### Requirement: Project SHALL define objective parity coverage metrics
Система SHALL публікувати coverage matrix за API namespaces, expressions, temporal analytics і optimizer rules з числовим відсотком покриття та release evidence для v0.6.

#### Scenario: Coverage gate on release candidate
- **WHEN** формується release candidate v0.6
- **THEN** pipeline дозволяє реліз лише якщо parity coverage досягає затвердженого порогу

### Requirement: Project SHALL enforce performance and stability gates
Система SHALL запускати micro/macro benchmarks, memory regression checks і race/stress suites як обов’язкові quality gates для v0.6, включно з окремими budget thresholds для temporal workloads.

#### Scenario: Benchmark and stability enforcement
- **WHEN** зміна впливає на execution core, optimizer або temporal analytics path
- **THEN** CI перевіряє performance budget і стабільність перед merge
