## v0.5 Migration Notes

### Alignment highlights

- Добавлены advanced join режимы: `semi`, `anti`, `cross`, `asof`.
- Добавлены reshape операции: `melt` и `pivot` в eager/lazy путях.
- SQL слой расширен поддержкой subquery в `FROM` и set-операций `UNION`/`INTERSECT`/`EXCEPT`.
- Lazy execution поддерживает sink materialization: parquet/csv/ipc.

### Migration guidance

- Если код использовал ручные workaround для semi/anti фильтрации после join, замените на нативные join modes.
- Проверьте SQL пайплайны с объединением выборок и обновите тесты под set-operation семантику.
- Для batch workflows сравните `collect` и `sink+readback` результат на критичных датасетах.
- Для time-aware join сценариев явно указывайте `AsofDirection` и `AsofTolerance`.
