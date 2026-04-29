# Prize Distribution Mode Fix — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix issue #77 — `POST /finalize` returns 200 but participant credits don't increase. Add `prize_dist_mode` field to API track form, strict-fail unknown mode in `distributeHackathonCredits`, backfill empty rows, ensure all error surfaces render typed messages instead of "internal error".

**Architecture:** Layered validator at model layer (`ValidatePrizeDistConfig`), called from API/web handlers before persisting. Strict `default` branch in distribution service returns typed error instead of `log.Warn`. Idempotent SQL migration backfills existing empty rows. Six handler touch points (API/web × create/update + finalize) get explicit typed-error branches so errors surface as user-meaningful messages.

**Tech Stack:** Go 1.x, Forgejo's XORM models, web framework, `tests.PrepareTestEnv` integration harness, YAML fixtures.

**Spec:** [docs/superpowers/specs/2026-04-28-prize-dist-mode-fix-design.md](../specs/2026-04-28-prize-dist-mode-fix-design.md)

---

## Pre-flight

- [ ] **P0: Confirm working directory and branch**

```bash
pwd  # expect /Users/h2oslabs/Workspace/hackforger
git status --short  # expect clean except possibly .claude/scheduled_tasks.lock
git rev-parse --abbrev-ref HEAD  # expect v0.1-dev/hackforger
```

- [ ] **P1: Run baseline tests to confirm green starting point**

```bash
go test ./services/hackforger/... -run TestDistributeHackathonCredits -v 2>&1 | tail -30
```

Expected: all four `TestDistributeHackathonCredits_*` tests PASS.

If any fail before changes, STOP and report — the baseline is broken; don't compound.

---

## Task 1: Model layer — `ValidatePrizeDistConfig` + `ErrInvalidPrizeDistMode`

**Files:**
- Modify: `models/hackforger/hackathon_track.go`
- Modify: `models/hackforger/hackathon_track_test.go`

- [ ] **Step 1: Write failing tests for `ValidatePrizeDistConfig`**

Append to `models/hackforger/hackathon_track_test.go`:

```go
func TestValidatePrizeDistConfig(t *testing.T) {
	t.Run("WinnerTakesAll_NoRatios", func(t *testing.T) {
		err := hackforger_model.ValidatePrizeDistConfig("winner_takes_all", "")
		assert.NoError(t, err)
	})

	t.Run("Equal_NoRatios", func(t *testing.T) {
		err := hackforger_model.ValidatePrizeDistConfig("equal", "")
		assert.NoError(t, err)
	})

	t.Run("Tiered_ValidRatios", func(t *testing.T) {
		err := hackforger_model.ValidatePrizeDistConfig("tiered", `[{"rank":1,"pct":60},{"rank":2,"pct":40}]`)
		assert.NoError(t, err)
	})

	t.Run("EmptyMode", func(t *testing.T) {
		err := hackforger_model.ValidatePrizeDistConfig("", "")
		require.Error(t, err)
		assert.True(t, hackforger_model.IsErrInvalidPrizeDistMode(err))
	})

	t.Run("UnknownMode", func(t *testing.T) {
		err := hackforger_model.ValidatePrizeDistConfig("custom", "")
		require.Error(t, err)
		assert.True(t, hackforger_model.IsErrInvalidPrizeDistMode(err))
		assert.Contains(t, err.Error(), `"custom"`)
	})

	t.Run("Tiered_MissingRatios", func(t *testing.T) {
		err := hackforger_model.ValidatePrizeDistConfig("tiered", "")
		require.Error(t, err)
		assert.True(t, hackforger_model.IsErrInvalidDistRatios(err))
	})

	t.Run("Tiered_MalformedJSON", func(t *testing.T) {
		err := hackforger_model.ValidatePrizeDistConfig("tiered", "not json")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid prize distribution ratios JSON")
	})

	t.Run("Tiered_RatiosSumNot100", func(t *testing.T) {
		err := hackforger_model.ValidatePrizeDistConfig("tiered", `[{"rank":1,"pct":50},{"rank":2,"pct":40}]`)
		require.Error(t, err)
		assert.True(t, hackforger_model.IsErrInvalidDistRatios(err))
	})

	t.Run("WinnerTakesAll_RatiosIgnored", func(t *testing.T) {
		// Non-tiered modes don't validate ratios — passing junk ratios is OK
		err := hackforger_model.ValidatePrizeDistConfig("winner_takes_all", "garbage")
		assert.NoError(t, err)
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./models/hackforger/... -run TestValidatePrizeDistConfig -v 2>&1 | tail -20
```

Expected: FAIL with `undefined: hackforger_model.ValidatePrizeDistConfig` and `IsErrInvalidPrizeDistMode`.

- [ ] **Step 3: Implement `ErrInvalidPrizeDistMode` and `ValidatePrizeDistConfig`**

In `models/hackforger/hackathon_track.go`, append after `ValidatePrizeDistRatios` (end of file):

