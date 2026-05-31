(function () {
  const STORAGE_KEY = "gopolars-docs-lang";
  const DEFAULT_LANG = "uk";

  function getLang() {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved && window.GOPOLARS_I18N[saved]) return saved;
    const nav = navigator.language || "";
    return nav.startsWith("uk") ? "uk" : nav.startsWith("en") ? "en" : DEFAULT_LANG;
  }

  function t(lang, key) {
    const pack = window.GOPOLARS_I18N[lang];
    return pack && pack[key] != null ? pack[key] : key;
  }

  function applyLanguage(lang) {
    const pack = window.GOPOLARS_I18N[lang];
    if (!pack) return;

    document.documentElement.lang = lang;

    document.querySelectorAll("[data-i18n]").forEach((el) => {
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

    document.querySelectorAll(".copy-btn").forEach((btn) => {
      btn.textContent = t(lang, "copy");
    });

    localStorage.setItem(STORAGE_KEY, lang);
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

  function setupLanguageSwitch() {
    document.querySelectorAll(".lang-btn").forEach((btn) => {
      btn.addEventListener("click", () => {
        const lang = btn.getAttribute("data-lang");
        if (!lang || !window.GOPOLARS_I18N[lang]) return;
        applyLanguage(lang);
      });
    });
  }

  function setupNavHighlight() {
    const sections = [...document.querySelectorAll("main section[id]")];
    const navLinks = [...document.querySelectorAll(".nav-group a[href^='#']")];
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

  setupCopyButtons();
  highlightCode();
  setupLanguageSwitch();
  setupNavHighlight();
  setupSearch();
  applyLanguage(getLang());
})();
