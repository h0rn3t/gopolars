## v0.6 Migration Notes

### Highlights

- Расширены namespace операции для string/datetime/list (trim, starts_with, list_get, dt_hour, dt_weekday).
- Добавлены temporal window API: `GroupByDynamic` и расширенный `RollingMean` в eager/lazy путях.
- Explain diagnostics и execution report обновлены до schema v2 с temporal/performance markers.
- Добавлены performance budget и regression evidence артефакты для release gates.

### Migration guidance

- Проверьте pipeline, использующие ручные временные бакеты, и перенесите их на `GroupByDynamic`.
- Для rolling аналитики зафиксируйте `Closed`, `Window`, `MinRows` в тестах для детерминированности.
- Обновите парсинг diagnostics/report output на schema v2 (`temporal_window_operations`, `performance_markers`).
- Подключите performance budget и regression report scripts в nightly/release workflows.