```go
// ErrInvalidPrizeDistMode represents a validation error for prize distribution mode.
type ErrInvalidPrizeDistMode struct {
	Mode string
}

// IsErrInvalidPrizeDistMode checks if an error is a ErrInvalidPrizeDistMode.
func IsErrInvalidPrizeDistMode(err error) bool {
	_, ok := err.(ErrInvalidPrizeDistMode)
	return ok
}

func (err ErrInvalidPrizeDistMode) Error() string {
	return fmt.Sprintf("invalid prize_dist_mode %q (must be winner_takes_all, tiered, or equal)", err.Mode)
}

func (err ErrInvalidPrizeDistMode) Unwrap() error {
	return util.ErrInvalidArgument
}

// ValidatePrizeDistConfig validates a prize distribution configuration.
// mode must be one of: winner_takes_all, tiered, equal.
// For tiered mode, ratiosJSON must parse to a valid PrizeDistRatio slice
// (non-empty, contiguous ranks, percentages summing to 100).
// For non-tiered modes, ratiosJSON is not inspected.
func ValidatePrizeDistConfig(mode, ratiosJSON string) error {
	switch mode {
	case "winner_takes_all", "equal":
		return nil
	case "tiered":
		ratios, err := ParsePrizeDistRatios(ratiosJSON)
		if err != nil {
			return err
		}
		return ValidatePrizeDistRatios(ratios)
	default:
		return ErrInvalidPrizeDistMode{Mode: mode}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./models/hackforger/... -run TestValidatePrizeDistConfig -v 2>&1 | tail -25
```

Expected: all 9 sub-tests PASS.

- [ ] **Step 5: Run full model package tests to confirm no regression**

```bash
go test ./models/hackforger/... 2>&1 | tail -10
```

Expected: PASS (or unchanged from baseline).

- [ ] **Step 6: Commit**

```bash
git add models/hackforger/hackathon_track.go models/hackforger/hackathon_track_test.go
git commit -m "$(cat <<'EOF'
feat(model): add ValidatePrizeDistConfig + ErrInvalidPrizeDistMode

Pure validator for prize distribution config. Returns typed errors
that callers in API/web handlers can branch on.

Refs #77

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Service layer — strict default + fixture cleanup

**Why fixture cleanup is part of this task:** `models/fixtures/hackathon_track.yml` contains tracks 5 and 6 with `prize_dist_mode: "custom"` (an invalid mode) plus `prize_dist_ratios: "50,30,20"` (a non-JSON comma string). Track 6 is in `hackathon_id: 1` and has 3 submissions referencing it. Without fixture cleanup, the new strict `default` branch will break all four existing `TestDistributeHackathonCredits_*` tests because the test loop iterates ALL tracks for hackathon 1.

**Files:**
- Modify: `services/hackforger/hackathon.go`
- Modify: `services/hackforger/hackathon_credits_test.go`
- Modify: `models/fixtures/hackathon_track.yml`

- [ ] **Step 1: Write failing test for strict default branch**

Append to `services/hackforger/hackathon_credits_test.go`:

```go
func TestDistributeHackathonCredits_RejectsUnknownMode(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Construct an in-memory track with empty mode and persist it directly via XORM,
	// bypassing ValidatePrizeDistConfig. This simulates either pre-fix legacy data
	// or a future code path that bypasses the public API.
	tainted := &hackforger_model.HackathonTrack{
		HackathonID:     1,
		Name:            "Tainted Track",
		PrizeCredits:    100,
		PrizeDistMode:   "", // <- the bug condition
		PrizeDistRatios: "",
	}
	_, err := db.GetEngine(db.DefaultContext).Insert(tainted)
	require.NoError(t, err)
	require.NotZero(t, tainted.ID)

	// Add at least one submission so the track has a winner candidate.
	sub := &hackforger_model.HackathonSubmission{
		HackathonID: 1,
		TrackID:     tainted.ID,
		UserID:      2,
		TotalScore:  90.0,
	}
	_, err = db.GetEngine(db.DefaultContext).Insert(sub)
	require.NoError(t, err)

	// Distribute should now FAIL loudly instead of silently skipping.
	err = distributeHackathonCredits(db.DefaultContext, 1)
	require.Error(t, err)
	assert.True(t, hackforger_model.IsErrInvalidPrizeDistMode(err),
		"expected ErrInvalidPrizeDistMode, got %T: %v", err, err)
}
```

- [ ] **Step 2: Run new test to verify it fails**

```bash
go test ./services/hackforger/ -run TestDistributeHackathonCredits_RejectsUnknownMode -v 2>&1 | tail -25
```

Expected: FAIL — currently `default` branch only `log.Warn`s and returns `nil`.

- [ ] **Step 3: Update fixture — set tracks 5 and 6 to valid mode**

Edit `models/fixtures/hackathon_track.yml`. Replace track 5 entry (currently lines ~43-55) and track 6 entry (lines ~57-68) so both use `winner_takes_all` with empty ratios. The exact replacements:

For track 5, change:
```yaml
  prize_dist_mode: "custom"
  prize_dist_ratios: "50,30,20"
```
to:
```yaml
  prize_dist_mode: "winner_takes_all"
  prize_dist_ratios: ""
```

For track 6, change:
```yaml
  prize_dist_mode: "custom"
  prize_dist_ratios: "60,40"
```
to:
```yaml
  prize_dist_mode: "winner_takes_all"
  prize_dist_ratios: ""
```

- [ ] **Step 4: Implement strict default branch in `distributeHackathonCredits`**

Edit `services/hackforger/hackathon.go`. Find the `default:` branch in `distributeHackathonCredits` (around line 952):

```go
		default:
			log.Warn("distributeHackathonCredits: unknown dist mode %q for track %d, skipping", track.PrizeDistMode, track.ID)
		}
```

Replace with:

```go
		default:
			return hackforger_model.ErrInvalidPrizeDistMode{Mode: track.PrizeDistMode}
		}
