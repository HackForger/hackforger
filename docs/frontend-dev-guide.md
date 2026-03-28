# HackForger Frontend Development Guide

## Architecture: SSR-First + Vue Enhancement

Forgejo is **not** a SPA. It uses server-side rendering (Go templates) with Vue 3 components for interactive enhancements on specific DOM elements.

```
Page Load Flow:
  Server (Go template) → Full HTML → Browser renders
  → onDomReady() → Vue mounts on specific elements → Interactive UI
```

## Vue Component Patterns

### Mounting

1. Go template renders a `<div id="hackforger-xxx" data-*="...">` placeholder
2. `web_src/js/features/hackforger/init.js` detects the element by ID
3. Dynamic import loads the Vue component (webpack code splitting)
4. `createApp(Component, props).mount(el)` — props from `data-*` attributes

### API Calls: Web Routes, NOT API Routes

**Rule: Vue components must call web routes (`POST /{owner}/{repo}/...`), not API routes (`/api/v1/...`).**

| Route type | Auth mechanism | Used by |
|-----------|---------------|---------|
| Web routes (`/{owner}/{repo}/...`) | Session cookie (automatic) | Browser JS, Vue components, form submissions |
| API routes (`/api/v1/...`) | Token (`?token=` or `Authorization` header) | CLI tools, CI/CD, third-party integrations, Agents |

**Why:** Browser sessions have cookies but no API token. Forgejo's `POST/PUT/DELETE` from `modules/fetch.js` are simple fetch wrappers that do NOT auto-attach tokens. Same-origin web routes receive session cookies automatically.

**Example — correct:**
```javascript
// Vue computed
apiBase() {
  return `/${this.repoOwner}/${this.repoName}/bounties/${this.bountyId}`;
}
// Vue method
async completeBounty() {
  await POST(this.apiBase + '/complete');  // Web route, session auth
}
```

**Example — wrong:**
```javascript
apiBase() {
  return `/api/v1/repos/${this.repoOwner}/${this.repoName}/bounties/${this.bountyId}`;
}
// ❌ API route requires token, will fail with "token is required"
```

### Forgejo Fetch Helpers

Import from `web_src/js/modules/fetch.js`:
```javascript
import {GET, POST, PUT, DELETE} from '../../modules/fetch.js';
```

These are thin wrappers around `fetch()`. They handle `Content-Type` for JSON/FormData but do NOT handle auth — auth comes from the browser session cookie on same-origin requests.

## Vue 3 Options API

All existing Forgejo Vue components use **Options API** (not Composition API). New HackForger components should follow this convention:

```javascript
export default {
  props: { ... },
  data() { return { ... }; },
  computed: { ... },
  mounted() { ... },
  methods: { ... },
};
```

## CSS Conventions

- **Tailwind CSS**: All classes use `tw-` prefix (configured with `prefix: "tw-"`, `important: true`)
- **Fomantic UI**: Use for dropdown, modal, form, button, label, segment, table components
- **Theme support**: New components must work in both light/dark themes via CSS variables

## Web Route Patterns for Vue Actions

When a Vue component needs to trigger server-side state changes, create paired web routes:

```go
// routers/web/web.go — inside repo group
m.Group("/bounties/{bounty_id}", func() {
    m.Get("/applications", handler.ListApplications)  // JSON response
    m.Post("/{action}", handler.BountyAction)          // action = complete/cancel/review/...
    m.Post("/winners", handler.SelectWinners)           // JSON body
})

// routers/web/hackforger/bounty.go — handlers return JSON
func BountyAction(ctx *context.Context) {
    // Parse action from path, call service, return JSON
    ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
```

## File Organization

```
web_src/js/
  features/hackforger/init.js     — Mount points (lazy import)
  components/hackforger/*.vue     — Vue SFC components
  modules/fetch.js                — Forgejo fetch helpers (shared)
```
