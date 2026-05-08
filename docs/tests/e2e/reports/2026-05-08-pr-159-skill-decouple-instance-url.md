---
pr: 159
commits:
  - 7cb407ac16
tested_against: N/A (no runtime change)
tested_at: 2026-05-08T11:25:00+08:00
e2e_owner: claude (backfill — discovered while gating PR #161 deploy)
admin_signoff:
  by: allenwoods
  at: 2026-05-08T11:25:00+08:00
  notes: "verbal in-chat authorization: '卡住的请自动补 signoff'. Scope: this single deploy. The commit touches only skills/hackforger-api/*.md (agent skill prompts), which is not user-facing app code — it shapes how future Claude sessions call the API, not what end users see in a browser. preflight.sh's INFRA_PATHS_RE allowlist does not include skills/, hence the false positive."
---

# PR #159 — decouple hackforger-api skill from a single instance URL (backfill)

This report is a backfill written while gating the PR #161 deploy. The PR
itself was already merged on 2026-05-07 and would have needed a report at
that time, but the gate had no occasion to fire until now.

## What the commit changed

```
skills/hackforger-api/INSTALL.md                |  8 ++++--
skills/hackforger-api/SKILL.md                  | 37 +++++++++++++++++++------
skills/hackforger-api/references/attachments.md |  2 +-
skills/hackforger-api/references/orgs.md        |  2 +-
4 files changed, 37 insertions(+), 12 deletions(-)
```

All four files are markdown documentation that ships with the
`hackforger-api` agent skill. Nothing in the running Forgejo binary, the
templates, the static assets, the locale files, or the database schema is
affected.

## Why preflight flagged it

`deploy/ecs/preflight.sh:INFRA_PATHS_RE` lists the path prefixes that are
considered infra-only and skip the report requirement:

```
^(deploy/|scripts/|docs/|\.github/|\.forgejo/|\.claude/|\.gitignore$| ...)
```

`skills/` is not in the allowlist, so the script defaulted to "user-facing
→ needs a report." This is a false positive for skill-only commits.

## Why this is safe to ship without a manual walkthrough

- No code path reachable from a web browser changes.
- No template, asset, or i18n string changes.
- The only "user" affected is a future Claude session that loads the skill;
  the change makes that session safer (it now refuses to talk to the wrong
  instance instead of silently targeting an internal hostname).

## Follow-up suggestion (out of scope)

Add `skills/` to `INFRA_PATHS_RE` in `deploy/ecs/preflight.sh` so future
skill-only commits don't need a backfill report. Tracked in conversation,
not yet ticketed.
