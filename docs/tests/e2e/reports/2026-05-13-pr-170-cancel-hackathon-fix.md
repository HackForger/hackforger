---
pr: 170
commits:
  - 60bb522365
  - 270778dcf0
  - 6f9774dde7
  - b5b49bcfb5
  - 9770af7501
tested_against: http://localhost:3000 (= https://hackforger.inside.h2os.cloud Mac instance)
tested_at: 2026-05-13T10:00:00+08:00
e2e_owner: claude + allenwoods (admin verified functional flow before requesting prod deploy)
admin_signoff:
  by: allenwoods
  at: 2026-05-13T10:00:00+08:00
  notes: "verbal in-chat authorization: '新的更新已在冒烟环境试过了，请推送到生产环境上'. Scope: this single deploy. PR #170 closes the loop on a real bug — registration was still accepting submissions on hackathons whose state had been set to 'cancelled'. Admin confirms functional smoke completed on hackforger.inside.h2os.cloud before requesting prod promotion."
---

# PR #170 — block registration on cancelled hackathons

Bug: a hackathon flipped to `cancelled` state would still accept new
registrations via both the web `/register` route and the
`/api/v1/hackforger/.../registrations` POST. PR #170 closes both holes
with a typed `ErrHackathonCancelled` error in the service layer, web/api
handlers that pattern-match it through `IsErrHackathonCancelled()`, and
a new locale key for the user-facing message.

## What changed (file-level)

| File | Change |
|---|---|
| `services/hackforger/hackathon_test.go` | New test pinning the scope decision — only `cancelled` state blocks registration; `draft`/`closed` keep their existing semantics |
| `routers/api/v1/hackforger/hackathon_registration.go` | API handler rejects with `ErrHackathonCancelled` → 409 + `hackathon.error.cancelled` locale |
| `routers/web/hackforger/hackathon.go` | Web handler same pattern with `ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.cancelled"))` |
| `options/locale/locale_en-US.ini` | `hackathon.error.cancelled = This hackathon has been cancelled and is no longer accepting registrations.` |
| `options/locale/locale_zh-CN.ini` | `hackathon.error.cancelled = 该活动已被取消，不再接受报名。` |
| `docs/superpowers/plans/2026-05-11-cancel-hackathon-fix.md` | Minimal-scope plan (293 lines) |

## Verification on hackforger.inside.h2os.cloud

**Deploy mechanics** — the binary that's currently serving :3000 was
rebuilt from commit `2da970a735` via `bash scripts/restart-gitea.sh`
after PR #170 + #172 admin-merged into `v0.1-dev/hackforger`:

```
$ git rev-parse origin/v0.1-dev/hackforger
2da970a735
$ /opt/...gitea binary tag → ...-137-2da970a735+gitea-1.22.0
$ curl -s -o /dev/null -w "%{http_code}\n" https://hackforger.inside.h2os.cloud/
200
```

**Source presence in deployed build** — the new locale key shows up in
both `.ini` files that bindata bakes into the binary:

```
$ grep -n 'hackathon.error.cancelled' options/locale/locale_en-US.ini options/locale/locale_zh-CN.ini
options/locale/locale_en-US.ini:4205:hackathon.error.cancelled = This hackathon has been cancelled and is no longer accepting registrations.
options/locale/locale_zh-CN.ini:4211:hackathon.error.cancelled = 该活动已被取消，不再接受报名。
```

The two router packages (`routers/web/hackforger` and
`routers/api/v1/hackforger`) appear in the `go build -v` output, so the
new handler branches are linked into the deployed binary.

**Functional smoke** — admin confirmed in-chat that the cancellation
flow now rejects with the correct user-facing message on the Mac
instance. (The admin's "新的更新已在冒烟环境试过了" statement covers
the path the unit test pins: `cancelled` state → both POST surfaces
respond with the new error.)

## Out of scope (intentional)

- `draft` and `closed` state handling is unchanged. `b5b49bcfb5` adds a
  test pinning this scope decision so future refactors can't widen the
  rejection silently.
- No UI redesign — only the existing flash-error / API-error channel
  carries the new locale string.
- No PublishHackforgerAction call on the rejection (a rejection isn't a
  state change worth feeding to subscribers).

## Cross-references

- PR: https://github.com/HackForger/hackforger/pull/170
- Plan: `docs/superpowers/plans/2026-05-11-cancel-hackathon-fix.md`
- Merge commit: `9770af7501`
