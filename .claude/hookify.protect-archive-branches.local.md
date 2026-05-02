---
name: protect-archive-branches
enabled: true
event: bash
action: block
conditions:
  - field: command
    operator: regex_match
    pattern: git\s+(rebase|branch\s+-D|branch\s+--delete\s+(-f|--force)|push\s+(.*\s+)?(-f|--force)|reset\s+--hard).*\b(dev/[\w.-]*(backup|snapshot)[\w.-]*|test-data-backup-2026-04-30)\b
  - field: command
    operator: not_contains
    pattern: CONFIRM_DESTROY=1
---

🛑 **Refusing to run destructive git op on a protected archive branch.**

The branch you targeted matches the archive pattern (`dev/*-backup-*`, `dev/*-snapshot-*`, or `test-data-backup-2026-04-30`). These branches are intentional historical snapshots — rebasing, force-pushing, or hard-resetting them destroys irrecoverable state.

**Before overriding, run the 3-question check** (from `~/.claude/projects/-Users-h2oslabs-Workspace-hackforger/memory/feedback_branch_rebase_check.md`):

1. Is the branch pushed to origin? (if not, no one else has a copy)
2. Is the branch / its files referenced by paths on the default branch? (e.g. `git grep <branch-or-path> v0.1-dev/hackforger`)
3. Are the unique commits "snapshot" / "backup" / "preserve" themed?

If any answer is YES → leave it alone, do not destroy.

If you've checked all three and the user has explicitly confirmed they want to destroy this branch (rare):

```bash
CONFIRM_DESTROY=1 <your original command>
```

The `CONFIRM_DESTROY=1` env-var prefix is the explicit override; this hook will not block it.
