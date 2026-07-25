# go-zod documentation site

Static, dependency-free docs site for GitHub Pages: a hash router over Markdown
in `content/`, rendered client-side.

## Local preview

```bash
cd docs
python3 -m http.server 5173
# open http://127.0.0.1:5173
```

Hash routes: `/#/guide/installation`, `/#/api/string`, …

## Publish on GitHub Pages

**Option A — GitHub Actions (recommended)**  
1. Merge this branch.  
2. Repo **Settings → Pages → Build and deployment → Source: GitHub Actions**.  
3. The workflow `.github/workflows/docs.yml` deploys the `docs/` folder.  
4. Site URL: `https://<user>.github.io/go-zod/`

**Option B — Deploy from branch**  
1. **Settings → Pages → Source: Deploy from a branch**.  
2. Branch: `main` (or this branch), folder: `/docs`.  
3. Save. Same URL as above.

## Authoring

- Pages live in `content/**/*.md`
- Nav is declared in `assets/js/nav.js` (`DOCS_NAV`)
- Callouts:

```md
:::tip Title
body
:::
```

Supported types: `tip`, `info`, `warn`, `danger`.
