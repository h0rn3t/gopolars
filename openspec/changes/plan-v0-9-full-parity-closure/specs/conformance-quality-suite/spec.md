## MODIFIED Requirements

### Requirement: Conformance suite SHALL include full closure gate
Система SHALL иметь gate, который валидирует, что remaining methods в parity matrix равны нулю.

#### Scenario: Run full parity closure gate
- **WHEN** CI запускает финальный gate
- **THEN** pipeline завершается ошибкой при `remaining_methods > 0`

### Requirement: Compare report SHALL provide deterministic evidence
Система SHALL публиковать детерминированный compare отчёт с benchmark метриками и сводными таблицами.

#### Scenario: Generate release-grade compare evidence
- **WHEN** запускается FULL_REPORT режим compare script
- **THEN** отчёт содержит единообразные метрики и пригоден для release evidence
