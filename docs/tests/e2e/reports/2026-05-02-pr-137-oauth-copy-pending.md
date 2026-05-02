---
pr: 137
commits:
  - f20cffd9bc
  - c1ffac9324
tested_against: https://hackforger.inside.h2os.cloud
tested_at: null
e2e_owner: pending
admin_signoff: null
---

# PR #137 — OAuth button + entry-link copy (pending verification)

> ⚠️ **This is a placeholder.** PR #137 ("OAuth button → 'Continue with X', entry
> links → '登录/注册'") was merged to `v0.1-dev/hackforger` before the smoke-test
> gate (`docs/notes/gitflow.md`) was instituted. Before this commit can be
> promoted to `prod` and deployed to `https://www.synnovator.com`, the
> following needs to happen.

## Stage 1 — Claude E2E (fills in `tested_at` + `e2e_owner`)

```bash
git checkout v0.1-dev/hackforger
bash scripts/restart-gitea.sh   # rebuilds + restarts the Mac instance
```

Open `https://hackforger.inside.h2os.cloud/` in a browser (or via
`agent-browser`) and verify:

1. **Landing page top-right** — link reads "登录 / 注册" (zh-CN) or
   "Sign in / Register" (en-US), not the old "Sign In" only.
   - Screenshot: `landing-topright-zh.png`, `landing-topright-en.png`
2. **Click the link** — redirects through Logto.
3. **Logto consent page** — primary button reads
   "继续使用 Logto" / "Continue with Logto" (provider name from DB), not the
   old "Sign in with Logto" string.
   - Screenshot: `logto-consent-button.png`
4. **Successful login** — lands on `/dashboard` as the logged-in user.

Replace this section with the actual run notes + `tested_at` + `e2e_owner`
once the run is done.

## Stage 2 — Admin manual sign-off (fills in `admin_signoff`)

After Stage 1 is committed, an admin pulls dev locally and walks the same
steps. They edit this file's frontmatter:

```yaml
admin_signoff:
  by: <admin handle>
  at: 2026-05-02T15:00:00+0800
  notes: "tested both locales; flow clean"
```

The `deploy/ecs/preflight.sh` gate refuses to deploy until `admin_signoff` is
non-null.

## Why this PR is the first under the new gate

PR #137 + the dark-theme follow-up #138 both shipped to `v0.1-dev/hackforger`
during the cloud-migration plumbing work (May 1–2, 2026). The cloud was
deployed from a snapshot **before** these PRs landed
(commit `1b6a3932b6` baseline = "scaffolding only"). So the cloud is
currently behind dev by these two commits.

When admin is ready to ship the auth/landing improvements to production, this
report (or one like it) needs to be filled in first.
