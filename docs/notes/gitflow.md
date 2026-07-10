# HackForger contribution and promotion flow

HackForger uses reviewed feature branches, an integration branch, and a production branch:

```text
feature/* -> v0.1-dev/hackforger -> prod
```

## Public repository gates

1. Develop in an isolated worktree.
2. Run focused tests and `bash scripts/check-public-repository-boundary.sh`.
3. Open a pull request against `v0.1-dev/hackforger`.
4. Require the public repository boundary check and maintainer review.
5. Promote an already-reviewed commit to `prod`; do not rebuild an unrelated worktree HEAD.

## Instance-specific release work

Production targets, credentials, smoke reports, runtime screenshots, deployment markers, and rollback instructions belong to the private business repository for that instance. The public repository must not contain real environment facts.

Deployment tooling must consume explicit private configuration, back up affected runtime content, verify manifests, and record runtime evidence. Private content publishing is independent from application binary promotion.