```

Note: the surrounding `for _, track := range tracks` loop and the `switch track.PrizeDistMode` are unchanged — only the `default` branch body is replaced. The `log` import becomes unused in this function but is still used elsewhere in the file (e.g., `CreateTrackWithRepo`), so no import change needed. Verify:

```bash
grep -c "log\." /Users/h2oslabs/Workspace/hackforger/services/hackforger/hackathon.go
```

Expected: a number > 1 (proving `log` import is still needed).

- [ ] **Step 5: Run new test + existing distribution tests**

```bash
go test ./services/hackforger/ -run TestDistributeHackathonCredits -v 2>&1 | tail -40
```

Expected: ALL 5 tests PASS — `_WinnerTakesAll`, `_Tiered`, `_Equal`, `_EqualRemainder`, `_ZeroPrize`, `_RejectsUnknownMode`. The 4 original tests pass because (a) fixture tracks 5 and 6 now have valid mode, and (b) the loop only walks hackathon 1's tracks which are all valid post-fixture-fix.

- [ ] **Step 6: Run full service package tests for regression check**

```bash
go test ./services/hackforger/... 2>&1 | tail -15
```

Expected: PASS (or unchanged from baseline).

- [ ] **Step 7: Commit**

```bash
git add services/hackforger/hackathon.go services/hackforger/hackathon_credits_test.go models/fixtures/hackathon_track.yml
git commit -m "$(cat <<'EOF'
fix(service): distributeHackathonCredits strict-fails unknown mode

Replace silent log.Warn + skip in default branch with typed
ErrInvalidPrizeDistMode error. Surfaces the bug from #77 instead of
masking it as a 200 OK with no payout.

Fixture tracks 5 and 6 used a placeholder "custom" mode with non-JSON
ratios — corrected to winner_takes_all so existing distribution tests
keep passing under the new strict branch.

Refs #77

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: API form — add fields + validation + error branches

**Files:**
- Modify: `routers/api/v1/hackforger/hackathon_track.go`

- [ ] **Step 1: Add `PrizeDistMode` and `PrizeDistRatios` to both forms**

Edit `routers/api/v1/hackforger/hackathon_track.go`. Replace the form structs (currently lines 16-31):

```go
// CreateTrackForm is the form for creating a hackathon track.
type CreateTrackForm struct {
	Name            string  `json:"name" binding:"Required"`
	Description     string  `json:"description"`
	PrizeAmount     float64 `json:"prize_amount"`
	PrizeCurrency   string  `json:"prize_currency"`
	PrizeCredits    int64   `json:"prize_credits"`
	PrizeDistMode   string  `json:"prize_dist_mode"`
	PrizeDistRatios string  `json:"prize_dist_ratios"`
}

// UpdateTrackForm is the form for updating a hackathon track.
type UpdateTrackForm struct {
	Name            *string  `json:"name"`
	Description     *string  `json:"description"`
	PrizeAmount     *float64 `json:"prize_amount"`
	PrizeCurrency   *string  `json:"prize_currency"`
	PrizeCredits    *int64   `json:"prize_credits"`
	PrizeDistMode   *string  `json:"prize_dist_mode"`
	PrizeDistRatios *string  `json:"prize_dist_ratios"`
}
```

- [ ] **Step 2: Update `CreateTrack` handler — apply default + validate + error branches**

Replace the `CreateTrack` function body (currently lines 85-109). The handler signature and swagger comment block stay intact; replace only the function body:

```go
func CreateTrack(ctx *context.APIContext) {
	f := web.GetForm(ctx).(*CreateTrackForm)
	h, err := hackforger_model.GetHackathonByID(ctx, ctx.ParamsInt64(":id"))
	if err != nil {
		if hackforger_model.IsErrHackathonNotExist(err) {
			ctx.NotFound()
		} else {
			ctx.InternalServerError(err)
		}
		return
	}

	// API parity with web: empty mode defaults to winner_takes_all.
	mode := f.PrizeDistMode
	if mode == "" {
		mode = "winner_takes_all"
	}
	if err := hackforger_model.ValidatePrizeDistConfig(mode, f.PrizeDistRatios); err != nil {
		if hackforger_model.IsErrInvalidPrizeDistMode(err) || hackforger_model.IsErrInvalidDistRatios(err) {
			ctx.Error(http.StatusBadRequest, "ValidatePrizeDistConfig", err)
			return
		}
		ctx.InternalServerError(err)
		return
	}

	t := &hackforger_model.HackathonTrack{
		HackathonID:     h.ID,
		Name:            f.Name,
		Description:     f.Description,
		PrizeAmount:     f.PrizeAmount,
		PrizeCurrency:   f.PrizeCurrency,
		PrizeCredits:    f.PrizeCredits,
		PrizeDistMode:   mode,
		PrizeDistRatios: f.PrizeDistRatios,
	}
	if err := hackforger_service.CreateTrackWithRepo(ctx, ctx.Doer, h, t); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusCreated, t)
}
```

- [ ] **Step 3: Update `UpdateTrack` handler — apply same pattern with pointer semantics**

Replace the `UpdateTrack` function body (currently lines 143-174). Pointer fields mean "absent in request" vs "explicit value":

