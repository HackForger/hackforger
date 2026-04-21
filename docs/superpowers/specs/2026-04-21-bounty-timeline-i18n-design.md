# Bounty Issue Timeline — i18n + Username Display

**Issue:** [#34 HF-016: Bounty 状态变更写入 Issue Feed](https://github.com/HackForger/hackforger/issues/34)
**Author:** Allen Woods
**Date:** 2026-04-21
**Status:** Draft

## Problem

`postBountyIssueComment` (`services/hackforger/bounty.go:25-43`) currently writes **English-only hardcoded markdown** to the linked Issue's comment thread, and references users by numeric ID instead of username:

```
📋 **Bounty Application** — user #4 applied
🏷 **Bounty Accepted** — claimed by user #4
✅ **Bounty Completed** — delivery accepted and bounty fulfilled.
❌ **Bounty Cancelled**
```

Two defects:

1. **No i18n.** The instance is zh-CN by default; the English strings are inconsistent with the rest of the product UI.
2. **`user #N` is unusable.** Readers of the Issue Timeline cannot tell who applied or who claimed without clicking through.

## Constraints

- **Architectural:** Cannot add new `CommentType` enum values. That would require modifying `models/issues/comment.go` and Timeline render templates, which fall outside the 11 HackForger injection points. We must continue using `CommentTypeComment` (free-form markdown).
- **Consequence:** Timeline comments are **stored as rendered markdown**, not re-localized per-reader. We pick a language at write time; all future readers see that language regardless of their own locale preference.
- **This is not a regression from native Forgejo** — all free-form Issue comments have this property. Forgejo's structured Timeline events (labels, close, rename) are the only localized-per-reader items, and that mechanism is closed to extension.

## Design

### Language selection rule

At write time, select language in this order:

1. **Repo owner's user-preferred language** (`repo.MustOwner(ctx).Language`, a `VARCHAR(5)` like `"zh-CN"` or `"en-US"`).
2. **Instance default** (`setting.Langs[0]`) if the owner has no preference set.
3. **Hard fallback:** `"zh-CN"` if `setting.Langs` is empty (defensive — should not occur).

Rationale: the repo owner is the de-facto steward of the Issue thread. Their language aligns with the project's internal sphere (team members, contributors active in that repo). Using the *actor's* (doer's) language would make the same Issue thread multilingual over time — worse UX than single-language-per-repo.

For Organization-owned repos, `MustOwner` returns the Org's User record, which also has a `Language` field.

### Helper signature change

`postBountyIssueComment` moves from accepting a pre-rendered `message string` to accepting a **locale key + args**:

```go
// Before
func postBountyIssueComment(ctx context.Context, bounty *hackforger_model.Bounty, doerID int64, message string)

// After
func postBountyIssueComment(ctx context.Context, bounty *hackforger_model.Bounty, doerID int64, key string, args ...any)
```

Inside the helper:

1. Load `doer` (existing).
2. Load `issue` + `issue.Repo` (existing, via `LoadRepo`).
3. Resolve language: `lang := issue.Repo.MustOwner(ctx).Language`, fallback chain above.
4. Build locale: `locale := translation.NewLocale(lang)`.
5. Render: `message := locale.TrString(key, args...)`.
6. `issue_service.CreateIssueComment(ctx, doer, issue.Repo, issue, message, nil)` (existing).

All error paths remain best-effort (log + return, never propagate — matches current behavior).

### Locale keys

Add 4 keys under the `[hackforger]` section of both `locale_en-US.ini` and `locale_zh-CN.ini`:

**`options/locale/locale_zh-CN.ini`:**
```ini
bounty.timeline.applied = 📋 **悬赏申请** — @%s 申请承接
bounty.timeline.accepted = 🏷 **悬赏已接受** — @%s 认领
bounty.timeline.completed = ✅ **悬赏已完成** — 交付已接受并发放
bounty.timeline.cancelled = ❌ **悬赏已取消**
```

**`options/locale/locale_en-US.ini`:**
```ini
bounty.timeline.applied = 📋 **Bounty Application** — @%s applied
bounty.timeline.accepted = 🏷 **Bounty Accepted** — claimed by @%s
bounty.timeline.completed = ✅ **Bounty Completed** — delivery accepted and bounty fulfilled
bounty.timeline.cancelled = ❌ **Bounty Cancelled**
```

**Why `@%s` instead of `**%s**` or plain `%s`:** Forgejo parses `@username` in comment markdown as a user mention, which auto-generates a notification to that user. For the Accept case this is exactly what we want (applicant learns they were accepted). For Apply, the mentioned user is the comment author — Forgejo suppresses self-mention notifications, so no noise. Consistent format across the four keys.

### Call site changes

**Apply** (`bounty.go:166`):

```go
applicant, err := user_model.GetUserByID(ctx, userID)
applicantName := fmt.Sprintf("user-%d", userID) // fallback for deleted users
if err == nil {
    applicantName = applicant.Name
}
postBountyIssueComment(ctx, bounty, userID, "hackforger.bounty.timeline.applied", applicantName)
```

**Accept** (`bounty.go:253`):

```go
applicant, err := user_model.GetUserByID(ctx, app.UserID)
applicantName := fmt.Sprintf("user-%d", app.UserID)
if err == nil {
    applicantName = applicant.Name
}
postBountyIssueComment(ctx, bounty, doerID, "hackforger.bounty.timeline.accepted", applicantName)
```

**Complete** (`bounty.go:402`):

```go
postBountyIssueComment(ctx, bounty, doerID, "hackforger.bounty.timeline.completed")
```

**Cancel** (`bounty.go:626`):

```go
postBountyIssueComment(ctx, bounty, doerID, "hackforger.bounty.timeline.cancelled")
```

Applicant-lookup errors fall back to `user-{id}` (never crash or skip the comment). This matches the existing best-effort contract.

## Out of Scope

- **Per-reader localization of Timeline comments.** Requires new `CommentType`; violates injection point rules.
- **Re-rendering past comments in new language.** Existing comments remain in the language they were written in.
- **Complete/Cancel showing actor name.** The doer is already shown in Forgejo's comment header (avatar + username). Adding it inside the message body would be redundant.
- **Static asset 404 on the live instance.** Separate deployment issue; tracked as a reminder to run `make frontend && make backend` before next restart.
- **Fixing any of the other HackForger Feed (`hackforger_action`) rendering.** That side is already i18n-correct via `community_feeds.tmpl`.

## Testing Strategy

1. **Unit-adjacent:** Existing `services/hackforger/notifier_test.go` covers feed publishing but not Timeline comments. No new unit test needed — the code path is straightforward locale-rendering.
2. **E2E manual (mandatory per memory):**
   - Set `hackforger` user Language to `zh-CN` (via `/user/settings`), verify bounty events produce Chinese comments.
   - Set to `en-US`, verify English.
   - Clear language (empty), verify fallback to `setting.Langs[0]`.
   - Confirm `@username` renders as a clickable mention link in Timeline.
3. **Regression:** Apply → Accept → Cancel on a fresh bounty; all 3 Timeline comments present with correct username + language.

## Risk & Rollback

- **Locale key typo** → `TrString` returns the key itself. Visible as `hackforger.bounty.timeline.applied` in Timeline. Safe, not crash-worthy.
- **Deleted user at read time** → username is baked into the stored comment at write time; deletion later doesn't corrupt the comment. The `@mention` just stops resolving to a link, plain text remains.
- **Rollback:** revert the 2 Go files + 2 locale files. Past Timeline comments persist in whatever language/format they were written; no data migration needed.

## Files Touched

| File | Change |
|---|---|
| `services/hackforger/bounty.go` | Helper signature + 4 call sites |
| `options/locale/locale_en-US.ini` | +4 keys under `[hackforger]` |
| `options/locale/locale_zh-CN.ini` | +4 keys under `[hackforger]` |
