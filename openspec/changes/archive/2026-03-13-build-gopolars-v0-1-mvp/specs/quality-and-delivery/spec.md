## ADDED Requirements

### Requirement: Project SHALL provide comprehensive automated tests
Система SHALL мати unit та integration тести для ключових операцій DataFrame/LazyFrame/IO/SQL, включно з edge cases для null, типів даних і join/agg поведінки.

#### Scenario: CI test execution
- **WHEN** запускається CI pipeline для pull request
- **THEN** всі обовʼязкові тести проходять перед мерджем змін

### Requirement: Project SHALL include performance benchmarks
Система SHALL містити benchmark-набір для фільтрації, агрегацій, join та end-to-end сценаріїв і SHALL збирати базові метрики продуктивності.

#### Scenario: Benchmark regression visibility
- **WHEN** запускається benchmark workflow
- **THEN** команда отримує повторювані метрики часу виконання та алокацій для порівняння регресій

### Requirement: CI/CD SHALL enforce code quality and release readiness
Система SHALL перевіряти `gofmt`, `go vet`, `go test`, `go test -race` та SHALL містити workflow підготовки версійного релізу модуля.

#### Scenario: Release gate checks
- **WHEN** готується релізна версія
- **THEN** pipeline блокує публікацію, якщо quality checks не пройдені
