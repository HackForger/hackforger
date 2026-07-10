# Security policy

Report vulnerabilities through [GitHub private vulnerability reporting](https://github.com/HackForger/hackforger/security/advisories/new), not a public issue.

Do not include live credentials, private keys, real production accounts,
internal addresses, customer data, or unredacted runtime evidence in a report.
If a credential may already have been disclosed, revoke or rotate it first;
removing a file or rewriting Git history does not invalidate the credential.

HackForger is a public, business-neutral project. Instance-specific security
facts and remediation evidence belong in the corresponding access-controlled
business repository or approved incident-management system.

Boundary guard files (every supported `CODEOWNERS` location, `.github/workflows/**`,
`.githooks/**`, the canonical development skill, boundary wrappers, checker,
tests, and marker policy) are immutable in an ordinary pull request. A necessary
guard update requires a dedicated security-owner review and an explicit,
audited ruleset bypass; unrelated source or content changes must not share that
bypass.

Repository rules must require the exact `Public repository boundary` commit
status, an up-to-date branch, and CODEOWNER review on both maintained release
branches. The branch/tag push workflow is only a post-disclosure audit; only the
installed pre-push hook and operator discipline can stop a first public push.
GitHub Actions workflows also share one App identity, so an organization that
does not fully trust repository writers must bind the required status to an
independent GitHub App or organization-enforced required workflow.

Dependabot version and security updates are intentionally disabled for this
repository. Dependabot-triggered `pull_request_target` workflows receive a
read-only token and cannot publish the required candidate commit status. Before
enabling either feature, deploy and verify a trusted GitHub App or equivalent
second-stage status writer, then update this guard through the dedicated
security-owner bypass process.
