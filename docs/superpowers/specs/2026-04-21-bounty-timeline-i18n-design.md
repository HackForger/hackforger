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

At write time, read `issue.Repo.MustOwner(ctx).Language` and pass it to `translation.NewLocale(lang)`. `NewLocale` (`modules/translation/translation.go:178-206`) handles all fallbacks internally:

- Empty / unknown lang → falls back to `setting.Langs[0]` (instance default).
- `setting.Langs` ever empty → `NewLocale` still returns a usable locale labeled "unknown"; `TrString` returns the key itself.

**The helper does NOT need its own fallback chain.** Pass `owner.Language` directly to `NewLocale`.

Rationale: the repo owner is the de-facto steward of the Issue thread. Their language aligns with the project's internal sphere. Using the *actor's* (doer's) language would make the same Issue thread multilingual over time — worse UX than single-language-per-repo.

For Organization-owned repos, `MustOwner` returns the Org's User record, which also has a `Language` field. **In practice, most Orgs do not set a language preference**, so org-owned bounties will typically render in `setting.Langs[0]` (zh-CN on our instance).

On `MustOwner` internal error, Forgejo returns a synthetic `&User{Name: "error", Language: ""}` (`models/repo/repo.go:484-494`) — the empty-Language path handles this.

### Helper signature change

`postBountyIssueComment` moves from accepting a pre-rendered `message string` to accepting a **locale key + args**:

```go
// Before
func postBountyIssueComment(ctx context.Context, bounty *hackforger_model.Bounty, doerID int64, message string)

// After
func postBountyIssueComment(ctx context.Context, bounty *hackforger_model.Bounty, doerID int64, key string, args ...any)
```

`TrString` signature verified: `TrString(trKey string, trArgs ...any) string` (`modules/translation/i18n/localestore.go:201`).

Inside the helper:

1. Load `doer` via `user_model.GetUserByID` (existing).
2. Load `issue` via `issues_model.GetIssueByID` (existing).
3. Load `issue.Repo` via `issue.LoadRepo(ctx)` (existing).
4. `lang := issue.Repo.MustOwner(ctx).Language`.
5. `locale := translation.NewLocale(lang)`.
6. `message := locale.TrString(key, args...)`.
7. `issue_service.CreateIssueComment(ctx, doer, issue.Repo, issue, message, nil)` (existing).

All error paths remain best-effort (log + return, never propagate — matches current behavior).

### Locale keys

Add the following four lines under the existing `[hackforger]` section (line 3932 in `locale_en-US.ini`, parallel location in `locale_zh-CN.ini`). **Do NOT prefix keys with `hackforger.` inside the file** — Forgejo composes the lookup key as `<section>.<key>` at load time. Verify by reference: existing key `bounty.error.issue_required` (file line 3957) is looked up as `hackforger.bounty.error.issue_required` from Go.

**`options/locale/locale_zh-CN.ini` (under `[hackforger]`):**
```ini
bounty.timeline.applied = 📋 **悬赏申请** — **%s** 申请承接
bounty.timeline.accepted = 🏷 **悬赏已接受** — **%s** 认领
bounty.timeline.completed = ✅ **悬赏已完成** — 交付已接受并发放
bounty.timeline.cancelled = ❌ **悬赏已取消**
```

**`options/locale/locale_en-US.ini` (under `[hackforger]`):**
```ini
bounty.timeline.applied = 📋 **Bounty Application** — **%s** applied
bounty.timeline.accepted = 🏷 **Bounty Accepted** — claimed by **%s**
bounty.timeline.completed = ✅ **Bounty Completed** — delivery accepted and bounty fulfilled
bounty.timeline.cancelled = ❌ **Bounty Cancelled**
```

Go call sites reference the keys with the section prefix:
- `locale.TrString("hackforger.bounty.timeline.applied", applicantName)`
- `locale.TrString("hackforger.bounty.timeline.accepted", applicantName)`
- `locale.TrString("hackforger.bounty.timeline.completed")`
- `locale.TrString("hackforger.bounty.timeline.cancelled")`

**Why `**%s**` (bold) and NOT `@%s` (mention):** Using `@username` would trigger Forgejo's user-mention parser, generating a separate notification to that user. HackForger already has its own notification path (`notify_service.HackforgerEntityStatusChanged` with audience routing) for these events, so a mention-notification would be duplicate and was not part of Issue #34's scope. Bold formatting gives visual emphasis without side effects. Usernames in Forgejo are restricted to `[a-zA-Z0-9._-]` so direct interpolation into markdown is safe from injection.

### Call site changes

**Helper for resolving display name** (add near `postBountyIssueComment`):

```go
// resolveUsername loads a user by ID and returns their login name; on error
// falls back to "user #<id>" to match the prior Timeline convention. Both
// callers inside bounty.go need this, and resolution is best-effort.
func resolveUsername(ctx context.Context, userID int64) string {
    u, err := user_model.GetUserByID(ctx, userID)
    if err != nil {
        return fmt.Sprintf("user #%d", userID)
    }
    return u.Name
}
```

**Apply** (`bounty.go:166`):

```go
postBountyIssueComment(ctx, bounty, userID, "hackforger.bounty.timeline.applied", resolveUsername(ctx, userID))
```

**Accept** (`bounty.go:253`):

```go
postBountyIssueComment(ctx, bounty, doerID, "hackforger.bounty.timeline.accepted", resolveUsername(ctx, app.UserID))
```

**Complete** (`bounty.go:402`):

```go
postBountyIssueComment(ctx, bounty, doerID, "hackforger.bounty.timeline.completed")
```

**Cancel** (`bounty.go:626`):

```go
postBountyIssueComment(ctx, bounty, doerID, "hackforger.bounty.timeline.cancelled")
```

Fallback format `user #<id>` matches the old output for rare deleted-user cases. Never crashes or skips the comment (best-effort contract).

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
