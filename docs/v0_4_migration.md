## v0.4 Migration Notes

### Potential breaking alignment areas

- SQL/window behavior now supports advanced constructs with stricter validation contracts.
- Nested list/struct transforms introduce explicit dtype/nullability contracts.
- Cloud dataset scans may include partition-derived columns in dataset mode.

### Migration guidance

- Проверьте SQL queries с window/CTE на соответствие обновлённым правилам.
- Для nested workflows обновите ожидания по explode/flatten null semantics.
- Для object-store dataset scans зафиксируйте partition naming `<key>=<value>`.
