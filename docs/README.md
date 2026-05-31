# Документація API / API documentation

Інтерактивна довідка публічного API `pkg/polars` для GitHub Pages.

| | |
| --- | --- |
| **Сайт** | [https://h0rn3t.github.io/gopolars/](https://h0rn3t.github.io/gopolars/) |
| **Мови** | Українська (за замовчуванням), English — перемикач **УК / EN**, або [`en.html`](en.html) / [`?lang=en`](index.html?lang=en) |
| **Підсвітка** | [Prism.js](https://prismjs.com/) — `language-go`, `language-bash` у прикладах |

## Локальний перегляд

```bash
cd docs
python3 -m http.server 8765
# http://localhost:8765/
```

Або відкрийте `index.html` напряму (CDN Prism потребує мережі).

## Структура

- `index.html` — розмітка секцій API
- `assets/i18n.js` + `assets/app.js` — вихідники перекладів і логіки
- `assets/docs.js` — збірка (`cat i18n.js app.js > docs.js`), підключається в `index.html`
- `assets/style.css` — тема та стилі токенів Prism

## Оновлення перекладів

1. Додайте ключ у `assets/i18n.js` для `uk` і `en`.
2. Перезберіть: `cat assets/i18n.js assets/app.js > assets/docs.js`
3. Повісьте `data-i18n="ключ"` на елемент у `index.html`.
4. Для атрибутів: `data-i18n-attr="placeholder:search.placeholder"`.

Перемикання мов — посилання `?lang=en` / `?lang=uk` (повне перезавантаження сторінки, працює без JS).

Приклади коду залишаються спільними; мова UI не змінює Go-фрагменти.

## Деплой

Push у `main` зі змінами в `docs/**` запускає [`.github/workflows/pages.yml`](../.github/workflows/pages.yml).
