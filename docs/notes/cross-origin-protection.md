# Cross-Origin Protection (CSRF) Behind Reverse Proxy

## Background

Forgejo (Go 1.26+) uses Go's built-in `net/http.CrossOriginProtection` for CSRF protection. This is **NOT** the traditional CSRF-token-in-form approach — there is no `_csrf` hidden input and no `CsrfTokenHtml` template variable.

### How It Works

`CrossOriginProtection` protects against CSRF by checking browser-sent headers on non-safe (POST/PUT/DELETE) requests:

1. **`Sec-Fetch-Site` header** (primary) — Modern browsers (all since 2023) send this. If value is `cross-site`, the request is rejected.
2. **`Origin` vs `Host` header** (fallback) — If `Sec-Fetch-Site` is absent, compares the `Origin` header's hostname with the `Host` header. Mismatch → 403 Forbidden.
3. **No headers** — Requests without either header (e.g., curl, API clients) are assumed non-browser and allowed through.
4. **Safe methods** (GET, HEAD, OPTIONS) are always allowed.

## The Reverse Proxy Problem

When Forgejo runs behind a reverse proxy:

- Browser sends `Origin: https://hackforger.example.invalid` (an example public URL)
- The `Host` header may or may not match, depending on proxy configuration
- If `Sec-Fetch-Site` is `same-origin`, everything works
- If the browser omits `Sec-Fetch-Site` (older browser, non-standard client), Go falls back to `Origin` vs `Host` comparison

### Symptom

Form POST submissions (manage page buttons, judge scoring, registration) silently fail — the browser sends the request but receives a 403, and the page just reloads without any visible error.

### Root Cause History

This was originally misdiagnosed as "missing CSRF token" (Bug #3, #4 in the E2E report). We spent multiple debugging sessions trying to add `{{.CsrfTokenHtml}}` to templates — but **this variable does not exist in Forgejo**. It rendered as empty string, and the real problem was the `CrossOriginProtection` check.

## The Fix

In `routers/web/web.go`, the `CrossOriginProtection` instance reads `ROOT_URL` from `app.ini` and registers it as a trusted origin:

```go
var crossOriginProtection = func() *http.CrossOriginProtection {
    cop := http.NewCrossOriginProtection()
    if appURL := strings.TrimRight(setting.AppURL, "/"); appURL != "" {
        cop.AddTrustedOrigin(appURL)
    }
    return cop
}()
```

This ensures that requests with `Origin: https://hackforger.example.invalid` are accepted even when the `Host` header differs (e.g., `localhost:3000` from the proxy).

### Why Not Hardcode?

The origin is derived from `setting.AppURL` (which reads `[server] ROOT_URL` from `app.ini`). This means:
- Different deployments automatically use the correct origin
- If accessed via a documentation IP (e.g., `http://192.0.2.10:3000`), only the configured `ROOT_URL` origin is trusted — other origins are rejected, which is the correct security behavior

## What NOT To Do

| Anti-pattern | Why it's wrong |
|---|---|
| Add `{{.CsrfTokenHtml}}` to templates | This variable doesn't exist in Forgejo. It renders as empty. |
| Add `_csrf` hidden input fields | Forgejo doesn't check for form-level CSRF tokens. |
| Set `DisableCSRF: true` on routes | Disables all cross-origin protection — security risk. |
| Add `header_up Host {upstream_hostport}` in Caddy | Caddy already preserves Host by default. And this doesn't help if `Sec-Fetch-Site` is the issue. |

## Proxy-Specific Notes

### Caddy example
- Preserves `Host` header by default — no extra config needed
- The `AddTrustedOrigin` fix handles edge cases

### Nginx
- **Requires** `proxy_set_header Host $host;` — without this, nginx rewrites Host to upstream, causing 100% CSRF failure
- Also add: `proxy_set_header X-Real-IP $remote_addr;` and `proxy_set_header X-Forwarded-Proto $scheme;`

## Relevant app.ini Settings

| Setting | Section | Purpose |
|---------|---------|---------|
| `ROOT_URL` | `[server]` | The public URL. Must match what users see in their browser. Used to derive trusted origin. |
| `DOMAIN` | `[server]` | Domain name of server. |
| `PROTOCOL` | `[server]` | How Forgejo listens. Do NOT set to `https` if Caddy handles TLS. |

## References

- Go docs: `go doc net/http.CrossOriginProtection`
- Forgejo code: `routers/web/web.go` line 140
- MDN: [Sec-Fetch-Site](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Sec-Fetch-Site)
