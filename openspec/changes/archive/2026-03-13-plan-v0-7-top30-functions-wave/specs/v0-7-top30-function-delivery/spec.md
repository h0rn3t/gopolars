## ADDED Requirements

### Requirement: System SHALL deliver all functions listed in v0.7 Top-30 backlog
Система SHALL реализовать весь набор функций из `docs/parity/v0_7_top30_functions.md` без пропусков.

#### Scenario: Validate Top-30 completion
- **WHEN** release pipeline проверяет список функций v0.7
- **THEN** каждая функция из Top-30 имеет реализованный API contract и test evidence

### Requirement: Top-30 implementation SHALL preserve cross-surface semantic consistency
Система MUST обеспечивать согласованное поведение для функций, которые доступны в eager/lazy/sql формах.

#### Scenario: Verify cross-surface equivalence
- **WHEN** эквивалентный pipeline запускается через разные surfaces
- **THEN** результаты и error semantics остаются эквивалентными в утвержденном scope

### Requirement: Top-30 delivery SHALL be measurable with explicit coverage tracking
Система SHALL публиковать измеримый отчёт о покрытии Top-30 capability.

#### Scenario: Publish Top-30 coverage report
- **WHEN** запускается CI quality stage для v0.7
- **THEN** формируется отчёт с числом реализованных функций и статусом conformance/performance checks
