---
pr: <fill-in-after-PR-opened>
commits:
  - f0c08420ab
  - cf386028b3
  - 01527c286e
  - a4e45cd727
  - 39200bd4c5
  - 07527ddf6c
tested_against: https://hackforger.inside.h2os.cloud
tested_at: 2026-05-03T21:10:00+08:00
e2e_owner: claude
admin_signoff: null
---

# Hackathon Registration: org-owned repos now appear in dropdown + API parity

## Stage 1 — Claude E2E

Reproduced the bug from #145 (linyilun, member of `H2OSLabs`, couldn't register `opc-2026-shuzhi-w1` with `H2OSLabs/page.h2oslabs.com`) and verified the fix end-to-end on Mac dev (port 3000, mirrors hackforger.inside.h2os.cloud).

### Web flow (agent-browser)

Logged in as `linyilun`, opened `/hackathon/opc-2026-shuzhi-w1`, inspected the repo dropdown.

**Before fix**: dropdown contained only the "— 选择仓库 —" placeholder option. Reproduced via the static state of the deployed version `1b6a3932b6`.

**After fix**: dropdown contained 10 options + placeholder. The new SearchRepository query (`Actor=user, Private=true, no OwnerID filter`) returns visible repos, post-filtered to Write+ access. Sample of what showed up for linyilun:

```
haiquan/AiRead
yanglaiyang/Alpha.deeprich
biMetaverse/AuroBIM
baoai/baoai-content-producer
moss/chasingdream
cedar/contextgen-life
contextgen-tech/contextgen-life      ← org-owned (org type=1)
gptmaas/decision-assistant-agent
jjkyao/kilo
linxi/LinxiHaven
```

Of the 10, **only `contextgen-tech/contextgen-life` (id 71) is owned by an Organization (type=1)**. The other 9 are user-owned (type=0) accounts whose names visually resemble orgs (e.g. `gptmaas`, `cedar`, `moss`). The dropdown now shows all of them — the bug (empty dropdown) is fixed.

**Org-derivation E2E**: selected `contextgen-tech/contextgen-life` (id 71), set track=26, title "org repo e2e test", clicked 报名. Resulting registration row:

| field | value |
| --- | --- |
| `team_name` | `contextgen-tech` (auto-derived from org name, override worked) |
| `org_id` | `49` (= contextgen-tech user.id with type=1) |
| `repo_id` | `71` |
| `status` | `1` (Approved) |

Screenshots:
- `01-dropdown-with-org-repos.png` — dropdown opened, showing the 10 options
- `04-org-repo-success.png` — post-submit state showing successful registration

### Why didn't I test the literal `H2OSLabs/page.h2oslabs.com`?

`linyilun` is a site admin (`is_admin=true`), so `GetUserRepoPermission` returns admin-level for every repo in the system — about 30+ repos. The dropdown's `dropdownMax = 10` cap (sorted by SearchRepository's natural recency / lexical order) puts `H2OSLabs/page.h2oslabs.com` past the cutoff. This is the documented §7 risk; for non-admin users (the typical case) the cap is plenty.

**Verified separately via API** that `linyilun` CAN register with repo 73 (`H2OSLabs/page.h2oslabs.com`) when explicitly passing `repo_id`: returns `OrgID:56, TeamName:"H2OSLabs", RepoID:73`, status 201, submission created. See API section below.

### API flow (curl)

| Case | Command | Expected | Actual |
| --- | --- | --- | --- |
| 1. Legacy solo `{team_name, track_id}` | `POST /api/v1/hackforger/hackathons/1/register` body `{"team_name":"linyilun-solo","track_id":0}` | 201, status=Approved | ✅ 201, body has `OrgID:0, RepoID:0, Status:1` |
| 2. Org repo binding `{repo_id:73}` | `POST .../1/register` body `{"team_name":"will-be-overridden","track_id":0,"repo_id":73,"title":"H2OSLabs entry"}` | 201, OrgID set, TeamName=H2OSLabs | ✅ 201, body has `OrgID:56, TeamName:"H2OSLabs", RepoID:73`, submission row with title="H2OSLabs entry" |
| 3. Read-only repo | `POST` with `repo_id` of a repo user has only Read access to | 403 RepoAccessDenied | ⚠️ NOT smoke-testable with available users (linyilun is site admin → admin on all). Logic is reviewed in code; admin to validate in Stage 2 with a non-admin user. |
| 4. Non-existent repo | `POST` with `repo_id:999999999` | 404 RepoNotExist | ✅ 404, body `"message":"repository does not exist [id: 999999999, ...]"` |

3/4 cases definitively pass. Case 3 deferred to admin (need a non-admin tester with limited repo access).

### Other observations

- **Web's existing submission creation is silent on failure**: when registering via the web flow with a user-owned repo (`moss/chasingdream`, id 60 on first attempt), the registration row was created but no submission row was created. The web handler logs `CreateSubmission on register: %v` and continues without erroring out. This is pre-existing behavior — the API mirrors it now (parity). Not a regression introduced by this PR.
- **Swagger annotation discrepancy**: `routers/api/v1/hackforger/hackathon_registration.go` has `// swagger:operation POST /hackforger/hackathons/{id}/registrations`, but the actual route is `POST /hackforger/hackathons/{id}/register` (api.go:1771). API docs are wrong but behavior is right. Out of scope for this PR; should be fixed separately.

## Stage 2 — Admin manual sign-off (pending)

Admin will pull this branch, restart Mac instance with the new binary (`bash scripts/restart-gitea.sh` after `cp .claude/worktrees/org-repo-fix/gitea ./gitea && codesign --force --sign - gitea` because macOS Gatekeeper kills replaced binaries — see [troubleshooting note below]).

Then walk through at https://hackforger.inside.h2os.cloud:
- [ ] Login as a non-admin user (e.g. test account, or use SynNovator's admin to mint a token for jjkyao for Case 3 verification)
- [ ] Repo dropdown visible on `/hackathon/opc-2026-shuzhi-w1` for the test user
- [ ] Pick an org-owned repo (org-team grant with Write+) and complete registration
- [ ] Verify: registration row has correct `org_id`, `team_name = org_name`
- [ ] (Case 3) Try registering with a read-only repo via API → expect 403 RepoAccessDenied
- [ ] Edit `admin_signoff` below with handle, ISO timestamp, notes

After admin signoff, this report unblocks `deploy/ecs/preflight.sh` for cloud deploy.

### Troubleshooting note (encountered during smoke)

**macOS Gatekeeper kills in-place binary replacements**. After `cp .claude/worktrees/org-repo-fix/gitea ./gitea`, running `./gitea --version` exits 137 (SIGKILL). Fix: `codesign --force --sign - gitea` after each copy. The script `scripts/restart-gitea.sh` doesn't do this automatically because it normally rebuilds in-place via `make build` (which signs as part of the Go toolchain output). For worktree-built binaries copied over, the sign step is needed.

This is a per-machine issue (only happens on macOS), not a code bug. Worth a one-liner addition to `scripts/restart-gitea.sh` if it bites again.
