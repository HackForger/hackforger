---
pr: 149
commits:
  - 4fef909af5
tested_against: https://hackforger.inside.h2os.cloud
tested_at: 2026-05-03T21:10:00+08:00
e2e_owner: claude
admin_signoff:
  by: allen.woods
  at: 2026-05-03T22:35:00+08:00
  notes: "Stage 2: ran the non-admin user (linyilun) registration flow on Mac dev — '没有问题'. Squash-merged as 4fef909af5 (was originally commits f0c08420ab..b6a7bc750b in branch fix/hackathon-org-repo-registration before squash-merge)."
---

# Hackathon Registration: org-owned repos now appear in dropdown + API parity

## Stage 1 — Claude E2E

Reproduced the bug from #145 (linyilun, member of `H2OSLabs`, couldn't register `opc-2026-shuzhi-w1` with `H2OSLabs/page.h2oslabs.com`) and verified the fix end-to-end on Mac dev (port 3000, mirrors hackforger.inside.h2os.cloud).

### Web flow (agent-browser)

Logged in as `linyilun`, opened `/hackathon/opc-2026-shuzhi-w1`, inspected the repo dropdown.

**Before fix**: dropdown contained only the "— 选择仓库 —" placeholder option. Reproduced via the static state of the deployed version `1b6a3932b6`.

**After fix** (and after the recency-ordering follow-up `4a0d67bb42`): dropdown contained 10 options + placeholder, ordered by `updated_unix DESC`. The new SearchRepository query (`Actor=user, Private=true, no OwnerID filter, OrderBy=SearchOrderByRecentUpdated`) returns visible repos, post-filtered to Write+ access. What linyilun sees now:

```
biMetaverse/AuroBIM                  ← org-owned (id 74)
H2OSLabs/page.h2oslabs.com           ← org-owned (id 73) — THE ORIGINAL TEST TARGET
SynNovatorGroup/SynNovatorRepo       ← org-owned (id 72)
contextgen-tech/contextgen-life      ← org-owned (id 71)
haiquan/track-26                     (id 70)
haiquan/AiRead                       (id 69)
282801145/OPC_Assistant              (id 68)
hulinweilai/Report_claw              (id 67)
xu/xu                                (id 65)
hy_walter/New_repository             (id 66)
```

**4 of the 10 are org-owned** (biMetaverse, H2OSLabs, SynNovatorGroup, contextgen-tech). The original bug — `H2OSLabs/page.h2oslabs.com` being invisible to linyilun — is fixed.

(Initial alphabetical-ordering implementation hid `page.h2oslabs.com` past slot 10 because linyilun is a site admin and gets all 30+ repos in the system. Subagent code-review caught this; commit `4a0d67bb42` switched to `OrderBy=SearchOrderByRecentUpdated` so recently-touched repos surface first. Smoke re-verified: target repo now appears.)

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

### Verified the literal `H2OSLabs/page.h2oslabs.com` target

After the recency-ordering fix in `4a0d67bb42`, this exact repo from the bug report now appears at slot 2 in the dropdown for linyilun (admin) when she views `/hackathon/opc-2026-shuzhi-w1`. Smoke confirmed via agent-browser snapshot of the rendered `<select>`.

Independently verified via API (Case 2 below): `linyilun` can register with `repo_id: 73` and the resulting registration row has `OrgID: 56` (= H2OSLabs.id), `TeamName: "H2OSLabs"` (auto-derived from org name), `RepoID: 73`, plus a submission row.

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
