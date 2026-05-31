/** Переклади UI документації API (uk / en). Код у прикладах спільний для обох мов. */
window.GOPOLARS_I18N = {
  uk: {
    "meta.title": "gopolars — Довідка API",
    "meta.description":
      "Довідка з публічного API gopolars — високопродуктивна DataFrame-бібліотека для Go, натхненна Polars.",
    "logo.subtitle": "Довідка API",
    "search.placeholder": "Пошук методів…",
    "nav.start": "Початок",
    "nav.types": "Типи",
    "nav.integrations": "Інтеграції",
    "nav.other": "Інше",
    "nav.overview": "Огляд",
    "nav.install": "Встановлення",
    "nav.quickstart": "Швидкий старт",
    "nav.dataframe": "DataFrame",
    "nav.lazyframe": "LazyFrame",
    "nav.series": "Series",
    "nav.expr": "Expr",
    "nav.groupby": "GroupBy",
    "nav.io": "IO",
    "nav.sql": "SQL",
    "nav.joins": "Joins",
    "nav.dtypes": "Типи даних",
    "nav.parity": "Parity",
    "nav.simd": "SIMD",
    "hero.badge": "pkg/polars · v0.6+",
    "hero.title": "DataFrame API для Go",
    "hero.lead":
      "<strong>gopolars</strong> — колонкова бібліотека з eager/lazy виконанням, SQL-планувальником та IO для CSV, JSON, Parquet і IPC. Публічний API зосередений у пакеті <code>github.com/h0rn3t/gopolars/pkg/polars</code>.",
    "hero.cta.start": "Швидкий старт",
    "hero.cta.repo": "Репозиторій",
    "hero.stat.parity": "методів Polars parity",
    "hero.stat.coverage": "покриття тестами pkg/",
    "hero.stat.interfaces": "основних інтерфейсів API",
    "install.title": "Встановлення",
    "install.lead":
      "Потрібен Go 1.26+. Опційне SIMD-прискорення — через <code>GOEXPERIMENT=simd</code> на AMD64.",
    "install.callout":
      "<strong>Імпорт:</strong> у коді використовуйте аліас <code>polars</code> для зручності — як у прикладах Polars Python.",
    "quickstart.title": "Швидкий старт",
    "quickstart.lead": "Типовий lazy-пайплайн: scan → filter → group → collect.",
    "dataframe.title": "DataFrame",
    "dataframe.lead":
      "Eager-таблиця. Створення через <code>NewDataFrame</code>, Arrow або IO.",
    "dataframe.table.constructor": "Конструктор / фабрика",
    "dataframe.table.desc": "Опис",
    "dataframe.row.newdf": "З колонок <code>[]frame.SeriesInput</code>",
    "dataframe.row.arrow": "Імпорт з Arrow Table",
    "dataframe.row.io": "Завантаження з файлів",
    "dataframe.transforms": "Трансформації",
    "dataframe.aggregations": "Агрегації та опис",
    "dataframe.export": "Експорт",
    "lazyframe.title": "LazyFrame",
    "lazyframe.lead":
      "Відкладене виконання з pushdown на scan. Завершення — <code>Collect</code> або streaming.",
    "lazyframe.table.method": "Метод",
    "lazyframe.table.purpose": "Призначення",
    "lazyframe.row.collect": "Materialize у DataFrame",
    "lazyframe.row.streaming": "Streaming collect з обмеженою пам'яттю",
    "lazyframe.row.explain": "План виконання для діагностики",
    "lazyframe.row.scan": "Lazy scan + projection/predicate pushdown",
    "lazyframe.row.sink": "Запис без повного materialize в пам'яті",
    "series.title": "Series",
    "series.lead":
      "Колонка з null-aware арифметикою, rolling-операціями та namespaces.",
    "series.table.api": "API",
    "series.table.desc": "Опис",
    "series.row.new": "Створення серії з типом і значеннями",
    "series.row.ns": "String, datetime, list, struct namespaces",
    "series.row.rolling": "Віконні агрегації",
    "expr.title": "Expr",
    "expr.lead":
      "Вирази для <code>Select</code>, <code>Filter</code>, <code>WithColumns</code>, <code>Agg</code>.",
    "expr.table.fn": "Функція",
    "expr.table.example": "Приклад",
    "expr.row.col": "Колонка та літерал",
    "expr.row.agg": "Агрегації в GroupBy",
    "expr.row.when": "Умовні вирази",
    "expr.row.ns": "<code>list_len</code>, <code>str_lower</code>, <code>dt_year</code>, <code>explode</code>, …",
    "groupby.title": "GroupBy",
    "groupby.lead":
      "<code>df.GroupBy(keys...).Agg(exprs...)</code> — eager; lazy через <code>LazyFrame.GroupBy</code>.",
    "io.title": "IO",
    "io.lead": "Фасад <code>NewIO()</code> — читання, scan і SQL на джерелах.",
    "io.table.eager": "Eager read",
    "io.table.lazy": "Lazy scan",
    "io.row.parquet": "partition datasets",
    "io.callout":
      "<strong>Object store:</strong> URI <code>s3://</code>, <code>gcs://</code>, <code>az://</code> через env-профіль кореневих шляхів (див. тести <code>object_store</code> у репозиторії).",
    "sql.title": "SQL",
    "sql.lead":
      "<code>NewSQLContext()</code> — реєстрація таблиць і виконання SELECT з CTE, вікнами, set-ops.",
    "sql.row.register": "Локальні таблиці",
    "sql.row.execute": "Запит → LazyFrame",
    "sql.row.catalog": "Каталог",
    "joins.title": "Joins",
    "joins.lead": "Режими через <code>JoinInput.Type</code>:",
    "dtypes.title": "Типи даних",
    "dtypes.lead": "Аліаси в <code>polars</code> з пакета <code>pkg/dtypes</code>:",
    "parity.title": "Parity з Python Polars",
    "parity.lead": "Повна матриця методів — у репозиторії; на цій сторінці — зведення.",
    "parity.table.object": "Об'єкт",
    "parity.table.done": "Реалізовано",
    "parity.table.status": "Статус",
    "parity.tag.ready": "ready",
    "parity.tag.non_goals": "5 non-goals",
    "parity.details": "Деталі:",
    "simd.title": "SIMD (опційно)",
    "simd.lead":
      "AMD64 + Go 1.26+ + <code>GOEXPERIMENT=simd</code> для прискорених Sum/Min/Max на float64.",
    "simd.fallback":
      "Без прапорця — автоматичний scalar fallback з тими самими результатами.",
    "copy": "Копіювати",
    "copy.done": "Скопійовано",
    "copy.error": "Помилка",
    "lang.label": "Мова",
  },
  en: {
    "meta.title": "gopolars — API Reference",
    "meta.description":
      "Public API reference for gopolars — a high-performance Go DataFrame library inspired by Polars.",
    "logo.subtitle": "API Reference",
    "search.placeholder": "Search methods…",
    "nav.start": "Getting started",
    "nav.types": "Types",
    "nav.integrations": "Integrations",
    "nav.other": "More",
    "nav.overview": "Overview",
    "nav.install": "Installation",
    "nav.quickstart": "Quick start",
    "nav.dataframe": "DataFrame",
    "nav.lazyframe": "LazyFrame",
    "nav.series": "Series",
    "nav.expr": "Expr",
    "nav.groupby": "GroupBy",
    "nav.io": "IO",
    "nav.sql": "SQL",
    "nav.joins": "Joins",
    "nav.dtypes": "Data types",
    "nav.parity": "Parity",
    "nav.simd": "SIMD",
    "hero.badge": "pkg/polars · v0.6+",
    "hero.title": "DataFrame API for Go",
    "hero.lead":
      "<strong>gopolars</strong> is a columnar library with eager/lazy execution, a SQL planner, and IO for CSV, JSON, Parquet, and IPC. The public API lives in <code>github.com/h0rn3t/gopolars/pkg/polars</code>.",
    "hero.cta.start": "Quick start",
    "hero.cta.repo": "Repository",
    "hero.stat.parity": "Polars parity methods",
    "hero.stat.coverage": "pkg/ test coverage",
    "hero.stat.interfaces": "core API surfaces",
    "install.title": "Installation",
    "install.lead":
      "Requires Go 1.26+. Optional SIMD acceleration via <code>GOEXPERIMENT=simd</code> on AMD64.",
    "install.callout":
      "<strong>Import:</strong> use the alias <code>polars</code> in your code — same ergonomics as Python Polars examples.",
    "quickstart.title": "Quick start",
    "quickstart.lead": "Typical lazy pipeline: scan → filter → group → collect.",
    "dataframe.title": "DataFrame",
    "dataframe.lead":
      "Eager table. Create with <code>NewDataFrame</code>, Arrow, or IO.",
    "dataframe.table.constructor": "Constructor / factory",
    "dataframe.table.desc": "Description",
    "dataframe.row.newdf": "From <code>[]frame.SeriesInput</code> columns",
    "dataframe.row.arrow": "Import from an Arrow table",
    "dataframe.row.io": "Load from files",
    "dataframe.transforms": "Transforms",
    "dataframe.aggregations": "Aggregations & describe",
    "dataframe.export": "Export",
    "lazyframe.title": "LazyFrame",
    "lazyframe.lead":
      "Deferred execution with scan pushdown. Finish with <code>Collect</code> or streaming.",
    "lazyframe.table.method": "Method",
    "lazyframe.table.purpose": "Purpose",
    "lazyframe.row.collect": "Materialize to DataFrame",
    "lazyframe.row.streaming": "Streaming collect with bounded memory",
    "lazyframe.row.explain": "Execution plan for diagnostics",
    "lazyframe.row.scan": "Lazy scan + projection/predicate pushdown",
    "lazyframe.row.sink": "Write without full in-memory materialize",
    "series.title": "Series",
    "series.lead":
      "Column with null-aware arithmetic, rolling ops, and namespaces.",
    "series.table.api": "API",
    "series.table.desc": "Description",
    "series.row.new": "Create a series from type and values",
    "series.row.ns": "String, datetime, list, struct namespaces",
    "series.row.rolling": "Window aggregations",
    "expr.title": "Expr",
    "expr.lead":
      "Expressions for <code>Select</code>, <code>Filter</code>, <code>WithColumns</code>, <code>Agg</code>.",
    "expr.table.fn": "Function",
    "expr.table.example": "Example",
    "expr.row.col": "Column and literal",
    "expr.row.agg": "Aggregations in GroupBy",
    "expr.row.when": "Conditional expressions",
    "expr.row.ns": "<code>list_len</code>, <code>str_lower</code>, <code>dt_year</code>, <code>explode</code>, …",
    "groupby.title": "GroupBy",
    "groupby.lead":
      "<code>df.GroupBy(keys...).Agg(exprs...)</code> — eager; lazy via <code>LazyFrame.GroupBy</code>.",
    "io.title": "IO",
    "io.lead": "<code>NewIO()</code> facade — reads, scans, and SQL over sources.",
    "io.table.eager": "Eager read",
    "io.table.lazy": "Lazy scan",
    "io.row.parquet": "partition datasets",
    "io.callout":
      "<strong>Object store:</strong> <code>s3://</code>, <code>gcs://</code>, <code>az://</code> URIs via env-configured root paths (see <code>object_store</code> tests in the repo).",
    "sql.title": "SQL",
    "sql.lead":
      "<code>NewSQLContext()</code> — register tables and run SELECT with CTEs, windows, and set ops.",
    "sql.row.register": "Local tables",
    "sql.row.execute": "Query → LazyFrame",
    "sql.row.catalog": "Catalog",
    "joins.title": "Joins",
    "joins.lead": "Modes via <code>JoinInput.Type</code>:",
    "dtypes.title": "Data types",
    "dtypes.lead": "Aliases in <code>polars</code> from <code>pkg/dtypes</code>:",
    "parity.title": "Python Polars parity",
    "parity.lead": "Full method matrix lives in the repo; this page shows a summary.",
    "parity.table.object": "Object",
    "parity.table.done": "Implemented",
    "parity.table.status": "Status",
    "parity.tag.ready": "ready",
    "parity.tag.non_goals": "5 non-goals",
    "parity.details": "Details:",
    "simd.title": "SIMD (optional)",
    "simd.lead":
      "AMD64 + Go 1.26+ + <code>GOEXPERIMENT=simd</code> for accelerated Sum/Min/Max on float64.",
    "simd.fallback":
      "Without the flag — automatic scalar fallback with identical results.",
    "copy": "Copy",
    "copy.done": "Copied",
    "copy.error": "Error",
    "lang.label": "Language",
  },
};

