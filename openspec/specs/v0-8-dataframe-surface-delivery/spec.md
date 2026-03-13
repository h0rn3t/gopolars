# v0-8-dataframe-surface-delivery Specification

## Purpose
TBD - created by archiving change plan-v0-8-dataframe-surface-wave. Update Purpose after archive.
## Requirements
### Requirement: System SHALL deliver all methods in v0.8 DataFrame wave
Система SHALL реализовать весь утвержденный список из 25 DataFrame методов v0.8 wave.

#### Scenario: Validate DataFrame wave completion
- **WHEN** release pipeline проверяет wave список
- **THEN** каждый метод из списка реализован и покрыт тестами

### Requirement: DataFrame wave SHALL preserve API consistency contracts
Система MUST сохранять immutability, predictable return contracts и naming consistency для новых методов.

#### Scenario: Verify API consistency
- **WHEN** пользователь комбинирует новые методы в цепочке DataFrame операций
- **THEN** поведение соответствует установленным DataFrame API контрактам

### Requirement: Wave completion SHALL be reflected in parity evidence
Система SHALL обновлять parity matrix и release evidence после закрытия wave.

#### Scenario: Publish v0.8 parity evidence
- **WHEN** wave реализован и тесты пройдены
- **THEN** matrix и сопутствующие отчеты показывают актуальный статус методов

