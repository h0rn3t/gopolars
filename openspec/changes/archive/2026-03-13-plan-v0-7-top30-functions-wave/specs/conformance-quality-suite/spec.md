## MODIFIED Requirements

### Requirement: Project SHALL maintain automated parity conformance suite
Система SHALL иметь выделенный differential профиль для Top-30 v0.7 функций.

#### Scenario: Run Top-30 differential suite
- **WHEN** CI запускает Top-30 conformance прогон
- **THEN** любое отклонение semantics фиксируется как failure с функцией-источником

### Requirement: Project SHALL define objective parity coverage metrics
Система SHALL публиковать coverage отчёт по Top-30 с целевым порогом `30/30` для release readiness.

#### Scenario: Gate v0.7 release by Top-30 completion
- **WHEN** формируется release candidate v0.7
- **THEN** pipeline разрешает релиз только при полном покрытии Top-30 и зелёных quality gates
