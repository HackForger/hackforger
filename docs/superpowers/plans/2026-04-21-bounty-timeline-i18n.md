# Bounty Issue Timeline i18n + Username Display — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Spec:** `docs/superpowers/specs/2026-04-21-bounty-timeline-i18n-design.md`
**Issue:** [#34 HF-016](https://github.com/HackForger/hackforger/issues/34)

**Goal:** Issue Timeline comments posted by `postBountyIssueComment` show the applicant's username (not `user #<id>`) and render in the repo-owner's preferred language (zh-CN by default on this instance), while keeping the HackForger Feed / native Forgejo integration unchanged.

**Architecture:** The helper resolves the repo owner's `Language` field via `repo.MustOwner(ctx)`, builds a `translation.Locale`, and renders the message from a locale key + args. Four new keys (`hackforger.bounty.timeline.{applied,accepted,completed,cancelled}`) live under the existing `[hackforger]` sections of both locale ini files. A small `resolveUsername` helper dedupes the `GetUserByID` + "user #N" fallback used by the Apply and Accept call sites.

**Tech Stack:** Go (Forgejo), `forgejo.org/modules/translation`, `options/locale/*.ini`, agent-browser for E2E.

**Work location:** Use the existing worktree at `.worktrees/verify-34` on branch `verify/issue-34-bounty-feed` (already contains the spec). Commit the implementation here, then PR back to `v0.1-dev/hackforger`.

---

### File Structure

| File | Role |
|---|---|
| `services/hackforger/bounty.go` | Modify `postBountyIssueComment` signature; add `resolveUsername`; update 4 call sites (apply/accept/complete/cancel) |
| `options/locale/locale_en-US.ini` | Add 4 keys under `[hackforger]` |
| `options/locale/locale_zh-CN.ini` | Add 4 keys under `[hackforger]` |
| `docs/tests/e2e/tasks/issue-34-timeline-i18n-e2e.md` | Manual E2E prompt (per CLAUDE.md memory: every plan must end with an E2E prompt) |

No new files in Go. No schema migration. No frontend changes.

---

### Task 1: Add Chinese locale keys

**Files:**
- Modify: `options/locale/locale_zh-CN.ini` (insert under the existing `[hackforger]` section that begins at line 4007)

- [ ] **Step 1: Locate insertion anchor**

The `[hackforger]` section in `locale_zh-CN.ini` starts around line 4007. The existing key `bounty.error.internal` sits later in the same section (e.g., around line 4040). Insert the four new keys immediately **before** the first `bounty.error.*` key so all `bounty.*` keys stay grouped.

Run to confirm the exact line range before editing:

```bash
grep -n "^bounty\.error\.internal\|^\[hackforger\]" options/locale/locale_zh-CN.ini
```

Expected: two or more line numbers, with `[hackforger]` before `bounty.error.internal`.

- [ ] **Step 2: Insert the four keys**

Add this block immediately before the first `bounty.error.*` line under `[hackforger]`:

```ini
bounty.timeline.applied = 📋 **悬赏申请** — **%s** 申请承接
bounty.timeline.accepted = 🏷 **悬赏已接受** — **%s** 认领
bounty.timeline.completed = ✅ **悬赏已完成** — 交付已接受并发放
bounty.timeline.cancelled = ❌ **悬赏已取消**
```

**Do not prefix keys with `hackforger.`** — the section name supplies that prefix at lookup time. (Example: sibling key `bounty.error.internal` is looked up from Go as `hackforger.bounty.error.internal`.)

- [ ] **Step 3: Verify syntax**

```bash
grep -n "^bounty\.timeline\." options/locale/locale_zh-CN.ini
```

Expected: four lines printed with the exact text from Step 2.

- [ ] **Step 4: Commit**

```bash
git add options/locale/locale_zh-CN.ini
git commit -m "i18n(#34): add zh-CN bounty timeline comment keys"
```

---

### Task 2: Add English locale keys (mirror Task 1)

**Files:**
- Modify: `options/locale/locale_en-US.ini` (under `[hackforger]` beginning at line 3932)

- [ ] **Step 1: Locate insertion anchor**

```bash
grep -n "^bounty\.error\.issue_locked\|^\[hackforger\]" options/locale/locale_en-US.ini
```

Expected: two line numbers. Insert the four new keys immediately before `bounty.error.issue_locked` so `bounty.*` stays grouped.

- [ ] **Step 2: Insert the four keys**

```ini
bounty.timeline.applied = 📋 **Bounty Application** — **%s** applied
bounty.timeline.accepted = 🏷 **Bounty Accepted** — claimed by **%s**
bounty.timeline.completed = ✅ **Bounty Completed** — delivery accepted and bounty fulfilled
bounty.timeline.cancelled = ❌ **Bounty Cancelled**
```

Same prefix rule as Task 1 — no `hackforger.` inside the file.

- [ ] **Step 3: Verify parity with Chinese file**

```bash
diff <(grep -oE '^bounty\.timeline\.[a-z]+' options/locale/locale_zh-CN.ini | sort) \
     <(grep -oE '^bounty\.timeline\.[a-z]+' options/locale/locale_en-US.ini | sort)
```

Expected: no output (the four key names match exactly across files). Per CLAUDE.md rule, all user-facing i18n keys must exist in both locale files.

- [ ] **Step 4: Commit**

```bash
git add options/locale/locale_en-US.ini
git commit -m "i18n(#34): add en-US bounty timeline comment keys"
```

---

### Task 3: Change `postBountyIssueComment` signature + add `resolveUsername`

**Files:**
- Modify: `services/hackforger/bounty.go:21-43` (helper + imports)

- [ ] **Step 1: Add new imports**

Edit the import block at the top of `services/hackforger/bounty.go` (currently lines 6-19). Add these two imports alphabetically:

```go
"forgejo.org/modules/translation"
```

After the edit the import block should include (order is `forgejo.org/models/...`, then `forgejo.org/modules/...`, then `forgejo.org/services/...`):

```go
import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/translation"
	issue_service "forgejo.org/services/issue"
	notify_service "forgejo.org/services/notify"
)
```

`setting` is NOT needed — `translation.NewLocale` handles all fallback.

- [ ] **Step 2: Replace the helper + add `resolveUsername`**

Replace the existing block at `services/hackforger/bounty.go:21-43`:

```go
// postBountyIssueComment adds a timeline comment to the bounty's linked Issue
// when a bounty event occurs (created, applied, accepted, completed, etc.).
// Errors are logged but not propagated — issue comments are best-effort and
// should not block bounty operations.
//
// The message is rendered from a locale key using the repo owner's preferred
// language (falling back to the instance default via translation.NewLocale).
// This keeps each Issue thread in a single language — the one aligned with
// the project's steward.
func postBountyIssueComment(ctx context.Context, bounty *hackforger_model.Bounty, doerID int64, key string, args ...any) {
	doer, err := user_model.GetUserByID(ctx, doerID)
	if err != nil {
		log.Error("postBountyIssueComment: GetUserByID(%d): %v", doerID, err)
		return
	}
	issue, err := issues_model.GetIssueByID(ctx, bounty.IssueID)
	if err != nil {
		log.Error("postBountyIssueComment: GetIssueByID(%d): %v", bounty.IssueID, err)
		return
	}
	if err := issue.LoadRepo(ctx); err != nil {
		log.Error("postBountyIssueComment: LoadRepo for issue %d: %v", bounty.IssueID, err)
		return
	}
	owner := issue.Repo.MustOwner(ctx)
	locale := translation.NewLocale(owner.Language)
	message := locale.TrString(key, args...)
	if _, err := issue_service.CreateIssueComment(ctx, doer, issue.Repo, issue, message, nil); err != nil {
		log.Error("postBountyIssueComment: CreateIssueComment for issue %d: %v", bounty.IssueID, err)
	}
}

// resolveUsername returns the login name for userID, or "user #<id>" if the
// user cannot be loaded (deleted, demoted, etc.). Used by bounty Timeline
// messages so deletions never block or crash the best-effort comment path.
func resolveUsername(ctx context.Context, userID int64) string {
	u, err := user_model.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Sprintf("user #%d", userID)
	}
	return u.Name
}
```

- [ ] **Step 3: Verify the helper compiles in isolation**

At this point the four call sites still pass `message string` and will fail to compile. That is expected — we fix them in Task 4. For now, just check the imports and syntax of this block:

```bash
gofmt -l services/hackforger/bounty.go
```

Expected: no output (file is gofmt-clean).

- [ ] **Step 4: Do NOT commit yet** — the file won't build until Task 4.

---

### Task 4: Update the four call sites

**Files:**
- Modify: `services/hackforger/bounty.go:166` (Apply)
- Modify: `services/hackforger/bounty.go:253` (Accept)
- Modify: `services/hackforger/bounty.go:402` (Complete)
- Modify: `services/hackforger/bounty.go:626` (Cancel)

Note: line numbers shift after Task 3 because `resolveUsername` is added. Use Grep to re-locate each call site before editing.

- [ ] **Step 1: Find the current call sites**

```bash
grep -n "postBountyIssueComment(ctx, bounty" services/hackforger/bounty.go
```

Expected: four lines. Confirm each matches one of the four actions (apply / accept / complete / cancel).

- [ ] **Step 2: Update the Apply call site**

Replace the line matching `postBountyIssueComment(ctx, bounty, userID, fmt.Sprintf("📋 **Bounty Application** — user #%d applied", userID))` with:

```go
postBountyIssueComment(ctx, bounty, userID, "hackforger.bounty.timeline.applied", resolveUsername(ctx, userID))
```

- [ ] **Step 3: Update the Accept call site**

Replace the line matching `postBountyIssueComment(ctx, bounty, doerID, fmt.Sprintf("🏷 **Bounty Accepted** — claimed by user #%d", app.UserID))` with:

```go
postBountyIssueComment(ctx, bounty, doerID, "hackforger.bounty.timeline.accepted", resolveUsername(ctx, app.UserID))
```

- [ ] **Step 4: Update the Complete call site**

Replace the line matching `postBountyIssueComment(ctx, bounty, doerID, "✅ **Bounty Completed** — delivery accepted and bounty fulfilled.")` with:

```go
postBountyIssueComment(ctx, bounty, doerID, "hackforger.bounty.timeline.completed")
```

- [ ] **Step 5: Update the Cancel call site**

Replace the line matching `postBountyIssueComment(ctx, bounty, doerID, "❌ **Bounty Cancelled**")` with:

```go
postBountyIssueComment(ctx, bounty, doerID, "hackforger.bounty.timeline.cancelled")
```

- [ ] **Step 6: Build the backend**

From the worktree root:

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend
```

Expected: build succeeds, writes `gitea` binary. No compilation errors about `postBountyIssueComment` or missing types.

- [ ] **Step 7: Run the existing notifier test**

```bash
go test ./services/hackforger/... -run TestHackforger -v
```

Expected: all existing tests pass. No new tests are added by this plan — the i18n refactor has no new testable unit behavior beyond key resolution, which is exercised in E2E.

- [ ] **Step 8: Commit**

```bash
git add services/hackforger/bounty.go
git commit -m "feat(#34): render bounty timeline comments via locale + show username"
```

---

### Task 5: Manual E2E verification

**Files:**
- Create: `docs/tests/e2e/tasks/issue-34-timeline-i18n-e2e.md`
- Create (at run time): `docs/tests/e2e/reports/issue-34-timeline-i18n-report.md`

Per CLAUDE.md and user memory: every plan must end with a manual E2E prompt + report template under `docs/tests/e2e/`, executed via agent-browser against `http://localhost:3000` with screenshots at key checkpoints.

- [ ] **Step 1: Write the E2E prompt**

Create `docs/tests/e2e/tasks/issue-34-timeline-i18n-e2e.md` with the following content:

````markdown
# E2E: Issue #34 — Bounty Timeline i18n + Username

**Goal:** Confirm Timeline comments on a bounty's linked Issue (a) show the applicant's username (not `user #<id>`), (b) render in the repo owner's language, and (c) the HackForger Feed continues to show the parallel event.

**Prereqs:**
- Server built on this branch and running on `http://localhost:3000`.
- Before starting: `make frontend && TAGS="bindata sqlite sqlite_unlock_notify" make backend && ./gitea web` (needed because the current internal instance has `/assets/js/*` 404 — Vue panel will not mount otherwise).
- Test accounts: `hacker_eve` (publisher), `hacker_frank` (applicant) — password `admin1234`.

**Report:** write findings to `docs/tests/e2e/reports/issue-34-timeline-i18n-report.md` using the template at the bottom.

## Scenario A — zh-CN repo owner (default)

1. Login as `hacker_eve`. Visit `/user/settings` and confirm Language is `简体中文` (or unset, which falls back to instance default zh-CN). Screenshot the setting.
2. Open `e2e-test-2026/ai-innovation` — confirm `hacker_eve` is the owner (or the owning org's language is unset / zh-CN).
3. Create a fresh bounty issue OR use an existing Open one with no prior applications.
4. Logout. Login as `hacker_frank`.
5. On the bounty Issue, use the Vue panel "申请承接" button.
6. Reload the Issue page. **Expected:** new Timeline comment reading `📋 悬赏申请 — hacker_frank 申请承接` (with `hacker_frank` bolded). Screenshot.
7. Logout. Login as `hacker_eve`.
8. Accept hacker_frank's application via the Vue panel.
9. Reload. **Expected:** Timeline comment `🏷 悬赏已接受 — hacker_frank 认领`. Screenshot.
10. Cancel the bounty from the panel (if the flow permits) OR mark it completed.
11. Reload. **Expected:** either `❌ 悬赏已取消` or `✅ 悬赏已完成 — 交付已接受并发放`. Screenshot.

## Scenario B — en-US repo owner

1. As admin `hackforger`, change the bounty repo's owner Language to `English (US)` via `/user/settings` (for user-owned) or the org admin panel (for org-owned). Screenshot.
2. On a *new* bounty issue in that repo, repeat the Apply → Accept flow as in Scenario A steps 4-9.
3. **Expected:** Timeline comments render in English (`📋 Bounty Application — hacker_frank applied`, `🏷 Bounty Accepted — claimed by hacker_frank`). Screenshot.

## Scenario C — parallel HackForger Feed

1. After running Scenario A, visit `hacker_eve`'s dashboard (`/`).
2. **Expected:** Feed shows `hacker_eve 认领了 <bounty title> 的悬赏` — the parallel Feed path (`hackforger_action` table, `community_feeds.tmpl`) is unchanged.
3. Screenshot to confirm no regression.

## PASS / FAIL criteria

- **PASS** when Scenarios A and B both show the username (no `user #<id>`) in the correct language AND Scenario C shows the Feed entry unchanged.
- **FAIL** if any Timeline comment still shows `user #<id>`, renders in the wrong language for the repo owner's setting, or if the parallel Feed entry is missing.

## Report template

```markdown
# Report: Issue #34 Timeline i18n + Username

Date: <YYYY-MM-DD>
Branch: verify/issue-34-bounty-feed
Result: PASS | FAIL

## Scenario A (zh-CN)
- Apply comment: <actual text> — [screenshot](../screenshots/issue-34/a-apply.png)
- Accept comment: <actual text> — [screenshot](../screenshots/issue-34/a-accept.png)
- Close/Complete: <actual text> — [screenshot](../screenshots/issue-34/a-close.png)

## Scenario B (en-US)
- Apply comment: <actual text> — [screenshot](../screenshots/issue-34/b-apply.png)
- Accept comment: <actual text> — [screenshot](../screenshots/issue-34/b-accept.png)

## Scenario C (Feed regression)
- Feed entry: <actual text> — [screenshot](../screenshots/issue-34/c-feed.png)

## Defects found
<none / or list>
```
````

- [ ] **Step 2: Commit the E2E prompt**

```bash
git add docs/tests/e2e/tasks/issue-34-timeline-i18n-e2e.md
git commit -m "test(#34): add E2E prompt for bounty timeline i18n"
```

- [ ] **Step 3: Execute the E2E**

Follow the prompt in Step 1. Write the report to `docs/tests/e2e/reports/issue-34-timeline-i18n-report.md` and drop screenshots under `tests/screenshots/issue-34/`.

- [ ] **Step 4: Commit the E2E report + screenshots**

```bash
git add docs/tests/e2e/reports/issue-34-timeline-i18n-report.md tests/screenshots/issue-34/
git commit -m "test(#34): E2E report confirming timeline i18n"
```

*(Screenshots path `tests/screenshots/` is gitignored per `.gitignore`, so screenshots won't actually be committed. The report file alone suffices — link screenshots via GitHub Release upload if needed for the PR description.)*

---

### Task 6: Reply to Issue #34 + open PR

- [ ] **Step 1: Push branch**

```bash
git push -u origin verify/issue-34-bounty-feed
```

- [ ] **Step 2: Open PR against default branch**

```bash
gh pr create --title "fix(#34): bounty timeline i18n + username display" --body "$(cat <<'EOF'
## Summary

- `postBountyIssueComment` now renders from locale keys instead of hardcoded English markdown
- Timeline comments use the repo owner's preferred language (falls back to instance default, zh-CN)
- Applicant is shown by username (`hacker_frank`) instead of `user #<id>`
- HackForger Feed path (`hackforger_action` table) is unchanged — already i18n-correct

## Test plan

- [ ] Scenario A: zh-CN owner renders Chinese Timeline comments with username
- [ ] Scenario B: en-US owner renders English Timeline comments with username
- [ ] Scenario C: HackForger Feed entry still appears (no regression)
- [ ] `make backend` succeeds
- [ ] Locale-key parity check (`diff` on bounty.timeline.* keys) passes between zh-CN and en-US

Spec: `docs/superpowers/specs/2026-04-21-bounty-timeline-i18n-design.md`
E2E: `docs/tests/e2e/tasks/issue-34-timeline-i18n-e2e.md`

Closes #34

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

- [ ] **Step 3: Comment on Issue #34**

```bash
gh issue comment 34 --body "已提交修复 PR — 在 Timeline 评论里显示申请人用户名并按 repo owner 语言渲染。等 PR 合并后请测试员再走一遍完整流程确认。"
```

Do NOT close the issue yet — the issue closes automatically when the PR merges (`Closes #34` in PR body).

---

## Self-Review Checklist

- [x] Every spec section has a task: problem → Tasks 1-4; language rule → Task 3 Step 2 (`NewLocale(owner.Language)`); helper signature change → Task 3 Step 2; locale keys → Tasks 1-2; call sites → Task 4; testing → Task 5; risk/rollback covered by commit granularity (4 isolated commits).
- [x] No placeholders.
- [x] Type/name consistency: helper is `postBountyIssueComment` in every task; `resolveUsername` is used in Tasks 3-4 with identical signature.
- [x] File paths absolute-to-repo and line-referenced where relevant.
- [x] E2E prompt written (per CLAUDE.md memory feedback).
- [x] Test: locale-key parity check between zh-CN and en-US files (CLAUDE.md rule).
