# v0.2 API Alignment Migration Notes

## Scope

Цей документ фіксує потенційні зміни API для узгодження з Python Polars у v0.2 та правила міграції.

## Potential Breaking Areas

1. Sort semantics
- Null і NaN ordering може бути уточнений для повного parity.
- Міграція: явно задавати сортування через параметри `Descending` і перевіряти expected ordering у тестах.

2. Explain format contract
- Explain переходить до стабільного staged-формату (`logical/optimized/physical`).
- Міграція: не покладатися на попередній plain-text формат; оновити golden fixtures.

3. SQL behavior alignment
- Семантика alias/having/order може бути уточнена під Polars-like поведінку.
- Міграція: для критичних запитів використовувати parity тести SQL vs API.

4. Future dtype expansion
- Додавання nested/categorical/decimal типів може вплинути на type inference і касти.
- Міграція: фіксувати схеми явно в IO та перевіряти касти у conformance-тестах.

## Compatibility Strategy

- Зберігати backward-compatible paths коли можливо.
- Для несумісних змін публікувати migration notes у release.
- Всі breaking alignment зміни мають супроводжуватися conformance і regression тестами.
