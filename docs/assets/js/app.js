(() => {
  const contentEl = document.getElementById("content");
  const sidebarEl = document.getElementById("sidebar");
  const tocEl = document.getElementById("toc");
  const pagerEl = document.getElementById("pager");
  const searchModal = document.getElementById("searchModal");
  const searchResults = document.getElementById("searchResults");
  const searchModalInput = document.getElementById("searchModalInput");
  const searchInput = document.getElementById("searchInput");
  const themeToggle = document.getElementById("themeToggle");

  const cache = new Map();
  let searchIndex = null;
  let tocObserver = null;

  function pathToFile(path) {
    if (!path || path === "/") return "content/index.md";
    return `content${path}.md`;
  }

  function currentPath() {
    const raw = location.hash.replace(/^#/, "") || "/";
    const path = raw.startsWith("/") ? raw : `/${raw}`;
    return path.split("#")[0] || "/";
  }

  function headingFragment() {
    const parts = location.hash.split("#").filter(Boolean);
    return parts.length >= 2 && parts[0].startsWith("/") ? parts[1] : null;
  }

  function renderSidebar(active) {
    sidebarEl.innerHTML = DOCS_NAV.map((group) => {
      const links = group.items
        .map((item) => {
          const cls = item.path === active ? "active" : "";
          return `<a class="${cls}" href="#${item.path}">${item.title}</a>`;
        })
        .join("");
      return `<div class="nav-group"><h3>${group.title}</h3>${links}</div>`;
    }).join("");
  }

  function slugify(text) {
    return String(text)
      .toLowerCase()
      .replace(/<[^>]+>/g, "")
      .replace(/&[a-z]+;/g, "")
      .replace(/[^\w\s-]/g, "")
      .trim()
      .replace(/\s+/g, "-");
  }

  function enhanceMarkdown(md) {
    return md.replace(
      /:::(\w+)(?:\s+([^\n]+))?\n([\s\S]*?):::/g,
      (_, type, title, body) => {
        const icons = { tip: "✓", info: "i", warn: "!", danger: "×" };
        const icon = icons[type] || "i";
        // The blank line matters: without it marked swallows the body into the
        // preceding raw HTML block and inline markdown stops rendering.
        const heading = title ? `<p class="callout-title">${title.trim()}</p>\n\n` : "";
        return `<div class="callout ${type}"><div class="icon">${icon}</div><div class="body">\n\n${heading}${body}\n</div></div>\n`;
      }
    );
  }

  function triggerPageEnter(root) {
    root.classList.remove("page-enter");
    // force reflow so the animation restarts on every route change
    void root.offsetWidth;
    root.classList.add("page-enter");
    const hero = root.querySelector(".hero");
    if (hero) {
      hero.classList.remove("page-enter");
      void hero.offsetWidth;
      hero.classList.add("page-enter");
    }
  }

  function renderToc(root) {
    if (tocObserver) {
      tocObserver.disconnect();
      tocObserver = null;
    }
    const headings = [...root.querySelectorAll("h2, h3")];
    if (!headings.length) {
      tocEl.innerHTML = "";
      return;
    }
    const links = headings
      .map((h) => {
        if (!h.id) h.id = slugify(h.textContent || "section");
        const depth = h.tagName === "H3" ? "depth-3" : "";
        return `<a class="${depth}" href="#${currentPath()}#${h.id}" data-id="${h.id}">${h.textContent}</a>`;
      })
      .join("");
    tocEl.innerHTML = `<h3>On this page</h3>${links}`;

    const tocLinks = [...tocEl.querySelectorAll("a[data-id]")];
    const byId = new Map(tocLinks.map((a) => [a.dataset.id, a]));

    const setActive = (id) => {
      tocLinks.forEach((a) => a.classList.toggle("active", a.dataset.id === id));
    };

    tocObserver = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((e) => e.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top);
        if (visible[0]) setActive(visible[0].target.id);
      },
      { rootMargin: "-64px 0px -70% 0px", threshold: [0, 1] }
    );
    headings.forEach((h) => tocObserver.observe(h));
    if (headings[0]) setActive(headings[0].id);
  }

  function langFromCode(code) {
    const cls = [...(code?.classList || [])].find((c) => c.startsWith("language-"));
    return cls ? cls.replace("language-", "") : "text";
  }

  function decorateCodeBlocks(root) {
    root.querySelectorAll("pre").forEach((pre) => {
      if (pre.closest(".code-block")) return;
      const code = pre.querySelector("code");
      const lang = langFromCode(code);

      const wrap = document.createElement("div");
      wrap.className = "code-block";

      const header = document.createElement("div");
      header.className = "code-block-header";
      header.innerHTML = `<span class="code-lang">${lang}</span>`;

      const btn = document.createElement("button");
      btn.className = "copy-btn";
      btn.type = "button";
      btn.textContent = "Copy";
      btn.addEventListener("click", async () => {
        const text = code?.textContent || pre.textContent;
        try {
          await navigator.clipboard.writeText(text);
        } catch {
          /* ignore */
        }
        btn.textContent = "✓";
        btn.classList.add("copied");
        setTimeout(() => {
          btn.textContent = "Copy";
          btn.classList.remove("copied");
        }, 1200);
      });
      header.appendChild(btn);

      pre.parentNode.insertBefore(wrap, pre);
      wrap.appendChild(header);
      wrap.appendChild(pre);
    });

    if (window.hljs) {
      root.querySelectorAll("pre code").forEach((block) => {
        try {
          hljs.highlightElement(block);
        } catch {
          /* ignore unknown langs */
        }
      });
    }
  }

  // Wide tables (the comparison matrix especially) need their own scroll area
  // so they never blow out the prose column.
  function decorateTables(root) {
    root.querySelectorAll("table").forEach((table) => {
      if (table.closest(".table-wrap")) return;
      const wrap = document.createElement("div");
      const cols = table.querySelector("tr")?.children.length || 0;
      wrap.className = cols > 4 ? "table-wrap wide" : "table-wrap";
      table.parentNode.insertBefore(wrap, table);
      wrap.appendChild(table);
    });
  }

  function renderPager(path) {
    const idx = DOCS_PAGES.findIndex((p) => p.path === path);
    if (idx < 0) {
      pagerEl.hidden = true;
      return;
    }
    const prev = DOCS_PAGES[idx - 1];
    const next = DOCS_PAGES[idx + 1];
    pagerEl.hidden = false;
    pagerEl.innerHTML = `
      ${
        prev
          ? `<a href="#${prev.path}"><span class="label">Previous</span><span class="title"><span class="arrow">←</span> ${prev.title}</span></a>`
          : `<span></span>`
      }
      ${
        next
          ? `<a class="next" href="#${next.path}"><span class="label">Next</span><span class="title">${next.title} <span class="arrow">→</span></span></a>`
          : `<span></span>`
      }
    `;
  }

  async function loadMarkdown(path) {
    if (cache.has(path)) return cache.get(path);
    const url = pathToFile(path);
    const res = await fetch(url);
    if (!res.ok) throw new Error(`Missing page: ${url}`);
    const text = await res.text();
    cache.set(path, text);
    return text;
  }

  async function render(path) {
    renderSidebar(path);
    contentEl.innerHTML = `<div class="loading">Loading…</div>`;
    try {
      const md = enhanceMarkdown(await loadMarkdown(path));
      const renderer = new marked.Renderer();
      renderer.heading = function (arg, levelMaybe) {
        let text;
        let level;
        if (typeof arg === "object" && arg !== null) {
          text = arg.text;
          level = arg.depth;
        } else {
          text = arg;
          level = levelMaybe;
        }
        const id = slugify(text);
        return `<h${level} id="${id}">${text}</h${level}>\n`;
      };

      contentEl.innerHTML = marked.parse(md, { renderer, gfm: true });
      contentEl.querySelectorAll("h1, h2, h3, h4").forEach((h) => {
        if (!h.id) h.id = slugify(h.textContent || "section");
      });
      decorateCodeBlocks(contentEl);
      decorateTables(contentEl);
      renderToc(contentEl);
      renderPager(path);
      triggerPageEnter(contentEl);

      const page = DOCS_PAGES.find((p) => p.path === path);
      document.title = page ? `${page.title} · go-z` : "go-z — schema-first validation for Go";

      const frag = headingFragment();
      if (frag) {
        const el = document.getElementById(frag);
        if (el) el.scrollIntoView({ block: "start" });
      } else {
        // Instant scroll on route change (no global smooth)
        window.scrollTo(0, 0);
      }
      document.body.classList.remove("sidebar-open");
    } catch (err) {
      contentEl.innerHTML = `<div class="error-state"><h1>Page not found</h1><p>${String(
        err.message || err
      )}</p><p><a href="#/">Back home</a></p></div>`;
      tocEl.innerHTML = "";
      pagerEl.hidden = true;
    }
  }

  async function buildSearchIndex() {
    if (searchIndex) return searchIndex;
    const entries = [];
    await Promise.all(
      DOCS_PAGES.map(async (page) => {
        try {
          const md = await loadMarkdown(page.path);
          const text = md
            .replace(/```[\s\S]*?```/g, " ")
            .replace(/:::[\s\S]*?:::/g, " ")
            .replace(/[#>*`_\[\]()!-]/g, " ")
            .replace(/\s+/g, " ")
            .trim();
          entries.push({
            path: page.path,
            title: page.title,
            group: page.group,
            text: text.slice(0, 5000),
          });
        } catch {
          /* skip */
        }
      })
    );
    searchIndex = entries;
    return searchIndex;
  }

  function runSearch(q) {
    const query = q.trim().toLowerCase();
    if (!query) {
      searchResults.innerHTML = `<div class="search-empty">Type to search across ${DOCS_PAGES.length} pages</div>`;
      return;
    }
    const terms = query.split(/\s+/).filter(Boolean);
    const hits = searchIndex
      .map((e) => {
        const title = e.title.toLowerCase();
        const group = e.group.toLowerCase();
        const body = e.text.toLowerCase();
        let score = 0;
        for (const t of terms) {
          if (title.includes(t)) score += 5;
          if (group.includes(t)) score += 2;
          if (body.includes(t)) score += 1;
        }
        return { e, score };
      })
      .filter((x) => x.score > 0)
      .sort((a, b) => b.score - a.score)
      .slice(0, 14);
    if (!hits.length) {
      searchResults.innerHTML = `<div class="search-empty">No results for “${q.replace(
        /[<>]/g,
        ""
      )}”</div>`;
      return;
    }
    searchResults.innerHTML = hits
      .map(
        ({ e }, i) =>
          `<a href="#${e.path}" data-idx="${i}" ${i === 0 ? 'aria-selected="true"' : ""}>
            <div class="path">${e.group}</div>
            <div class="title">${e.title}</div>
          </a>`
      )
      .join("");
  }

  function openSearch() {
    searchModal.classList.add("open");
    searchModal.setAttribute("aria-hidden", "false");
    searchModalInput.value = "";
    searchResults.innerHTML = `<div class="search-empty">Type to search across ${DOCS_PAGES.length} pages</div>`;
    searchModalInput.focus();
    buildSearchIndex().then(() => {
      if (searchModalInput.value) runSearch(searchModalInput.value);
    });
  }
  function closeSearch() {
    searchModal.classList.remove("open");
    searchModal.setAttribute("aria-hidden", "true");
  }

  function applyTheme(theme) {
    const next = theme === "dark" ? "dark" : "light";
    document.documentElement.dataset.theme = next;
    localStorage.setItem("go-z-theme", next);
    if (themeToggle) {
      themeToggle.setAttribute("aria-label", next === "dark" ? "Switch to light theme" : "Switch to dark theme");
    }
  }

  function initTheme() {
    const stored = localStorage.getItem("go-z-theme");
    if (stored === "light" || stored === "dark") {
      applyTheme(stored);
      return;
    }
    const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
    applyTheme(prefersDark ? "dark" : "light");
  }

  // Smooth-scroll only for same-page heading anchors
  document.addEventListener("click", (e) => {
    const a = e.target.closest('a[href*="#"]');
    if (!a) return;
    const href = a.getAttribute("href") || "";
    if (!href.includes("#")) return;
    const parts = href.replace(/^#/, "").split("#").filter(Boolean);
    // "#/path#heading" or just "#heading" relative patterns used by TOC
    let frag = null;
    let pathPart = null;
    if (href.startsWith("#/") || href.startsWith("#")) {
      if (parts.length >= 2 && parts[0].startsWith("/")) {
        pathPart = parts[0];
        frag = parts[1];
      }
    }
    if (!frag || !pathPart) return;
    if (pathPart !== currentPath()) return;
    const el = document.getElementById(frag);
    if (!el) return;
    e.preventDefault();
    el.scrollIntoView({ behavior: "smooth", block: "start" });
    history.replaceState(null, "", `#${pathPart}#${frag}`);
  });

  window.addEventListener("hashchange", () => render(currentPath()));
  document.getElementById("menuToggle").addEventListener("click", () => {
    document.body.classList.toggle("sidebar-open");
  });
  searchInput.addEventListener("focus", openSearch);
  searchInput.addEventListener("click", openSearch);
  searchModal.addEventListener("click", (e) => {
    if (e.target === searchModal) closeSearch();
  });
  searchModalInput.addEventListener("input", (e) => runSearch(e.target.value));
  searchResults.addEventListener("click", (e) => {
    if (e.target.closest("a")) closeSearch();
  });
  document.addEventListener("keydown", (e) => {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
      e.preventDefault();
      openSearch();
    }
    if (e.key === "Escape") closeSearch();
  });

  if (themeToggle) {
    themeToggle.addEventListener("click", () => {
      const cur = document.documentElement.dataset.theme === "dark" ? "dark" : "light";
      applyTheme(cur === "dark" ? "light" : "dark");
    });
  }

  initTheme();
  if (!location.hash) location.hash = "#/";
  render(currentPath());
})();
