## MODIFIED Requirements

### Requirement: Project SHALL maintain automated parity conformance suite
Система SHALL містити автоматизований набір parity-тестів, що порівнює результати gopolars і Python Polars на однакових fixture datasets, включно з nightly differential профілем для SQL/window/nested/cloud сценаріїв.

#### Scenario: Differential result validation
- **WHEN** CI запускає differential conformance job
- **THEN** система виявляє будь-які semantic deviations і маркує pipeline як failed

#### Scenario: Nightly differential expansion
- **WHEN** запускається nightly conformance прогін
- **THEN** система покриває розширені v0.4 capability fixtures і публікує детальний звіт відхилень

### Requirement: Project SHALL define objective parity coverage metrics
Система SHALL публікувати coverage matrix за API namespaces, expressions, IO features і optimizer rules з числовим відсотком покриття, доповненим capability-level evidence для release.

#### Scenario: Coverage gate on release candidate
- **WHEN** формується release candidate v0.2
- **THEN** pipeline дозволяє реліз лише якщо parity coverage досягає затвердженого порогу

#### Scenario: Capability evidence for v0.4 release
- **WHEN** готується v0.4 release candidate
- **THEN** система надає coverage/evidence matrix по SQL/window, nested, cloud IO і observability capabilities
