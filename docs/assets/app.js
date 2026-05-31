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
      if (el.closest(".lang-switch")) return;
      const key = el.getAttribute("data-i18n");
      const value = pack[key];
      if (value == null) return;
      el.innerHTML = value;
    });

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
    syncURL(lang);
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

  function bindLangControl(el) {
    const lang = el.getAttribute("data-lang");
    if (!lang) return;

    if (el.tagName === "A") {
      el.addEventListener("click", (e) => {
        if (!i18nReady()) return;
        e.preventDefault();
        applyLanguage(lang);
      });
      return;
    }

    el.addEventListener("click", () => {
      if (!i18nReady()) return;
      applyLanguage(lang);
    });
  }

  function setupLanguageSwitch() {
    document.querySelectorAll(".lang-btn, .lang-link[data-lang]").forEach(bindLangControl);
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