```go
func UpdateTrack(ctx *context.APIContext) {
	t, err := hackforger_model.GetTrackByID(ctx, ctx.ParamsInt64(":tid"))
	if err != nil {
		if hackforger_model.IsErrTrackNotExist(err) {
			ctx.NotFound()
			return
		}
		ctx.InternalServerError(err)
		return
	}
	f := web.GetForm(ctx).(*UpdateTrackForm)
	if f.Name != nil {
		t.Name = *f.Name
	}
	if f.Description != nil {
		t.Description = *f.Description
	}
	if f.PrizeAmount != nil {
		t.PrizeAmount = *f.PrizeAmount
	}
	if f.PrizeCurrency != nil {
		t.PrizeCurrency = *f.PrizeCurrency
	}
	if f.PrizeCredits != nil {
		t.PrizeCredits = *f.PrizeCredits
	}
	if f.PrizeDistMode != nil {
		mode := *f.PrizeDistMode
		if mode == "" {
			mode = "winner_takes_all"
		}
		t.PrizeDistMode = mode
	}
	if f.PrizeDistRatios != nil {
		t.PrizeDistRatios = *f.PrizeDistRatios
	}

	// Validate the resulting state (after merging request into stored).
	if err := hackforger_model.ValidatePrizeDistConfig(t.PrizeDistMode, t.PrizeDistRatios); err != nil {
		if hackforger_model.IsErrInvalidPrizeDistMode(err) || hackforger_model.IsErrInvalidDistRatios(err) {
			ctx.Error(http.StatusBadRequest, "ValidatePrizeDistConfig", err)
			return
		}
		ctx.InternalServerError(err)
		return
	}

	if err := hackforger_model.UpdateTrack(ctx, t); err != nil {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusOK, t)
}
```

- [ ] **Step 4: Verify file still compiles**

```bash
go build ./routers/api/v1/hackforger/... 2>&1 | head -20
```

Expected: no output (clean build).

- [ ] **Step 5: Commit**

```bash
git add routers/api/v1/hackforger/hackathon_track.go
git commit -m "$(cat <<'EOF'
fix(api): track form accepts prize_dist_mode + validates config

Adds prize_dist_mode and prize_dist_ratios to both CreateTrackForm
and UpdateTrackForm. Empty mode defaults to winner_takes_all (matches
web parity). Invalid mode or malformed tiered ratios return 400 with
typed error.

Closes the API ↔ Web parity gap from #77 — agents creating tracks via
API will no longer produce silently broken finalize flows.

Refs #77

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Web handlers — call validator + add typed flash branches

**Files:**
- Modify: `routers/web/hackforger/hackathon.go`

- [ ] **Step 1: Add validator call to `ManageTrackPost` (track create)**

Find `ManageTrackPost` (currently around line 756). Locate the block right before `t := &hackforger_model.HackathonTrack{...}` (around line 768). Replace the prize-mode block + struct construction (currently lines 762-797) with:

```go
	prizeCredits, _ := strconv.ParseInt(ctx.FormString("prize_credits"), 10, 64)
	prizeDistMode := ctx.FormString("prize_dist_mode")
	if prizeDistMode == "" {
		prizeDistMode = "winner_takes_all"
	}

	var ratiosJSON string
	if prizeDistMode == "tiered" {
		ratiosJSON = strings.TrimSpace(ctx.FormString("prize_dist_ratios"))
	}

	if err := hackforger_model.ValidatePrizeDistConfig(prizeDistMode, ratiosJSON); err != nil {
		switch {
		case hackforger_model.IsErrInvalidPrizeDistMode(err):
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.invalid_prize_dist_mode"))
		case hackforger_model.IsErrInvalidDistRatios(err):
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.invalid_ratios"))
		default:
			log.Error("ValidatePrizeDistConfig: %v", err)
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.internal"))
		}
		ctx.Redirect("/hackathon/" + h.Slug + "/manage")
		return
	}

	t := &hackforger_model.HackathonTrack{
		HackathonID:     h.ID,
		Name:            ctx.FormString("name"),
		Description:     ctx.FormString("description"),
		PrizeCredits:    prizeCredits,
		PrizeDistMode:   prizeDistMode,
		PrizeDistRatios: ratiosJSON,
	}
```

This removes the prior in-line `ParsePrizeDistRatios` + `ValidatePrizeDistRatios` block (now centralized in `ValidatePrizeDistConfig`) and the prior `t.PrizeDistRatios = ratiosJSON` assignment (now done inline in struct literal).

- [ ] **Step 2: Add validator call to `ManageTrackUpdatePost` (track update)**

Find `ManageTrackUpdatePost` (currently around line 806). Replace the prize-mode block (currently lines 821-844) with:

```go
	track.PrizeCredits, _ = strconv.ParseInt(ctx.FormString("prize_credits"), 10, 64)
	track.PrizeDistMode = ctx.FormString("prize_dist_mode")
	if track.PrizeDistMode == "" {
		track.PrizeDistMode = "winner_takes_all"
	}

	var ratiosJSON string
	if track.PrizeDistMode == "tiered" {
		ratiosJSON = strings.TrimSpace(ctx.FormString("prize_dist_ratios"))
	}

	if err := hackforger_model.ValidatePrizeDistConfig(track.PrizeDistMode, ratiosJSON); err != nil {
		switch {
		case hackforger_model.IsErrInvalidPrizeDistMode(err):
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.invalid_prize_dist_mode"))
		case hackforger_model.IsErrInvalidDistRatios(err):
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.invalid_ratios"))
		default:
			log.Error("ValidatePrizeDistConfig: %v", err)
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.internal"))
		}
		ctx.Redirect("/hackathon/" + h.Slug + "/manage")
		return
	}
	track.PrizeDistRatios = ratiosJSON
```

- [ ] **Step 3: Add typed branch to `FinalizeConfirm` (web finalize)**

Find `FinalizeConfirm` (currently around line 1110). Replace the error-handling switch (currently lines 1116-1122) with:

```go
	if err := hackforger_service.ConfirmFinalize(ctx, ctx.Doer.ID, h); err != nil {
		switch {
		case hackforger_model.IsErrInvalidHackathonPhase(err):
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.invalid_phase"))
		case hackforger_model.IsErrInvalidPrizeDistMode(err):
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.invalid_prize_dist_mode"))
		default:
			log.Error("ConfirmFinalize: %v", err)
			ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.internal"))
		}
	} else {
		ctx.Flash.Success(ctx.Tr("hackforger.hackathon.manage.finalize_success"))
	}
	ctx.Redirect("/hackathon/" + h.Slug + "/manage")
