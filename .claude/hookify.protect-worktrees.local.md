---
name: protect-worktrees
enabled: true
event: bash
action: block
conditions:
  - field: command
    operator: regex_match
    pattern: git\s+worktree\s+remove\s+(.*\s+)?\.claude/worktrees/
  - field: command
    operator: not_contains
    pattern: CONFIRM_DESTROY=1
---

🛑 **Refusing to remove a worktree under `.claude/worktrees/`.**

The worktrees in `.claude/worktrees/` are load-bearing:

- **`.claude/worktrees/dev/`** — backs the `dev/test-data-backup-2026-04-30` archive branch. The default branch's `docs/notes/pg-migration-pitfalls.md` references files at `.claude/worktrees/dev/data-snapshot/RESTORE.md`. Removing the worktree breaks that reference.
- Other worktrees may back active dev branches you didn't realize were in use.

**Before overriding**, verify:

1. The worktree's branch isn't referenced from the default branch (`git grep <worktree-path> v0.1-dev/hackforger`)
2. No file in the worktree is unique (compare against the branch HEAD)
3. The user has explicitly asked to remove it (not just "clean up")

If all three pass, override with:

```bash
CONFIRM_DESTROY=1 git worktree remove .claude/worktrees/<name>
```
