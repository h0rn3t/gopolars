## Readiness Checklist

### Scope and Architecture
- [ ] MVP v0.3 scope заморожений і погоджений по capability-блоках
- [ ] Архітектурні рішення для Series/Lazy Scan/Parquet задокументовані та прийняті
- [ ] Arrow `23.0.1` і `parquet-go v0.29.0` зафіксовані у dependency policy

### Implementation Readiness
- [ ] Для кожної capability створений implementation backlog і owner
- [ ] Визначені phase exit criteria та контрольні точки roadmap
- [ ] Для потенційних **BREAKING** змін зафіксовані migration notes

### Quality and Performance Gates
- [ ] CI gate включає форматування, vet, unit, race і conformance перевірки
- [ ] Coverage/parity thresholds визначені і автоматично перевіряються
- [ ] Benchmark regression budget та baseline policy працюють у pipeline

### Testing and Documentation
- [ ] Unit/integration/differential/benchmark план покриває всі MVP capability
- [ ] Runnable examples проходять автоматичну перевірку
- [ ] Developer docs оновлені для DataFrame/Series/LazyFrame, IO і null/NaN semantics