```

- [ ] **Step 4: Build to verify imports / syntax are clean**

```bash
go build ./routers/web/hackforger/... 2>&1 | head -20
```

Expected: no output. The file already imports `hackforger_model`, `hackforger_service`, `log`, `strings` — no import changes needed. Verify with:

```bash
head -25 /Users/h2oslabs/Workspace/hackforger/routers/web/hackforger/hackathon.go | grep "\"forgejo.org\|\"strings\""
```

- [ ] **Step 5: Commit**

```bash
git add routers/web/hackforger/hackathon.go
git commit -m "$(cat <<'EOF'
fix(web): track create/update + finalize use typed error branches

ManageTrackPost and ManageTrackUpdatePost call ValidatePrizeDistConfig
(centralized) instead of inline ratio parsing. Both surface typed
errors to Flash with specific i18n messages instead of a generic
"internal error" banner.

FinalizeConfirm gains an IsErrInvalidPrizeDistMode branch so a stored
bad-mode track produces a meaningful flash instead of being swallowed
as internal.

Refs #77

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: API finalize handler — add typed branch + delete dead code

**Files:**
- Modify: `routers/api/v1/hackforger/hackathon_judge.go`
- Modify: `routers/api/v1/hackforger/hackathon.go`

- [ ] **Step 1: Add typed branch to `FinalizeConfirmAPI`**

Edit `routers/api/v1/hackforger/hackathon_judge.go`. Find `FinalizeConfirmAPI` (line ~557). Replace the error-handling block (currently lines 567-573) with:

```go
	if err := hackforger_service.ConfirmFinalize(ctx, ctx.Doer.ID, h); err != nil {
		switch {
		case hackforger_model.IsErrInvalidHackathonPhase(err):
			ctx.Error(http.StatusConflict, "InvalidPhase", err)
		case hackforger_model.IsErrInvalidPrizeDistMode(err):
			ctx.Error(http.StatusUnprocessableEntity, "InvalidPrizeDistMode", err)
		default:
			ctx.InternalServerError(err)
		}
		return
	}
```

- [ ] **Step 2: Delete dead `FinalizeHackathon` handler**

Edit `routers/api/v1/hackforger/hackathon.go`. Verify it's truly unwired:

```bash
grep -n "FinalizeHackathon\b" /Users/h2oslabs/Workspace/hackforger/routers/api/v1/api.go
```

Expected: NO match (only `FinalizeConfirmAPI` should be wired in `api.go`).

