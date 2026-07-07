---
pr: 191
commits:
  - 95526fddca09b089cb6f4ebbbc95edf5513e1a7e
  - 4e1467554648aa0d49cdb7791edb5db4973b780f
  - 8b3867616534a271671db77fccb8f4cde68d5939
  - 4fcb8a5a2ef8f7859192a87b473d41da85bb473a
  - d9df3350e934ea0801db858ac40787bada213bbd
tested_against: static landing files + https://www.synnovator.com post-deploy smoke
tested_at: 2026-07-07T16:31:02+0800
e2e_owner: codex
admin_signoff:
  by: h2oslabs-operator
  at: 2026-07-07T16:31:02+0800
  notes: "In-chat production authorization: requested PR #191 be read and the new landing page deployed. Operator explicitly allowed skipping local test environment deployment because the change only touches static landing pages. This is the same landing-only fast path used by recent landing deploys; production deploy still runs the blessed deploy/ecs/redeploy.sh smoke checks."
---

# PR #191 - S3 and Xuedoushan landing update - smoke report

## Scope

PR #191 updates static landing assets only:

- `custom/public/assets/landing/index.html`
- `custom/public/assets/landing/s1-ranking.html`
- `custom/public/assets/landing/s2-ranking.html`
- `custom/public/assets/landing/assets/images/轮播图3.webp`
- `custom/public/assets/landing/assets/images/轮播图4.webp`
- screenshot evidence under `docs/tests/e2e/reports/assets/`

No Go code, migrations, locale bindata, `web_src/`, dependency files, or server
configuration changed.

## Pre-deploy static verification

Performed on the merged `v0.1-dev/hackforger` tip
`d9df3350e934ea0801db858ac40787bada213bbd`.

- Landing carousel now has S1, S2, S3, and Xuedoushan slides.
- S2 `立即报名` scrolls to `#opc-stage-02`; the target element exists.
- S3 `立即报名` scrolls to `#opc-stage-03`; the target element exists.
- Xuedoushan `立即报名` links to
  `https://www.synnovator.com/xuedoushan/` with `target="_blank"` and
  `rel="noopener noreferrer"`.
- New banner images exist at the runtime-served custom public path:
  `assets/images/轮播图3.webp` and `assets/images/轮播图4.webp`.
- S1/S2 ranking modal detail rendering no longer emits the removed fields
  `对外展示项目名`, `平台仓库链接`, or `演示链接`; it keeps `团队/作者`,
  `系统项目名`, and `项目简介`.

Local Mac test-environment redeploy was intentionally skipped per operator
authorization because this is a static landing-page change.

## Production verification gate

Deployment must use `bash deploy/ecs/redeploy.sh`, which:

- runs `deploy/ecs/preflight.sh`;
- rebuilds the binary;
- syncs `custom/templates/` and `custom/public/` to ECS;
- restarts production;
- checks `/api/v1/version`, the navbar locale smoke, and public landing HTTPS
  status.

Post-deploy command output is the final production smoke evidence for this
landing-only fast path.
