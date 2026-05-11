# Cancel Hackathon: Block New Registrations (Minimal Fix)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make cancelled hackathons reject new registrations through both API and Web entry points, and stop the detail page from showing a misleading "Register" button. This is the minimum viable fix for [#167](https://github.com/HackForger/hackforger/issues/167) — full delete and broader permission hardening are out of scope (see [#169](https://github.com/HackForger/hackforger/issues/169) for the systemic API permission gap).

**Architecture:** Three handler-level status checks against `HackathonStatusCancelled`. No model, service, fixture, or schema changes. Status check sits next to existing `IsPublished` and `AllowsAction` guards — same layer, same error-flash pattern.

**Tech Stack:** Go (Forgejo handlers), testify, Forgejo i18n (`locale_*.ini`)

---

## Out of Scope (Explicit Non-Goals)

- ❌ Setting `is_published=false` on cancel — keeps `PublishHackathon`'s "already published" guard intact, manage page UI stays consistent
- ❌ Deleting current/future phases on cancel — already-registered users can still submit, judges can still score (by design)
- ❌ Refunds, notifications, or activity hiding — operational tasks, low-volume edge case
- ❌ Owner/Admin permission check on `DELETE /hackathons/{id}` — moved to [#169](https://github.com/HackForger/hackforger/issues/169)

---

## File Map

| File | Change |
|---|---|
| `routers/api/v1/hackforger/hackathon_registration.go` | Add cancelled status check in `Register` handler |
| `routers/web/hackforger/hackathon.go` | Add cancelled status check in `RegisterPost`; force `CanRegister=false` in `ViewHackathon` for cancelled |
| `options/locale/locale_en-US.ini` | Add `hackforger.hackathon.error.cancelled` |
| `options/locale/locale_zh-CN.ini` | Add `hackforger.hackathon.error.cancelled` |
| `services/hackforger/hackathon_test.go` | Test: cancelled hackathon rejects registration via service-side check |

---

## Task 1: Add i18n error key

**Files:**
- Modify: `options/locale/locale_en-US.ini`
- Modify: `options/locale/locale_zh-CN.ini`

- [ ] **Step 1: Find the `[hackforger]` section in en locale**

```bash
cd /Users/daiming/workspace/hackforger && \
  grep -n "hackforger.hackathon.error" options/locale/locale_en-US.ini | head -5
```

Note the line numbers — you'll add the new key alongside existing `hackforger.hackathon.error.*` keys.

- [ ] **Step 2: Add the en key**

In `options/locale/locale_en-US.ini`, locate any `hackforger.hackathon.error.*` line (e.g. `hackforger.hackathon.error.invalid_phase`) and add this key directly after it:

```ini
hackforger.hackathon.error.cancelled = This hackathon has been cancelled and is no longer accepting registrations.
```

- [ ] **Step 3: Add the zh key**

In `options/locale/locale_zh-CN.ini`, find the matching error key block and add:

```ini
hackforger.hackathon.error.cancelled = 该活动已被取消，不再接受报名。
```

- [ ] **Step 4: Commit**

```bash
git add options/locale/locale_en-US.ini options/locale/locale_zh-CN.ini
git commit -m "i18n(hackforger): add hackathon.error.cancelled key"
```

---

## Task 2: Block registration in API handler

**Files:**
- Modify: `routers/api/v1/hackforger/hackathon_registration.go` (around line 75, just after `IsPublished` check)

- [ ] **Step 1: Add cancelled check after the IsPublished guard**

Current code (around lines 74–78):

```go
	if !h.IsPublished {
		ctx.Error(http.StatusBadRequest, "InvalidPhase", "hackathon is not accepting registrations")
		return
	}
```

Insert immediately after the closing brace of `!h.IsPublished` (so before the organizer self-registration check):

```go
	if h.StatusCache == hackforger_model.HackathonStatusCancelled {
		ctx.Error(http.StatusBadRequest, "Cancelled", "hackathon has been cancelled and is no longer accepting registrations")
		return
	}
```

- [ ] **Step 2: Verify compilation**

```bash
cd /Users/daiming/workspace/hackforger && go build ./routers/api/v1/... 2>&1 | tail -5
```

Expected: clean build.

- [ ] **Step 3: Commit**

```bash
git add routers/api/v1/hackforger/hackathon_registration.go
git commit -m "fix(hackforger): API register handler rejects cancelled hackathons"
```

---

## Task 3: Block registration in Web handler + suppress Register button

**Files:**
- Modify: `routers/web/hackforger/hackathon.go` — two edits: `ViewHackathon` and `RegisterPost`

- [ ] **Step 1: Force CanRegister=false for cancelled in ViewHackathon**

Locate this block in `ViewHackathon` (around lines 212–215):

```go
	canRegister, _ := hackforger_service.AllowsAction(ctx, "hackathon", h.ID, "register")
	ctx.Data["CanRegister"] = canRegister
	canSubmit, _ := hackforger_service.AllowsAction(ctx, "hackathon", h.ID, "submit_work")
	ctx.Data["CanSubmit"] = canSubmit
```

Change the first two lines to:

```go
	canRegister, _ := hackforger_service.AllowsAction(ctx, "hackathon", h.ID, "register")
	canRegister = canRegister && h.StatusCache != hackforger_model.HackathonStatusCancelled
	ctx.Data["CanRegister"] = canRegister
	canSubmit, _ := hackforger_service.AllowsAction(ctx, "hackathon", h.ID, "submit_work")
	ctx.Data["CanSubmit"] = canSubmit
```

- [ ] **Step 2: Add cancelled status check in RegisterPost**

Locate the duplicate-registration check block in `RegisterPost` (around lines 339–344):

```go
	// Check duplicate registration first (more specific error)
	if _, err := hackforger_model.GetRegistration(ctx, h.ID, ctx.Doer.ID); err == nil {
		ctx.Flash.Error(ctx.Tr("hackforger.hackathon.register.already_registered"))
		ctx.Redirect("/hackathon/" + h.Slug)
		return
	}
```

Insert this block immediately **before** the duplicate-registration check (so cancelled error takes priority over the duplicate error, since cancelled is a stronger reason to reject):

```go
	if h.StatusCache == hackforger_model.HackathonStatusCancelled {
		ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.cancelled"))
		ctx.Redirect("/hackathon/" + h.Slug)
		return
	}
```

- [ ] **Step 3: Verify compilation**

```bash
cd /Users/daiming/workspace/hackforger && go build ./routers/web/... 2>&1 | tail -5
```

Expected: clean build.

- [ ] **Step 4: Commit**

```bash
git add routers/web/hackforger/hackathon.go
git commit -m "fix(hackforger): web register handler rejects cancelled hackathons + hides register button"
```

---

## Task 4: Add a test covering cancelled rejection

The test sits at the service-friendly level: cancel hackathon id=3 (Open + is_published=true), then verify it's in Cancelled state and that the existing `AllowsAction` continues to behave as expected (still true, because we deliberately don't delete the phase — that's by design under minimal scope). The handler-level cancelled check is what blocks registration; we test the **state transition** plus a direct **status sanity check** asserting our scope decision.

**Files:**
- Modify: `services/hackforger/hackathon_test.go`

- [ ] **Step 1: Append the test**

Append to `services/hackforger/hackathon_test.go`:

```go
func TestCancelHackathon_DoesNotTouchIsPublished(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// id=3 is "Open Hackathon", is_published=true, status_cache=1 (Open)
	h, err := hackforger_model.GetHackathonByID(db.DefaultContext, 3)
	require.NoError(t, err)
	require.True(t, h.IsPublished, "precondition: hackathon must be published")

	require.NoError(t, hackforger_service.CancelHackathon(db.DefaultContext, 2, h))

	// Reload from DB
	h2, err := hackforger_model.GetHackathonByID(db.DefaultContext, 3)
	require.NoError(t, err)

	// Status moves to Cancelled — this is what the handler-level guard keys off of
	assert.Equal(t, hackforger_model.HackathonStatusCancelled, h2.StatusCache,
		"cancel must set status_cache to Cancelled (5)")

	// is_published deliberately stays true — we want PublishHackathon's
	// "already published" guard to remain a stop on re-publishing,
	// and we want the manage page to keep hiding the Publish button.
	assert.True(t, h2.IsPublished,
		"minimal scope: cancel must NOT modify is_published")
}
```

- [ ] **Step 2: Run the test**

```bash
cd /Users/daiming/workspace/hackforger && \
  TAGS="sqlite sqlite_unlock_notify" go test ./services/hackforger/... \
  -run TestCancelHackathon_DoesNotTouchIsPublished -v 2>&1 | tail -15
```

Expected: PASS.

- [ ] **Step 3: Run the full service suite to confirm no regressions**

```bash
cd /Users/daiming/workspace/hackforger && \
  TAGS="sqlite sqlite_unlock_notify" go test ./services/hackforger/... 2>&1 | tail -10
```

Expected: all green.

- [ ] **Step 4: Commit**

```bash
git add services/hackforger/hackathon_test.go
git commit -m "test(hackforger): verify cancel leaves is_published intact"
```

---

## Task 5: Full backend build + open PR

- [ ] **Step 1: Full build**

```bash
cd /Users/daiming/workspace/hackforger && \
  TAGS="bindata sqlite sqlite_unlock_notify" make backend 2>&1 | tail -10
```

Expected: clean build, no errors.

- [ ] **Step 2: Push and open PR**

```bash
git push -u origin claude/elastic-bartik-049fc5
gh pr create \
  --base v0.1-dev/hackforger \
  --title "fix(hackforger): block registration on cancelled hackathons (issue #167)" \
  --body "$(cat <<'EOF'
## Summary

- Cancelled hackathons now reject new registrations from both the API (`POST /hackforger/hackathons/{id}/register`) and the web form (`POST /hackathon/<slug>/register`)
- Detail page's "Register" button is hidden when the hackathon is cancelled, so users don't see an action that would just fail
- Adds `hackforger.hackathon.error.cancelled` i18n key (zh + en)

## Scope (minimal by design)

- ✅ Block new registrations on cancelled hackathons
- ❌ Do **not** modify `is_published` (keeps existing manage-page UI logic and \`PublishHackathon\` guard intact)
- ❌ Do **not** delete current/future phases (already-registered users can still submit, judges can still score — by design)
- ❌ Refunds / notifications / explore-list filtering deferred to operations or future iterations
- ❌ API write-endpoint permission gap is a separate systemic issue tracked in #169

Closes #167.

## Test plan

- [ ] `TestCancelHackathon_DoesNotTouchIsPublished` passes
- [ ] Full \`services/hackforger\` test suite green
- [ ] Backend builds cleanly
- [ ] Manual: cancel an Open hackathon → API register returns 400 \"Cancelled\"; web register flashes \"该活动已被取消\"; detail page shows no Register button

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```
