// Навігація та копіювання прикладів коду
document.querySelectorAll("pre").forEach((block) => {
  const btn = document.createElement("button");
  btn.className = "copy-btn";
  btn.type = "button";
  btn.textContent = "Копіювати";
  btn.addEventListener("click", async () => {
    const code = block.querySelector("code");
    if (!code) return;
    try {
      await navigator.clipboard.writeText(code.innerText);
      btn.textContent = "Скопійовано";
      setTimeout(() => {
        btn.textContent = "Копіювати";
      }, 1600);
    } catch {
      btn.textContent = "Помилка";
    }
  });
  block.appendChild(btn);
});

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

const search = document.getElementById("api-search");
if (search) {
  search.addEventListener("input", () => {
    const q = search.value.trim().toLowerCase();
    document.querySelectorAll("[data-search]").forEach((row) => {
      const text = row.getAttribute("data-search") || "";
      row.hidden = q !== "" && !text.toLowerCase().includes(q);
    });
  });
}
