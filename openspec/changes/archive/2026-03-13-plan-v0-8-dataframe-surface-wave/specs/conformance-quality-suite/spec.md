## MODIFIED Requirements

### Requirement: Project SHALL maintain automated parity conformance suite
Система SHALL включать отдельный conformance профиль для v0.8 DataFrame method wave.

#### Scenario: Run DataFrame v0.8 conformance profile
- **WHEN** CI запускает профиль v0.8
- **THEN** любые семантические отклонения по 25 методам фиксируются как failure

### Requirement: Project SHALL define objective parity coverage metrics
Система SHALL публиковать coverage отчет по списку 25 методов v0.8 wave.

#### Scenario: Gate v0.8 release by DataFrame wave completion
- **WHEN** формируется release candidate v0.8
- **THEN** pipeline требует полного покрытия списка и зеленых quality gates
