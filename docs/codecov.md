# Налаштування Codecov

Покриття збирається в job `coverage` workflow [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) і завантажується в [Codecov](https://app.codecov.io/gh/h0rn3t/gopolars).

## Передумови

1. Обліковий запис на [codecov.io](https://about.codecov.io/).
2. Встановлений [Codecov GitHub App](https://github.com/apps/codecov) для організації `h0rn3t` з доступом до репозиторію `gopolars`.
3. Репозиторій **приватний** — без upload token завантаження не працює.

## Крок 1. Підключити репозиторій

1. Увійдіть на [app.codecov.io](https://app.codecov.io/).
2. **Add repository** → оберіть `h0rn3t/gopolars`.
3. На сторінці налаштування репозиторію скопіюйте **Repository upload token** (розділ *General*).

## Крок 2. Додати секрет у GitHub

```bash
gh secret set CODECOV_TOKEN -R h0rn3t/gopolars
# вставте токен з Codecov, коли CLI запитає значення
```

Або вручну: **GitHub** → `h0rn3t/gopolars` → **Settings** → **Secrets and variables** → **Actions** → **New repository secret**:

| Name            | Value                    |
|-----------------|--------------------------|
| `CODECOV_TOKEN` | upload token з Codecov   |

## Крок 3. Перевірити default branch

У Codecov: **Settings** → **General** → **Default branch** = `main` (як у GitHub).

## Крок 4. Запустити CI

Після push або PR workflow `coverage` згенерує `coverage.out` для `./pkg/...` і завантажить звіт.

- Бадж у README: `https://codecov.io/gh/h0rn3t/gopolars/graph/badge.svg`
- Дашборд: `https://app.codecov.io/gh/h0rn3t/gopolars`

## Що вимірюється

- **Пакети:** лише `./pkg/...` (публічна бібліотека).
- **Прапорець:** `unit`.
- **Ігнорується:** `bench/`, `examples/`, `test/`, `*_test.go` (див. [`codecov.yml`](../codecov.yml)).

Conformance-тести в `test/` не входять у метрику Codecov; вони залишаються в job `ci` (`go test -race ./...`).

## Статуси PR

У [`codecov.yml`](../codecov.yml) перевірки `project` і `patch` увімкнені як **informational** — PR не блокується через покриття, поки команда не підніме пороги. Щоб увімкнути блокування, змініть `informational: true` на `false`.

## Усунення проблем

| Симптом | Дія |
|---------|-----|
| Бадж «unknown» | Переконайтесь, що `coverage` job пройшов і токен додано |
| `Token required` у логах CI | Додайте `CODECOV_TOKEN` (приватний репо) |
| Немає коментаря в PR | Встановіть Codecov GitHub App; `checkout` з `fetch-depth: 2` |
| PR з fork без покриття | Секрети недоступні форкам — очікувана поведінка GitHub |