If the grep returns no match, delete the dead handler. Find the function in `hackathon.go` (currently around lines 339-366 — it's the swagger comment block + function body). Remove lines starting from the swagger comment header above `func FinalizeHackathon` through the closing brace of the function. Specifically, remove from a line that begins with `// FinalizeHackathon ` (a line above the comment block) down to and including the closing `}` of the function — typically removes ~30 lines.

After deletion, verify the next function (`CancelHackathon`) is now adjacent to the previous function. Build to confirm:

```bash
go build ./routers/api/v1/hackforger/... 2>&1 | head -20
```

Expected: no output.

- [ ] **Step 3: Confirm no test references the deleted handler**

```bash
grep -rn "FinalizeHackathon\b" /Users/h2oslabs/Workspace/hackforger/tests/ /Users/h2oslabs/Workspace/hackforger/routers/ 2>&1 | head -10
```

Expected: NO match (only the one we just deleted should have existed).

- [ ] **Step 4: Build entire backend**

```bash
go build ./... 2>&1 | head -20
```

Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add routers/api/v1/hackforger/hackathon_judge.go routers/api/v1/hackforger/hackathon.go
git commit -m "$(cat <<'EOF'
fix(api): finalize surfaces typed prize-dist error; remove dead handler

FinalizeConfirmAPI gains an IsErrInvalidPrizeDistMode branch returning
422 (semantics: stored data is invalid for the operation requested).

The unrouted FinalizeHackathon handler in hackathon.go was dead code
that flattened every error to 400 with err.Error() exposed — violates
the project's i18n-error-pattern. Removed to eliminate regression
risk if someone re-wires it later.

Refs #77

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: i18n keys (en + zh)

**Files:**
- Modify: `options/locale/locale_en-US.ini`
- Modify: `options/locale/locale_zh-CN.ini`

- [ ] **Step 1: Add English key**

Edit `options/locale/locale_en-US.ini`. Insert the new key under the `[hackforger]`-section error namespace, immediately after the existing `hackathon.error.invalid_phase_action` line (~line 4216):

```ini
hackathon.error.invalid_prize_dist_mode = Invalid prize distribution mode. Must be winner_takes_all, tiered, or equal.
```

- [ ] **Step 2: Add Chinese key in the equivalent location in `locale_zh-CN.ini`**

Find the corresponding `hackathon.error.invalid_phase_action` line in `locale_zh-CN.ini`:

```bash
grep -n "hackathon.error.invalid_phase_action" /Users/h2oslabs/Workspace/hackforger/options/locale/locale_zh-CN.ini
```

Insert immediately after it:

```ini
hackathon.error.invalid_prize_dist_mode = 无效的奖金分配模式。必须是 winner_takes_all、tiered 或 equal 之一。
```

- [ ] **Step 3: Verify both keys are present**

```bash
grep -n "invalid_prize_dist_mode" /Users/h2oslabs/Workspace/hackforger/options/locale/locale_en-US.ini /Users/h2oslabs/Workspace/hackforger/options/locale/locale_zh-CN.ini
```

Expected: exactly 2 lines, one in each file.

- [ ] **Step 4: Commit**

```bash
git add options/locale/locale_en-US.ini options/locale/locale_zh-CN.ini
git commit -m "$(cat <<'EOF'
i18n: add invalid_prize_dist_mode for hackforger track validation

Bilingual key required by project policy (en + zh under [hackforger]).
Surfaced by web Flash and API ctx.Tr fallbacks introduced in #77 fix.

Refs #77

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Migration — backfill empty `prize_dist_mode`

**Files:**
- Create: `models/forgejo_migrations/v14k_clean-empty-prize-dist-mode.go`

- [ ] **Step 1: Create migration file**

Create `models/forgejo_migrations/v14k_clean-empty-prize-dist-mode.go` with content:

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import "xorm.io/xorm"

func init() {
	registerMigration(&Migration{
		Description: "Backfill empty prize_dist_mode to 'winner_takes_all'",
		Upgrade:     cleanEmptyPrizeDistMode,
	})
}

func cleanEmptyPrizeDistMode(x *xorm.Engine) error {
	_, err := x.Exec(
		"UPDATE hackathon_track SET prize_dist_mode = 'winner_takes_all' " +
			"WHERE prize_dist_mode = '' OR prize_dist_mode IS NULL",
	)
	return err
}
```

- [ ] **Step 2: Confirm filename matches the migration filename regex**

```bash
ls /Users/h2oslabs/Workspace/hackforger/models/forgejo_migrations/v14k_clean-empty-prize-dist-mode.go
```

Expected: file listed (no error). The regex `/(?P<group>v[0-9]+[a-z])_(?P<id>[^/]+)\.go$` accepts this name.

- [ ] **Step 3: Build to verify package compiles**

```bash
go build ./models/forgejo_migrations/... 2>&1 | head -10
```

Expected: no output.

- [ ] **Step 4: Smoke-test the migration on the running dev instance**

The dev instance was last built in this session. Apply the migration by restarting:

```bash
bash scripts/restart-gitea.sh 2>&1 | tail -10
```

Expected: HTTP 200 verification passes. Then verify the live data is now clean:

```bash
curl -s -H "Authorization: token $FORGEJO_TOKEN" \
  "${FORGEJO_URL:-https://hackforger.inside.h2os.cloud}/api/v1/hackforger/hackathons/93/tracks" | \
  python3 -c "import sys,json; tracks=json.load(sys.stdin); [print(t['ID'], repr(t['PrizeDistMode'])) for t in tracks]"
```

Expected: track 121 now shows `'winner_takes_all'` (was `''`).

- [ ] **Step 5: Commit**

```bash
git add models/forgejo_migrations/v14k_clean-empty-prize-dist-mode.go
git commit -m "$(cat <<'EOF'
migration(v14k): backfill empty prize_dist_mode to winner_takes_all

One-shot UPDATE for hackathon_track rows that were inserted via API
before the form accepted prize_dist_mode (the bug from #77). Idempotent:
no-op if all rows already valid. Safe in production.

Refs #77

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Integration test — full regression trip

**Files:**
- Create: `tests/integration/hackforger_track_validation_test.go`

This test covers Section 8 row 3 of the spec (single-layer API CreateTrack). The full-trip test (Section 8 row 4 — create→register→submit→score→finalize→assert credits) requires extensive setup that overlaps with `hackforger_hackathon_test.go`'s existing flows. Defer the full trip to manual E2E by the tester (per Section 10) and cover the API contract here.

- [ ] **Step 1: Create the integration test file**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIHackforgerTrackPrizeDistValidation(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	_, adminToken := hackforgerLoginAs(t, 1)

	// Create a draft hackathon to attach tracks to.
	hackathonBody := map[string]any{
		"name": "Track Validation Test",
		"slug": "track-validation-test",
	}
	req := NewRequestWithJSON(t, "POST", "/api/v1/hackforger/hackathons", hackathonBody).
		AddTokenAuth(adminToken)
	resp := MakeRequest(t, req, http.StatusCreated)
	var h struct {
		ID int64 `json:"id"`
	}
	DecodeJSON(t, resp, &h)
	require.NotZero(t, h.ID)

	t.Run("OmittingMode_DefaultsToWinnerTakesAll", func(t *testing.T) {
		body := map[string]any{
			"name":          "Default Mode Track",
			"prize_credits": 100,
		}
		req := NewRequestWithJSON(t, "POST",
			"/api/v1/hackforger/hackathons/"+itoa(h.ID)+"/tracks", body).
			AddTokenAuth(adminToken)
		resp := MakeRequest(t, req, http.StatusCreated)
		var got struct {
			ID            int64  `json:"ID"`
			PrizeDistMode string `json:"PrizeDistMode"`
		}
		DecodeJSON(t, resp, &got)
		assert.Equal(t, "winner_takes_all", got.PrizeDistMode)
	})

	t.Run("ExplicitInvalidMode_Returns400", func(t *testing.T) {
		body := map[string]any{
			"name":            "Bogus Mode Track",
			"prize_credits":   100,
			"prize_dist_mode": "definitely_not_a_mode",
		}
		req := NewRequestWithJSON(t, "POST",
			"/api/v1/hackforger/hackathons/"+itoa(h.ID)+"/tracks", body).
			AddTokenAuth(adminToken)
		MakeRequest(t, req, http.StatusBadRequest)
	})

	t.Run("Tiered_ValidRatios_Created", func(t *testing.T) {
		body := map[string]any{
			"name":              "Tiered OK Track",
			"prize_credits":     100,
			"prize_dist_mode":   "tiered",
			"prize_dist_ratios": `[{"rank":1,"pct":60},{"rank":2,"pct":40}]`,
		}
		req := NewRequestWithJSON(t, "POST",
			"/api/v1/hackforger/hackathons/"+itoa(h.ID)+"/tracks", body).
			AddTokenAuth(adminToken)
		MakeRequest(t, req, http.StatusCreated)
	})

	t.Run("Tiered_BadRatios_Returns400", func(t *testing.T) {
		body := map[string]any{
			"name":              "Tiered Bad Track",
			"prize_credits":     100,
			"prize_dist_mode":   "tiered",
			"prize_dist_ratios": `[{"rank":1,"pct":50},{"rank":2,"pct":40}]`, // sum 90, not 100
		}
		req := NewRequestWithJSON(t, "POST",
			"/api/v1/hackforger/hackathons/"+itoa(h.ID)+"/tracks", body).
			AddTokenAuth(adminToken)
		MakeRequest(t, req, http.StatusBadRequest)
	})

	t.Run("Tiered_MalformedJSON_Returns400", func(t *testing.T) {
		body := map[string]any{
			"name":              "Tiered Malformed Track",
			"prize_credits":     100,
			"prize_dist_mode":   "tiered",
			"prize_dist_ratios": "not even close to json",
		}
		req := NewRequestWithJSON(t, "POST",
			"/api/v1/hackforger/hackathons/"+itoa(h.ID)+"/tracks", body).
			AddTokenAuth(adminToken)
		MakeRequest(t, req, http.StatusBadRequest)
	})
}
```

- [ ] **Step 2: Verify `itoa` and `hackforgerLoginAs` helpers exist**

These are conventions from existing hackforger integration tests:

```bash
grep -rn "^func itoa\|^func hackforgerLoginAs" /Users/h2oslabs/Workspace/hackforger/tests/integration/ 2>&1 | head -5
```

Expected: both functions found in the integration package. If `itoa` is missing, replace `itoa(h.ID)` with `fmt.Sprintf("%d", h.ID)` and add `"fmt"` to imports.

- [ ] **Step 3: Run the new test**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" go test ./tests/integration/ -run TestAPIHackforgerTrackPrizeDistValidation -v 2>&1 | tail -30
```

Expected: all 5 sub-tests PASS.

If the integration harness requires a different invocation per Forgejo conventions, check existing test runs:

```bash
grep -rn "go test.*tests/integration" /Users/h2oslabs/Workspace/hackforger/Makefile 2>&1 | head -5
```

- [ ] **Step 4: Commit**

```bash
git add tests/integration/hackforger_track_validation_test.go
git commit -m "$(cat <<'EOF'
test(integration): API track create validates prize_dist_mode

Five sub-tests cover the API contract surface from the #77 fix:
- omitting mode defaults to winner_takes_all
- explicit unknown mode returns 400
- tiered + valid ratios returns 201
- tiered + sum-not-100 ratios returns 400
- tiered + malformed JSON returns 400

The full-trip regression (create→register→submit→score→finalize→
assert credits) is exercised by the tester via E2E per the spec.

Refs #77

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Final verification + push

- [ ] **Step 1: Run all changed-package tests**

```bash
go test ./models/hackforger/... ./services/hackforger/... 2>&1 | tail -15
```

Expected: PASS.

- [ ] **Step 2: Build full backend with production tags**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend 2>&1 | tail -15
```

Expected: clean build.

- [ ] **Step 3: Restart dev instance and verify live**

```bash
bash scripts/restart-gitea.sh 2>&1 | tail -5
```

Expected: `HTTP 200 on localhost:3000`.

- [ ] **Step 4: Smoke-test the live fix on hackathon 93**

Confirm the migration ran and track 121 is now clean:

```bash
curl -s -H "Authorization: token $FORGEJO_TOKEN" \
  "${FORGEJO_URL:-https://hackforger.inside.h2os.cloud}/api/v1/hackforger/hackathons/93/tracks" | \
  python3 -c "import sys,json; tracks=json.load(sys.stdin); [print(t['ID'], repr(t['PrizeDistMode'])) for t in tracks]"
```

Expected: `121 'winner_takes_all'`.

- [ ] **Step 5: Verify a fresh API CreateTrack defaults correctly**

Pick any draft hackathon owned by the test user:

```bash
NEW_TRACK=$(curl -s -X POST \
  -H "Authorization: token $FORGEJO_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"PR Smoke Track","prize_credits":50}' \
  "${FORGEJO_URL:-https://hackforger.inside.h2os.cloud}/api/v1/hackforger/hackathons/93/tracks")
echo "$NEW_TRACK" | python3 -c "import sys,json; t=json.load(sys.stdin); print('mode=', repr(t.get('PrizeDistMode')))"
```

Expected: `mode= 'winner_takes_all'`.

If creation fails because hackathon 93 is already finalized, pick another hackathon from `GET /api/v1/hackforger/hackathons` that is `Status: Draft` or `Open`. Document the chosen ID in your PR description.

- [ ] **Step 6: Push branch + open PR**

```bash
git push -u origin v0.1-dev/hackforger
```

Wait — the working branch is the default branch. Decide whether to push directly or open a feature branch.

If a feature branch is preferred (recommended for review):

```bash
git checkout -b fix/issue-77-prize-dist-mode
git push -u origin fix/issue-77-prize-dist-mode

gh pr create --title "fix(#77): prize_dist_mode missing from API track form causes silent finalize" --body "$(cat <<'EOF'
## Summary
- Adds `prize_dist_mode` + `prize_dist_ratios` to API track form (parity with web)
- Strict-fails unknown mode in `distributeHackathonCredits` (was: silent log.Warn)
- Backfill migration `v14k` for existing empty rows
- Removes unrouted `FinalizeHackathon` dead code
- Bilingual i18n + typed flash branches in 6 handler touch points

## Root cause
`CreateTrackForm` omitted `PrizeDistMode`. XORM inserted Go zero value `""`, bypassing SQL `DEFAULT 'winner_takes_all'`. `distributeHackathonCredits` `default` switch branch silently skipped the track. `finalize` returned 200 with no payout.

## Test plan
- [x] Unit: `TestValidatePrizeDistConfig` (9 sub-tests) PASS
- [x] Service unit: `TestDistributeHackathonCredits_RejectsUnknownMode` PASS; existing 4 distribution tests still PASS after fixture cleanup
- [x] Integration: `TestAPIHackforgerTrackPrizeDistValidation` (5 sub-tests) PASS
- [x] Smoke: `make backend` clean; dev instance restarts with HTTP 200; track 121 shows `winner_takes_all` post-migration; new API track with no mode field defaults to `winner_takes_all`
- [ ] **Tester E2E re-verification requested in #77 follow-up comment**: re-run hackathon finalize on a fresh hackathon with default mode → assert participant credits balance increases

## Bug 2 (parked, not in this PR)
`current-phase = null` + registration gating — likely UX/skill-doc issue, no clear backend defect from current evidence. Asking the tester (in the #77 comment) to re-run with explicit unix-second timestamps spanning hours and all three phase types (`registration` → `development` → `judging`) so we can confirm. If it still fails, separate spec.

## Skill doc update (separate)
`skills/hackforger-api/references/hackathons.md` should document `prize_dist_mode` and `prize_dist_ratios`. Tracked as a follow-up since the skill repo path is portable and may be vendored to multiple clients independently.

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

- [ ] **Step 7: Post follow-up comment on issue #77**

```bash
gh issue comment 77 --repo HackForger/hackforger --body "$(cat <<'EOF'
PR opened for the **prize distribution / finalize-no-payout** finding.

Code-bug fix:
- API `CreateTrackForm` / `UpdateTrackForm` now accept `prize_dist_mode` + `prize_dist_ratios` (was silently dropped).
- `distributeHackathonCredits` now strict-fails on unknown mode (returns 422) instead of `log.Warn`-ing and pretending success.
- Migration `v14k` backfills existing `prize_dist_mode = ''` rows on the live instance to `winner_takes_all`.

**Please re-verify after merge:**

1. Pick a fresh hackathon (or hackathon 93 — track 121 was backfilled by the migration).
2. Run finalize end-to-end and confirm `GET /api/v1/hackforger/credits/balance` for the winner increases by the track's `PrizeCredits`.

**For the phase / `current-phase = null` finding (Bug 2 in the report):**

Code-bug investigation didn't find a backend defect from current evidence. Most likely the previous setup either (a) used a too-narrow time window that elapsed during the test, or (b) only created a `judging` phase without `registration` (which by design rejects `register` action).

Please reproduce with this exact recipe and report back:
1. Get a fresh `now_unix` from the live host: `date -u +%s`
2. Create three phases via `POST /api/v1/hackforger/hackathons/{id}/phases`, all with windows spanning **hours, not minutes**:
   - `registration`: `start_time = now_unix - 3600`, `end_time = now_unix + 7200`
   - `development`: `start_time = now_unix + 7200`, `end_time = now_unix + 10800`
   - `judging`: `start_time = now_unix + 10800`, `end_time = now_unix + 14400`
3. `GET /api/v1/hackforger/hackathons/{id}/current-phase` — expected: returns the registration phase, not null.
4. `POST /api/v1/hackforger/hackathons/{id}/register` as a non-owner user — expected: succeeds.
5. Capture full curl responses for steps 3 + 4.

If either step fails, attach the curl traces and we'll cut a separate spec.
EOF
)"
```

- [ ] **Step 8: Mark TaskList Bug 1 task done**

```bash
echo "Open Claude Code TaskList and mark Bug 1 (#50) as completed."
```

This is a manual step in the conversation, not a shell command. The plan execution agent should also flag Bug 2 (#51) as still pending — awaiting tester re-verification per the #77 comment.

---

## Self-review notes (against spec)

Spec coverage check (Sections 4, 5, 7, 8, 11, 12 of the spec):

- ✅ Section 4 layered validation → Tasks 1, 3, 4
- ✅ Section 5 touch points all covered: API CreateTrack/UpdateTrack (Task 3); API FinalizeConfirm (Task 5 step 1); API FinalizeHackathon dead-code delete (Task 5 step 2); web ManageTrackPost / ManageTrackUpdatePost (Task 4 steps 1+2); web FinalizeConfirm (Task 4 step 3); service distribute (Task 2 step 4)
- ✅ Section 5 i18n keys → Task 6
- ✅ Section 7 migration → Task 7
- ✅ Section 8 model unit tests → Task 1; service unit (in-memory bypass) → Task 2 step 1; integration single-layer → Task 8. The full regression trip (create→register→submit→score→finalize→assert credits) is delegated to the tester per Section 10 — this matches Section 8's "E2E is not part of this PR."
- ✅ Section 11 file list matches Tasks 1-8 collectively.
- ✅ Section 12 acceptance criteria 1-7 covered by the tests in Tasks 1, 2, 6, 7, 8; AC 8 (dead handler removal) by Task 5 step 2; AC 9 (tester re-verification) by Task 9 step 7.

No placeholders. All code blocks are complete. All file paths are absolute or repo-relative as appropriate.
