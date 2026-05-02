# Gitflow + Smoke Test Gate

How feature work moves from a developer's branch to `https://www.synnovator.com`,
and the rule that keeps unverified code from getting promoted.

## Three-stage flow

```
[1] feat/* branch                 ─── developer / Claude does work
       │  PR (Claude opens it)
       ▼
[2] v0.1-dev/hackforger  (= "dev")   ─── integration branch
       │                                 │
       │                                 │  Claude restarts the Mac instance and
       │                                 │  runs E2E smoke against
       │                                 │  https://hackforger.inside.h2os.cloud
       │                                 │  → commits docs/tests/e2e/reports/<file>.md
       │  fast-forward (admin)
       ▼
[3] prod                              ─── release branch
       │                                 │
       │                                 │  Admin restarts the Mac instance against
       │                                 │  the prod tip and manually walks the
       │                                 │  acceptance steps at the same hostname
       │                                 │  → admin signs off in the same report file
       │  bash deploy/ecs/preflight.sh && build-linux.sh && scp + restart
       ▼
[4] www.synnovator.com  (Huawei ECS 203.x)
```

All three test stages — Claude's E2E, admin's manual sign-off, and the eventual
"does it still work" sanity check — happen against the **same Mac instance** at
`https://hackforger.inside.h2os.cloud`. The branch you check out determines what
code is built; the data and OAuth setup don't change between dev/prod testing.

## Branch ownership

| Branch | Who advances it | What lands here | What testing happens |
| --- | --- | --- | --- |
| `feat/<topic>` | anyone (Claude or human) | day-to-day work, scoped to one change | unit + small integration tests |
| `v0.1-dev/hackforger` | PR merge | reviewed feature commits | **Claude E2E** at hackforger.inside.h2os.cloud |
| `prod` | admin (fast-forward only) | a subset of dev that admin has personally walked through | **admin manual sign-off** at hackforger.inside.h2os.cloud |
| (cloud HEAD) | `deploy/ecs/preflight.sh` + scp | whatever's on `prod` at the moment of deploy | **none after deploy** — verification happened on Mac |

`prod` should never have commits that aren't already on `v0.1-dev/hackforger`.
The intent is "prod is dev minus the parts that haven't been admin-verified yet."

## Smoke test report — the contract

Every PR that touches user-facing code MUST produce one report file under
`docs/tests/e2e/reports/`. The file lives forever; it is the audit trail that
proves the change was exercised at hackforger.inside.h2os.cloud.

### Naming convention

```
docs/tests/e2e/reports/YYYY-MM-DD-<short-slug>.md
```

Examples (existing):
- `2026-04-29-landing-page-impl.md`
- `2026-04-21-issue-83-milestones-epoch-date-report.md`

### Required frontmatter

The report MUST have a frontmatter block at the top so the preflight script
can find it:

```markdown
---
pr: 137
commits:
  - f20cffd9bc
  - c1ffac9324
tested_against: https://hackforger.inside.h2os.cloud
tested_at: 2026-05-02T11:00:00+0800
e2e_owner: claude
admin_signoff: null     # filled in step 3
---

# <feature title> — E2E smoke test

(report body — what was tested, screenshots, bugs found, ...)
```

When the admin walks through the same change at stage 3, they edit the same
file and set `admin_signoff` to their handle + timestamp:

```yaml
admin_signoff:
  by: allen
  at: 2026-05-02T15:00:00+0800
  notes: "tested login + register flows in zh-CN, all clean"
```

The cutoff for "user-facing" is generous: any change that affects what an
external user sees, clicks, types into, or reads. See the path list in
`deploy/ecs/preflight.sh` for the exact rule.

## What user-facing means (preflight rule)

Commits touching ONLY these paths can skip the report:

```
deploy/        scripts/        docs/        .github/        .forgejo/
.gitignore     .editorconfig   Makefile     CHANGELOG.md    *.md (root)
```

Any commit touching paths outside that allowlist is "user-facing" and needs a
report. The `preflight.sh` script enforces this.

## Promotion gate (`deploy/ecs/preflight.sh`)

