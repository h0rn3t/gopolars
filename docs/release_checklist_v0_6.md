## v0.6 Release Checklist

- Namespace parity fixtures покрывают расширенные string/datetime/list сценарии.
- Temporal windows (`group_by_dynamic`, `rolling`) проверены в eager/lazy parity тестах.
- Explain diagnostics и execution report соответствуют schema v2 и CI контрактам.
- Performance budgets из `docs/performance/v0_6_budgets.json` валидированы в CI.
- Regression report для micro/macro benchmark сформирован и приложен к release evidence.
- Coverage matrix `docs/parity/v0_6_coverage.json` удовлетворяет порогам.
- Migration notes и compatibility evidence опубликованы.