(function () {
  const STORAGE_KEY = "gopolars-docs-lang";
  const DEFAULT_LANG = "uk";

  function i18nReady() {
    return window.GOPOLARS_I18N && window.GOPOLARS_I18N.uk && window.GOPOLARS_I18N.en;
  }

  function langFromURL() {
    const q = new URLSearchParams(location.search).get("lang");
    return q === "uk" || q === "en" ? q : null;
  }

  function readStoredLang() {
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved && window.GOPOLARS_I18N && window.GOPOLARS_I18N[saved]) return saved;
    } catch (_) {
      /* file:// або блокування localStorage */
    }
    return null;
  }

  function writeStoredLang(lang) {
    try {
      localStorage.setItem(STORAGE_KEY, lang);
    } catch (_) {
      /* ігноруємо */
    }
  }

  function getLang() {
    const urlLang = langFromURL();
    if (urlLang) return urlLang;
    const stored = readStoredLang();
    if (stored) return stored;
    const nav = (navigator.language || "").toLowerCase();
    if (nav.startsWith("uk")) return "uk";
    if (nav.startsWith("en")) return "en";
    return DEFAULT_LANG;
  }

  function t(lang, key) {
    const pack = window.GOPOLARS_I18N && window.GOPOLARS_I18N[lang];
    return pack && pack[key] != null ? pack[key] : key;
  }

  function syncURL(lang) {
    try {
      const url = new URL(location.href);
      if (url.searchParams.get("lang") === lang) return;
      url.searchParams.set("lang", lang);
      history.replaceState(null, "", url.pathname + url.search + url.hash);
    } catch (_) {
      /* file:// */
    }
  }

  function applyLanguage(lang) {
    if (!i18nReady()) {
      console.error("[gopolars docs] GOPOLARS_I18N не завантажено — перевірте assets/i18n.js");
      return;
    }
    const pack = window.GOPOLARS_I18N[lang];
    if (!pack) return;

    document.documentElement.lang = lang;
    document.documentElement.dataset.docsLang = lang;

    document.querySelectorAll("[data-i18n]").forEach((el) => {
      const key = el.getAttribute("data-i18n");
      const value = pack[key];
      if (value == null) return;
      el.innerHTML = value;
    });

    const pill = document.getElementById("docs-lang-pill");
    if (pill) {
      pill.textContent = lang === "en" ? "English" : "Українська";
      pill.hidden = false;
    }

    document.querySelectorAll("[data-i18n-attr]").forEach((el) => {
      const spec = el.getAttribute("data-i18n-attr");
      spec.split(";").forEach((pair) => {
        const [attr, key] = pair.split(":").map((s) => s.trim());
        const value = pack[key];
        if (attr && value != null) el.setAttribute(attr, value);
      });
    });

    const titleKey = document.querySelector("title")?.getAttribute("data-i18n");
    if (titleKey && pack[titleKey]) {
      document.title = pack[titleKey];
    }

    document.querySelectorAll(".lang-btn").forEach((btn) => {
      const active = btn.getAttribute("data-lang") === lang;
      btn.classList.toggle("active", active);
      btn.setAttribute("aria-pressed", active ? "true" : "false");
    });

    document.querySelectorAll(".lang-link").forEach((link) => {
      link.classList.toggle("active", link.getAttribute("data-lang") === lang);
    });

    document.querySelectorAll(".copy-btn").forEach((btn) => {
      btn.textContent = t(lang, "copy");
    });

    writeStoredLang(lang);

    if (langFromURL() !== lang) {
      syncURL(lang);
    }
  }

  function highlightCode() {
    if (typeof Prism === "undefined") return;
    document.querySelectorAll("pre.code-block code").forEach((block) => {
      Prism.highlightElement(block);
    });
  }

  function setupCopyButtons() {
    const lang = getLang();
    document.querySelectorAll("pre.code-block").forEach((block) => {
      if (block.querySelector(".copy-btn")) return;
      const btn = document.createElement("button");
      btn.className = "copy-btn";
      btn.type = "button";
      btn.textContent = t(lang, "copy");
      btn.addEventListener("click", async () => {
        const code = block.querySelector("code");
        if (!code) return;
        const text = code.textContent || "";
        try {
          await navigator.clipboard.writeText(text);
          btn.textContent = t(getLang(), "copy.done");
          setTimeout(() => {
            btn.textContent = t(getLang(), "copy");
          }, 1600);
        } catch {
          btn.textContent = t(getLang(), "copy.error");
        }
      });
      block.appendChild(btn);
    });
  }

  function langHref(lang) {
    try {
      const u = new URL(location.href);
      u.searchParams.set("lang", lang);
      return u.pathname + u.search + u.hash;
    } catch {
      return `?lang=${lang}`;
    }
  }

  function setupLanguageSwitch() {
    const current = getLang();
    document.querySelectorAll(".lang-btn[data-lang], .lang-footer-links .lang-link[data-lang]").forEach((el) => {
      const lang = el.getAttribute("data-lang");
      if (lang) el.href = langHref(lang);
      const active = lang === current;
      el.classList.toggle("active", active);
      if (el.classList.contains("lang-btn")) {
        el.setAttribute("aria-pressed", active ? "true" : "false");
      }
    });
  }

  function setupNavHighlight() {
    const sections = [...document.querySelectorAll("main section[id]")];
    const navLinks = [...document.querySelectorAll(".nav-group a[href^='#']")];
    if (!sections.length || !navLinks.length) return;

    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (!entry.isIntersecting) return;
          const id = entry.target.id;
          navLinks.forEach((link) => {
            link.classList.toggle("active", link.getAttribute("href") === `#${id}`);
          });
        });
      },
      { rootMargin: "-30% 0px -60% 0px", threshold: 0 }
    );
    sections.forEach((s) => observer.observe(s));
  }

  function setupSearch() {
    const search = document.getElementById("api-search");
    if (!search) return;
    search.addEventListener("input", () => {
      const q = search.value.trim().toLowerCase();
      document.querySelectorAll("[data-search]").forEach((row) => {
        const text = row.getAttribute("data-search") || "";
        const cells = row.textContent || "";
        row.hidden =
          q !== "" && !text.toLowerCase().includes(q) && !cells.toLowerCase().includes(q);
      });
    });
  }

  function init() {
    if (!i18nReady()) {
      console.error("[gopolars docs] Немає перекладів — assets/i18n.js");
      return;
    }
    setupCopyButtons();
    setupLanguageSwitch();
    setupNavHighlight();
    setupSearch();
    applyLanguage(getLang());
    highlightCode();
    window.gopolarsDocsHighlight = highlightCode;
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
