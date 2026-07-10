# HackForger Development Guide

HackForger is a business-neutral Forgejo fork that adds Hackathon, Bounty, Grant, Credits, Feed, and related collaboration modules.

Before changing code, documentation, CI, deployment tooling, templates, locales, or custom assets, read and follow [`skills/hackforger-development/SKILL.md`](skills/hackforger-development/SKILL.md).

## Public repository boundary

- Keep reusable platform code, neutral examples, generic deployment primitives, and their tests in this repository.
- Put branded pages, campaign or customer content, instance configuration, real environment facts, production runbooks, and runtime evidence in the corresponding private business repository.
- Keep credentials in a secret manager or ignored local environment file; never commit them to public or private Git.
- Do not use `git add -f` to bypass ignored private-content paths.
- Run `bash scripts/check-public-repository-boundary.sh` before committing and
  again before pushing. CI cannot undo disclosure on an already-public branch.

## Architecture rules

- Preserve the dependency direction `routers -> services -> models -> modules`; never import upward.
- Put CRUD data access in `models/` and business state transitions in `services/`.
- Pass `context.Context` first, return typed errors, and use `db.WithTx` for multi-table operations.
- Publish HackForger feed events for every user-visible state change.
- Keep HackForger additions in the existing `*/hackforger/` extension points where possible.
- Follow Forgejo API, template, locale, and test patterns.
- Use Go template SSR with Vue 3 enhancement; browser actions use session-authenticated web routes rather than token API routes.

## Error handling and i18n

- Never surface raw `err.Error()` to users without mapping a typed error.
- Add every user-facing string to both `locale_en-US.ini` and `locale_zh-CN.ini` under the HackForger section.
- Do not add traditional CSRF token fields; Forgejo uses Go cross-origin protection.

## Workflow and verification

- Use an isolated worktree for non-trivial changes.
- Feature branches merge into `v0.1-dev/hackforger`; production promotion uses the repository's reviewed release process.
- Store instance-specific release instructions and evidence in the private business repository.
- Run focused tests first, followed by the relevant repository gate bundle.
- User-facing changes require real browser verification; deployment changes require fixture tests, backup proof, and post-deploy runtime evidence.

Common commands:

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend
make frontend
go test ./models/hackforger/... -v
go test ./services/hackforger/... -v
bash scripts/check-public-repository-boundary.sh
git diff --check
```

Use `gh` for the GitHub-hosted repository. Do not add Forgejo Actions for GitHub CI.