```
Inputs:
  - the local `prod` branch HEAD
  - the SHA last deployed to the cloud (read from
    /var/lib/hackforger/.last-deploy on 203.x via SSH)
Behavior:
  - For each commit between last-deploy SHA and prod HEAD:
      - if commit touches only allowlisted paths → SKIP
      - else search docs/tests/e2e/reports/ for a frontmatter `commits:`
        list that contains this SHA AND a non-null `admin_signoff`
      - if no matching report → FAIL with the SHA + message
  - On success: echo "ready to deploy: <SHA>"
  - On failure: exit 1, print every offending commit
```

If preflight passes, the operator runs:

```bash
bash deploy/ecs/build-linux.sh
scp gitea-linux-amd64 hackforger@203.119.115.130:/opt/hackforger/gitea-new
ssh hackforger@203.119.115.130 \
  'sudo install -m 755 -o hackforger -g hackforger \
     /opt/hackforger/gitea-new /opt/hackforger/gitea && \
   sudo systemctl restart gitea && \
   echo $(git rev-parse HEAD) | sudo tee /var/lib/hackforger/.last-deploy'
```

The last line records the deployed SHA for the next preflight check.

## Worked example — PR #137 ("登录/注册" copy)

The OAuth button + entry-link copy change (PR #137, merged commit `f20cffd9bc`)
is a perfect case study because it has not yet been smoke-tested under this new
process.

### Today's situation

```bash
# What's on dev but not on prod (not yet tested by admin):
git log --oneline origin/v0.1-dev/hackforger ^prod 2>/dev/null
# (assuming prod branch exists and is at the post-clean-slate baseline)

# What's on prod but not on cloud (not yet deployed):
git log --oneline prod ^<last-deploy-sha>
```

`f20cffd9bc` would show up. Running `bash deploy/ecs/preflight.sh` would
report:

```
FAIL: f20cffd9bc feat(auth): rename OAuth button to "Continue with X" ...
       no e2e report references this SHA
       (touches: web_src/, templates/, options/locale/ — these need a report)
```

### Bringing it into compliance

To get `f20cffd9bc` past the gate:

1. **Restart Mac with the change**:
   ```bash
   git checkout v0.1-dev/hackforger
   bash scripts/restart-gitea.sh
   ```
2. **Claude opens `https://hackforger.inside.h2os.cloud`** and walks the
   sign-in flow:
   - landing → "Sign in / Register" link visible at top-right (zh-CN + en-US)
   - click → Logto consent → "Continue with X" button shows correct provider
   - successful login → lands on `/dashboard`
   - take screenshots at each step
3. **Claude writes the report**:
   `docs/tests/e2e/reports/2026-05-02-pr-137-oauth-copy.md`
   with frontmatter:
   ```yaml
   ---
   pr: 137
   commits: [f20cffd9bc]
   tested_against: https://hackforger.inside.h2os.cloud
   tested_at: 2026-05-02T11:00:00+0800
   e2e_owner: claude
   admin_signoff: null
   ---
   ```
4. **Commit the report** (no PR needed, goes straight to dev).
5. **Admin pulls dev locally**, fast-forwards `prod`, restarts Mac
   (`scripts/restart-gitea.sh`), opens the same URL, walks the same steps,
   then edits the report's `admin_signoff` field and commits.
6. **Now preflight passes for `f20cffd9bc`** and it's eligible for the next
   cloud deploy.

### Default — what to do if there's no report

If you discover a user-facing commit on `prod` without a report (we have one
right now: `f20cffd9bc` and `c1ffac9324`), the cloud deploy is **blocked**
until either:
- a backfill report is written and admin-signed, OR
- the commit is reverted off `prod`

There is no "deploy-now-test-later" override. The script has no `--force`
flag — if you need it, write the report first.

## What this document does not cover

- Hotfixes that bypass `dev` (e.g. a critical security patch directly on `prod`).
  For now, treat these as a pause-the-process exception: write the report
  retroactively within 24h.
- Schema migrations — they have their own rules in
  `docs/notes/pg-migration-pitfalls.md`.
- Rollback. See `deploy/ecs/cutover-runbook.md` for the rollback drill.
