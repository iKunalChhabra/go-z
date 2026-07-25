(() => {
  const contentEl = document.getElementById("content");
  const sidebarEl = document.getElementById("sidebar");
  const tocEl = document.getElementById("toc");
  const pagerEl = document.getElementById("pager");
  const searchModal = document.getElementById("searchModal");
  const searchResults = document.getElementById("searchResults");
  const searchModalInput = document.getElementById("searchModalInput");
  const searchInput = document.getElementById("searchInput");

  const cache = new Map();
  let searchIndex = null;

  function pathToFile(path) {
    if (!path || path === "/") return "content/index.md";
    return `content${path}.md`;
  }

  function currentPath() {
    const raw = location.hash.replace(/^#/, "") || "/";
    // Support "#/path" and legacy "#path"
    const path = raw.startsWith("/") ? raw : `/${raw}`;
    // Strip secondary heading fragment if somehow present in hash path
    return path.split("#")[0] || "/";
  }

  function headingFragment() {
    // "#/guide/errors#zoderror" → "zoderror"
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
        const heading = title ? `<p><strong>${title.trim()}</strong></p>\n` : "";
        return `<div class="callout ${type}"><div class="icon">${icon}</div><div class="body">\n\n${heading}${body}\n</div></div>\n`;
      }
    );
  }

  function renderToc(root) {
    const headings = [...root.querySelectorAll("h2, h3")];
    if (!headings.length) {
      tocEl.innerHTML = "";
      return;
    }
    const links = headings
      .map((h) => {
        if (!h.id) h.id = slugify(h.textContent || "section");
        const depth = h.tagName === "H3" ? "depth-3" : "";
        return `<a class="${depth}" href="#${currentPath()}#${h.id}">${h.textContent}</a>`;
      })
      .join("");
    tocEl.innerHTML = `<h3>On this page</h3>${links}`;
  }

  function decorateCodeBlocks(root) {
    root.querySelectorAll("pre").forEach((pre) => {
      if (pre.querySelector(".copy-btn")) return;
      const btn = document.createElement("button");
      btn.className = "copy-btn";
      btn.type = "button";
      btn.textContent = "Copy";
      btn.addEventListener("click", async () => {
        const code = pre.querySelector("code")?.textContent || pre.textContent;
        await navigator.clipboard.writeText(code);
        btn.textContent = "Copied";
        setTimeout(() => (btn.textContent = "Copy"), 1200);
      });
      pre.appendChild(btn);
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
          ? `<a href="#${prev.path}"><span class="label">Previous</span><span class="title">← ${prev.title}</span></a>`
          : `<span></span>`
      }
      ${
        next
          ? `<a class="next" href="#${next.path}"><span class="label">Next</span><span class="title">${next.title} →</span></a>`
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
      // marked v9+ uses token object; v8 used (text, level)
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
      // Ensure IDs even if renderer path differed
      contentEl.querySelectorAll("h1, h2, h3, h4").forEach((h) => {
        if (!h.id) h.id = slugify(h.textContent || "section");
      });
      decorateCodeBlocks(contentEl);
      renderToc(contentEl);
      renderPager(path);

      const page = DOCS_PAGES.find((p) => p.path === path);
      document.title = page ? `${page.title} · go-zod` : "go-zod — Zod for Go";

      const frag = headingFragment();
      if (frag) {
        const el = document.getElementById(frag);
        if (el) el.scrollIntoView({ block: "start" });
      } else {
        window.scrollTo({ top: 0 });
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
    searchModalInput.value = "";
    searchResults.innerHTML = `<div class="search-empty">Type to search across ${DOCS_PAGES.length} pages</div>`;
    searchModalInput.focus();
    buildSearchIndex().then(() => {
      if (searchModalInput.value) runSearch(searchModalInput.value);
    });
  }
  function closeSearch() {
    searchModal.classList.remove("open");
  }

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

  if (!location.hash) location.hash = "#/";
  render(currentPath());
})();
