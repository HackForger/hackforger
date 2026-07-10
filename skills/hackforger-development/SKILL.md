---
name: hackforger-development
description: Develop, review, test, document, or deploy HackForger while preserving Forgejo architecture and the public-repository boundary. Use for changes to HackForger Go code, templates, locales, custom assets, CI, deployment tooling, agent instructions, or repository documentation, especially when work may involve branded content, instance-specific configuration, production operations, or a separate private content repository.
---

# HackForger development

Keep HackForger reusable and business-neutral while following Forgejo's codebase conventions.

## Start safely

1. Verify the repository, branch, worktree, and authoritative task source before editing.
2. Use an isolated worktree for non-trivial changes. Preserve unrelated user changes.
3. Read the nearest repository instructions and the files that own the behavior.
4. On a trusted clone, run `bash scripts/install-public-boundary-hook.sh`; it
   refuses to replace an unrelated existing `core.hooksPath`. Re-run it after
   an audited boundary-guard update when the installed snapshot reports stale.
5. Run `bash scripts/check-public-repository-boundary.sh` before and after the change.

Run the boundary check before every push, not only before merge. A branch in a
public GitHub repository is already public when pushed; CI can reject it but
cannot undo that first disclosure. Use a private business repository for
business work and a fork for untrusted public contributions.

The installed, commit-versioned pre-push hook scans every outgoing commit
version, including files added and later deleted, plus outgoing ref metadata.
The standalone boundary command additionally scans both staged Git blobs and
differing worktree bytes. `--no-verify` is forbidden for ordinary work. A
dedicated guard-update branch may use it only after explicit security-owner
approval, with the local and ruleset bypass recorded in the security review.

## Enforce the repository boundary

Keep these in the public HackForger repository:

- reusable application code and tests;
- neutral templates, examples, fixtures, and documentation;
- generic deployment primitives driven by required configuration;
- CI and tooling that enforce this boundary.

Put these in the appropriate access-controlled business repository:

- branded landing pages, copy, media, campaign data, and business help content;
- real domains, hosts, addresses, accounts, filesystem layouts, and topology;
- instance-specific deployment configuration, runbooks, reports, and evidence;
- business-specific wrappers around the generic hydration or deployment tools.

Keep passwords, tokens, private keys, and other credentials in a secret manager or ignored local environment file, not in either Git repository.

When a request needs business-specific material:

1. Resolve the business repository from the task handoff,
   `HACKFORGER_PRIVATE_CONTENT_REPO`, or an operator-provided sibling-workspace
   path. Do not guess a repository name.
2. Confirm the target repository is access-controlled (`visibility=private`)
   before writing.
3. Add reusable capability to HackForger and concrete business content to the private repository.
4. If the private repository is unavailable or ambiguous, stop and ask; never place the content temporarily in HackForger.

Do not solve a boundary failure with an allowlist unless the matched text is genuinely reusable and neutral.

Boundary enforcement is self-protected. Changes to `CODEOWNERS`, any GitHub
workflow, the boundary wrapper, checker, tests, or marker policy require a
dedicated security-owner review and an explicit audited ruleset bypass; never
mix such a change with ordinary application or business-content work.

Do not treat the advisory push workflow as prevention: a branch or tag is
already public when that workflow starts. Maintainers must keep the exact
`Public repository boundary` status and CODEOWNER review required on maintained
branches. If repository writers are outside the trust boundary, use an
independent status-writing GitHub App or an organization-enforced required
workflow; repository workflows share the GitHub Actions App identity.

## Follow Forgejo architecture

- Preserve the dependency direction `routers -> services -> models -> modules`.
- Pass `context.Context` first and return typed errors.
- Use `db.WithTx` for multi-table state changes.
- Publish HackForger feed events for state changes.
- Follow existing Forgejo API, template, locale, and test patterns.
- Use Vue 3 for frontend components and justify new Go dependencies.
- Use `gh` for the GitHub-hosted repository; do not add Forgejo Actions for GitHub CI.

## Hydrate private content safely

- Treat the private repository as the source of truth; do not copy hydrated output into the public Git index.
- Require a versioned manifest and checksum verification before publishing.
- Stage content outside the public worktree or in an ignored directory.
- Back up the live target before replacement.
- Prefer a validated staging directory and atomic directory swap. Do not use `rsync --delete` for private content.
- Verify the deployed manifest, key files, and public behavior after publication.

## Verify before completion

Run the narrowest relevant tests first, then the repository-required gate bundle. At minimum:

```bash
bash scripts/check-public-repository-boundary.sh
git diff --check
```

For deployment tooling, also run syntax checks and a non-production fixture test. For user-facing changes, exercise the real web flow with browser evidence. Separate local proof from remote CI and production proof.
