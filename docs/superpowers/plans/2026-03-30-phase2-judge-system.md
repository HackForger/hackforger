# Phase 2 Judge System Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade hackathon judging from single-score MVP to multi-criteria scoring with per-track judges, organizer-defined rubrics, interactive Vue scoring UI, and two-step finalize.

**Architecture:** Bottom-up (Models → Services → Routers → Templates → Vue). New `hackathon_judge_criteria` and `hackathon_track_criteria` tables. Existing `hackathon_judge` gains `TrackID`, `hackathon_judge_score` gains `CriteriaID`. Service layer adds criteria management, weighted rank calculation, and two-step finalize. JudgeScoreCard.vue provides interactive scoring.

**Tech Stack:** Go (XORM ORM, go-chi router), Go templates (Fomantic UI), Vue 3 (Options API, lazy-load), SQLite.

**Spec:** `docs/superpowers/specs/2026-03-30-phase2-judge-system-design.md`

**Merge dependency:** Feed refactor (Line B) merges first. Use current `PublishHackforgerAction` format during development; update 2 call sites at merge time.

---

## File Map

### New Files
| File | Responsibility |
|------|---------------|
| `models/hackforger/hackathon_judge_criteria.go` | Criteria model + CRUD + `ErrCriteriaNotExist` |
| `models/hackforger/hackathon_track_criteria.go` | Track criteria override model + CRUD + `SeedTrackCriteria` |
| `services/hackforger/hackathon_criteria.go` | Criteria lifecycle + `EffectiveCriteria` + `GetEffectiveRubric` |
| `web_src/js/components/hackforger/JudgeScoreCard.vue` | Interactive scoring UI |

### Modified Files
| File | Changes |
|------|---------|
| `models/hackforger/hackathon_judge.go` | Add `TrackID`, update UNIQUE, update `ErrDuplicateJudge`, modify all CRUD signatures |
| `models/hackforger/hackathon_judge_score.go` | Add `CriteriaID`, add `UNIQUE(s)` tags, update `ErrDuplicateScore`, add `ListScoresByJudgeAndSubmission`, rewrite `HasAllJudgesScored` |
| `services/hackforger/hackathon_judge.go` | Replace `SubmitScore` → `SubmitScores`, rewrite `CalculateRanks`, add `PersistRanks`, add typed errors |
| `services/hackforger/hackathon.go` | Add criteria check to `StartJudging`, replace `FinalizeHackathon` with `PreviewFinalize` + `ConfirmFinalize`, add `ErrNoCriteria` + `ErrInvalidHackathonPhase` |
| `routers/web/hackforger/hackathon.go` | New handlers: criteria CRUD, track criteria, finalize preview/confirm, rewrite judge page/score, update manage judge |
| `routers/api/v1/hackforger/hackathon_judge.go` | Update forms, add criteria endpoints, update score endpoint, add finalize endpoints |
| `routers/web/web.go:534-549` | Update manage group routes (criteria, finalize), update judge group routes |
| `routers/api/v1/api.go:1788-1793` | Update judge/score API routes, add criteria routes |
| `templates/hackforger/hackathon/manage.tmpl` | Add criteria section, track criteria overrides, enhanced judge section, finalize preview |
| `templates/hackforger/hackathon/judge.tmpl` | Rewrite: Vue mount point + data attributes |
| `templates/hackforger/hackathon/leaderboard.tmpl` | Per-track tabs, criteria breakdown columns |
| `templates/hackforger/hackathon/view.tmpl` | Update judge button visibility |
| `web_src/js/features/hackforger/init.js` | Add JudgeScoreCard lazy-load mount |
| `options/locale/locale_en-US.ini` | Add ~35 i18n keys under `[hackforger]` |
| `options/locale/locale_zh-CN.ini` | Add ~35 i18n keys under `[hackforger]` |

---

## Chunk 1: Model Layer — New Tables + Modified Tables

### Task 1: Create `hackathon_judge_criteria` model

**Files:**
- Create: `models/hackforger/hackathon_judge_criteria.go`

- [ ] **Step 1: Create the criteria model file**

Create `models/hackforger/hackathon_judge_criteria.go` with:
- `HackathonJudgeCriteria` struct (ID, HackathonID, Name, Description, MaxScore, Weight, SortOrder, CreatedUnix, UpdatedUnix)
- `init()` with `db.RegisterModel`
- `ErrCriteriaNotExist` typed error with `IsErrCriteriaNotExist()` and `Unwrap() → util.ErrNotExist`
- CRUD functions: `CreateCriteria`, `UpdateCriteria`, `DeleteCriteria`, `ListCriteriaByHackathon`, `GetCriteriaByID`, `CountCriteriaByHackathon`

Reference `models/hackforger/hackathon_track.go` for the pattern (struct, init, error type, CRUD).

- [ ] **Step 2: Verify compilation**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" go build ./models/hackforger/...`
Expected: Clean compilation.

- [ ] **Step 3: Commit**

```bash
git add models/hackforger/hackathon_judge_criteria.go
git commit -m "feat(judge): add HackathonJudgeCriteria model with CRUD"
```

### Task 2: Create `hackathon_track_criteria` model

**Files:**
- Create: `models/hackforger/hackathon_track_criteria.go`

- [ ] **Step 1: Create the track criteria model file**

Create `models/hackforger/hackathon_track_criteria.go` with:
- `HackathonTrackCriteria` struct (ID, TrackID `UNIQUE(s)`, CriteriaID `UNIQUE(s)`, Enabled, Weight)
- `init()` with `db.RegisterModel`
- CRUD functions: `CreateTrackCriteria`, `UpdateTrackCriteria`, `ListTrackCriteria`, `GetTrackCriteria`
- `SeedTrackCriteria(ctx, trackID int64, criteria []*HackathonJudgeCriteria) error` — idempotent bulk-insert (check existence before insert per criteria+track pair)

- [ ] **Step 2: Verify compilation**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" go build ./models/hackforger/...`
Expected: Clean compilation.

- [ ] **Step 3: Commit**

```bash
git add models/hackforger/hackathon_track_criteria.go
git commit -m "feat(judge): add HackathonTrackCriteria model with CRUD and idempotent seed"
```

### Task 3: Modify `hackathon_judge.go` — add TrackID

**Files:**
- Modify: `models/hackforger/hackathon_judge.go`

- [ ] **Step 1: Update the struct and error type**

In `models/hackforger/hackathon_judge.go`:
- Add `TrackID int64 \`xorm:"UNIQUE(s) INDEX NOT NULL"\`` to `HackathonJudge` struct
- Update `ErrDuplicateJudge` to include `TrackID int64` field
- Update `ErrDuplicateJudge.Error()` to include TrackID in message

- [ ] **Step 2: Update CRUD function signatures**

- `AddJudge(ctx, hackathonID, trackID, userID int64)` — add trackID to existence check and insert
- `RemoveJudge(ctx, hackathonID, trackID, userID int64)` — add trackID to WHERE
- `ListJudges(ctx, hackathonID int64, trackID ...int64)` — optional trackID filter: if len(trackID) > 0, add `AND track_id = ?`
- `IsJudge(ctx, hackathonID, trackID, userID int64)` — add trackID to WHERE
- Add `IsJudgeForAnyTrack(ctx, hackathonID, userID int64) (bool, error)` — checks `hackathon_id=? AND user_id=?` without track filter
- `CountJudges(ctx, hackathonID int64, trackID ...int64)` — optional trackID filter

- [ ] **Step 3: Verify compilation**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" go build ./models/hackforger/...`
Expected: Compilation errors in callers (service/router layers). This is expected — we'll fix them in later tasks.

- [ ] **Step 4: Commit**

```bash
git add models/hackforger/hackathon_judge.go
git commit -m "feat(judge): add TrackID to HackathonJudge model, update CRUD signatures"
```

### Task 4: Modify `hackathon_judge_score.go` — add CriteriaID

**Files:**
- Modify: `models/hackforger/hackathon_judge_score.go`

- [ ] **Step 1: Update the struct and error type**

In `models/hackforger/hackathon_judge_score.go`:
- Add `UNIQUE(s)` tags to `SubmissionID`, `JudgeID` fields
- Add `CriteriaID int64 \`xorm:"UNIQUE(s) INDEX NOT NULL"\``
- Update `ErrDuplicateScore` to include `CriteriaID int64`
- Update `ErrDuplicateScore.Error()` to include CriteriaID

- [ ] **Step 2: Update CRUD functions**

- `GetScore(ctx, judgeID, submissionID, criteriaID int64)` — add `criteria_id = ?` to WHERE
- `CreateScore` — update duplicate check WHERE to include `criteria_id = ?`
- `ListScoresBySubmission` — unchanged (returns all criteria scores for a submission)
- Add `ListScoresByJudgeAndSubmission(ctx, judgeID, submissionID int64) ([]*HackathonJudgeScore, error)` — WHERE `judge_id=? AND submission_id=?`
- Rewrite `HasAllJudgesScored(ctx, hackathonID, trackID, submissionID int64) (bool, error)`:
  1. Get judges for track: `ListJudges(ctx, hackathonID, trackID)`
  2. Get enabled criteria IDs for track (query `hackathon_track_criteria` WHERE `track_id=? AND enabled=true`, join with `hackathon_judge_criteria`)
  3. For each judge: count scores WHERE `judge_id=? AND submission_id=? AND criteria_id IN (?)`
  4. If any judge's count < len(enabled criteria), return false

Note: `HasAllJudgesScored` now depends on `hackathon_track_criteria` and `hackathon_judge_criteria` tables (created in Tasks 1-2). This cross-model dependency is acceptable in the models layer since it's a pure data query.

- [ ] **Step 3: Verify compilation**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" go build ./models/hackforger/...`
Expected: Compilation errors in callers (service layer). Expected — fixed in Chunk 2.

- [ ] **Step 4: Commit**

```bash
git add models/hackforger/hackathon_judge_score.go
git commit -m "feat(judge): add CriteriaID to HackathonJudgeScore, rewrite HasAllJudgesScored"
```

### Task 5: Add typed errors to models

**Files:**
- Modify: `models/hackforger/hackathon.go` (add error types near existing errors)

- [ ] **Step 1: Add model-layer typed errors**

Add to `models/hackforger/hackathon.go` (after existing `ErrHackathonSlugAlreadyExist`):

```go
type ErrInvalidHackathonPhase struct {
    HackathonID int64
    Current     HackathonStatus
    Expected    HackathonStatus
}

func IsErrInvalidHackathonPhase(err error) bool {
    _, ok := err.(ErrInvalidHackathonPhase)
    return ok
}

func (err ErrInvalidHackathonPhase) Error() string {
    return fmt.Sprintf("invalid hackathon phase [id: %d, current: %d, expected: %d]", err.HackathonID, err.Current, err.Expected)
}

func (err ErrInvalidHackathonPhase) Unwrap() error {
    return util.ErrInvalidArgument
}

type ErrNotJudge struct {
    UserID      int64
    HackathonID int64
    TrackID     int64
}

func IsErrNotJudge(err error) bool {
    _, ok := err.(ErrNotJudge)
    return ok
}

func (err ErrNotJudge) Error() string {
    return fmt.Sprintf("user is not a judge [user_id: %d, hackathon_id: %d, track_id: %d]", err.UserID, err.HackathonID, err.TrackID)
}

func (err ErrNotJudge) Unwrap() error {
    return util.ErrPermissionDenied
}

type ErrNoCriteria struct {
    HackathonID int64
}

func IsErrNoCriteria(err error) bool {
    _, ok := err.(ErrNoCriteria)
    return ok
}

func (err ErrNoCriteria) Error() string {
    return fmt.Sprintf("no scoring criteria defined [hackathon_id: %d]", err.HackathonID)
}

func (err ErrNoCriteria) Unwrap() error {
    return util.ErrInvalidArgument
}
```

- [ ] **Step 2: Verify compilation**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" go build ./models/hackforger/...`
Expected: Clean compilation.

- [ ] **Step 3: Commit**

```bash
git add models/hackforger/hackathon.go
git commit -m "feat(judge): add ErrInvalidHackathonPhase, ErrNotJudge, ErrNoCriteria typed errors"
```

---

## Chunk 2: Service Layer

### Task 6: Create criteria service

**Files:**
- Create: `services/hackforger/hackathon_criteria.go`

- [ ] **Step 1: Create the criteria service file**

Create `services/hackforger/hackathon_criteria.go` with:

```go
package hackforger

// EffectiveCriteria represents a criterion with track-level weight resolution.
type EffectiveCriteria struct {
    CriteriaID  int64
    Name        string
    Description string
    MaxScore    float64
    Weight      float64
    SortOrder   int
}
```

Functions:
- `AddCriteria(ctx, hackathonID, name, description, maxScore, weight, sortOrder)` — check hackathon status (Draft/Open only), create criterion, seed all existing tracks using `SeedTrackCriteria`
- `UpdateCriteria(ctx, criteriaID, name, description, maxScore, weight, sortOrder)` — check hackathon status, update
- `RemoveCriteria(ctx, criteriaID)` — check hackathon status, wrap in `db.WithTx`: delete criterion + cascade delete track_criteria + cascade delete judge_scores for that criteria
- `SetTrackCriteriaOverride(ctx, trackID, criteriaID, enabled, weight)` — validate weight > 0 if enabled, update track_criteria
- `GetEffectiveRubric(ctx, trackID)` — join `hackathon_judge_criteria` with `hackathon_track_criteria` on `criteria_id`, filter `enabled=true`, resolve weight (track override if > 0, else criteria default), order by `sort_order`

Status check helper: load criteria → get hackathon → verify status is Draft or Open. If not, return `ErrInvalidHackathonPhase`.

Also add service-layer typed errors:
```go
type ErrScoreOutOfRange struct { CriteriaID int64; Score, MaxScore float64 }
type ErrIncompleteRubric struct { Missing []int64 }
```
With `IsErr*()` and `Unwrap()` for each.

- [ ] **Step 2: Verify compilation**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" go build ./services/hackforger/...`
Expected: Clean compilation (may still have errors from Task 3-4 caller updates — address next).

- [ ] **Step 3: Commit**

```bash
git add services/hackforger/hackathon_criteria.go
git commit -m "feat(judge): add criteria service with EffectiveRubric, CRUD, and typed errors"
```

### Task 7: Update hackathon_judge.go service — SubmitScores + CalculateRanks

**Files:**
- Modify: `services/hackforger/hackathon_judge.go`

- [ ] **Step 1: Add CriteriaScore type and replace SubmitScore with SubmitScores**

In `services/hackforger/hackathon_judge.go`:

Replace the existing `SubmitScore` function with:

```go
type CriteriaScore struct {
    CriteriaID int64
    Score      float64
    Comment    string
}

func SubmitScores(ctx context.Context, judgeID, submissionID int64, scores []CriteriaScore) error {
    // 1. Load submission
    sub, err := hackforger_model.GetSubmissionByID(ctx, submissionID)
    // 2. Load hackathon, verify status == Judging
    // 3. IsJudge(ctx, h.ID, sub.TrackID, judgeID) — if not, return ErrNotJudge
    // 4. GetEffectiveRubric(ctx, sub.TrackID) — get enabled criteria
    // 5. Build map of criteria: validate all enabled criteria present, check score ranges
    // 6. db.WithTx: for each score, get existing → update or create
    // 7. PublishHackforgerAction (use current format, TODO for feed refactor)
}
```

- [ ] **Step 2: Rewrite CalculateRanks, add PersistRanks**

Replace the existing `CalculateRanks` function:

```go
type RankedSubmission struct {
    SubmissionID   int64
    Title          string
    UserID         int64
    TrackID        int64
    WeightedTotal  float64
    CriteriaScores map[int64]float64
    Rank           int
}

// CalculateRanks — pure computation, no DB writes.
func CalculateRanks(ctx context.Context, hackathonID int64) (map[int64][]RankedSubmission, error) {
    tracks, err := hackforger_model.ListTracksByHackathon(ctx, hackathonID)
    if err != nil { return nil, err }
    result := make(map[int64][]RankedSubmission)
    for _, track := range tracks {
        rubric, err := GetEffectiveRubric(ctx, track.ID)
        if err != nil { return nil, err }
        if len(rubric) == 0 { continue }
        subs, _, err := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{
            HackathonID: hackathonID, TrackID: track.ID,
        })
        if err != nil { return nil, err }
        // For each sub: get all scores, compute per-criteria averages,
        // compute weighted total = sum(avg * weight) / sum(weights)
        // Sort by weighted total desc, assign ranks 1-based
        result[track.ID] = ranked
    }
    return result, nil
}

// PersistRanks writes rankings to hackathon_submission table.
func PersistRanks(ctx context.Context, rankings map[int64][]RankedSubmission) error {
    var allRankings []hackforger_model.SubmissionRanking
    for _, subs := range rankings {
        for _, s := range subs {
            allRankings = append(allRankings, hackforger_model.SubmissionRanking{
                SubmissionID: s.SubmissionID, TotalScore: s.WeightedTotal, Rank: s.Rank,
            })
        }
    }
    return hackforger_model.UpdateSubmissionRanks(ctx, allRankings)
}
```

- [ ] **Step 3: Verify compilation**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" go build ./services/hackforger/...`
Expected: Compilation errors in router callers referencing old `SubmitScore`. Expected — fixed in Chunk 3.

- [ ] **Step 4: Commit**

```bash
git add services/hackforger/hackathon_judge.go
git commit -m "feat(judge): replace SubmitScore with multi-criteria SubmitScores, rewrite CalculateRanks"
```

### Task 8: Update hackathon.go service — StartJudging validation + Two-Step Finalize

**Files:**
- Modify: `services/hackforger/hackathon.go`

- [ ] **Step 1: Add criteria check to StartJudging**

In `services/hackforger/hackathon.go`, function `StartJudging` (line ~167):

After the existing `subCount == 0` check, add:
```go
criteriaCount, err := hackforger_model.CountCriteriaByHackathon(ctx, h.ID)
if err != nil { return err }
if criteriaCount == 0 {
    return hackforger_model.ErrNoCriteria{HackathonID: h.ID}
}
```

- [ ] **Step 2: Replace FinalizeHackathon with PreviewFinalize + ConfirmFinalize**

Replace the `FinalizeHackathon` function (lines ~186-212) with:

```go
func PreviewFinalize(ctx context.Context, hackathonID int64) (map[int64][]RankedSubmission, error) {
    h, err := hackforger_model.GetHackathonByID(ctx, hackathonID)
    if err != nil { return nil, err }
    if h.Status != hackforger_model.HackathonStatusJudging {
        return nil, hackforger_model.ErrInvalidHackathonPhase{
            HackathonID: h.ID, Current: h.Status, Expected: hackforger_model.HackathonStatusJudging,
        }
    }
    return CalculateRanks(ctx, hackathonID)
}

func ConfirmFinalize(ctx context.Context, doerID int64, h *hackforger_model.Hackathon) error {
    if h.Status != hackforger_model.HackathonStatusJudging {
        return hackforger_model.ErrInvalidHackathonPhase{...}
    }
    tagTrackRepos(ctx, doerID, h.ID, "submission-deadline", "Submission deadline snapshot")
    rankings, err := CalculateRanks(ctx, h.ID)
    if err != nil { return err }
    if err := PersistRanks(ctx, rankings); err != nil { return err }
    if err := hackforger_model.UpdateHackathonStatus(ctx, h.ID, hackforger_model.HackathonStatusFinished); err != nil { return err }
    createTrackReleases(ctx, doerID, h)
    // Build winners summary from rankings
    trackWinners := make([]map[string]any, 0)
    for trackID, subs := range rankings {
        if len(subs) > 0 {
            trackWinners = append(trackWinners, map[string]any{
                "track_id": trackID, "winner_submission_id": subs[0].SubmissionID, "winner_user_id": subs[0].UserID,
            })
        }
    }
    // TODO: update to new feed API after feed refactor merges
    _ = PublishHackforgerAction(ctx, &HackforgerActionOpts{
        ActUserID: doerID, OpType: hackforger_model.ActionHackathonFinalized,
        AudienceType: AudienceGlobal,
        Content: hackforger_model.HackforgerActionContent{
            EntityType: "hackathon", EntityID: h.ID, EntityName: h.Name, EntitySlug: h.Slug,
            Extra: map[string]any{"tracks": trackWinners},
        },
    })
    return nil
}
```

- [ ] **Step 3: Also replace fmt.Errorf in StartHacking**

Replace `fmt.Errorf("hackathon must be in Open status...")` in `StartHacking` (line ~155) with:
```go
return hackforger_model.ErrInvalidHackathonPhase{HackathonID: h.ID, Current: h.Status, Expected: hackforger_model.HackathonStatusOpen}
```

Do the same for `StartJudging` (line ~169) and `CancelHackathon` where `fmt.Errorf` is used.

- [ ] **Step 4: Verify compilation of services**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" go build ./services/hackforger/...`
Expected: May have errors from router callers — expected.

- [ ] **Step 5: Commit**

```bash
git add services/hackforger/hackathon.go
git commit -m "feat(judge): two-step finalize, criteria validation in StartJudging, typed errors"
```

### Task 9: Update CreateTrackWithRepo to auto-seed criteria

**Files:**
- Modify: `services/hackforger/hackathon.go`

- [ ] **Step 1: Add auto-seed call after track creation**

Find the `CreateTrackWithRepo` function. After the track is created and repo is initialized, add:

```go
// Auto-seed track criteria for any existing hackathon criteria
criteria, _ := hackforger_model.ListCriteriaByHackathon(ctx, hackathonID)
if len(criteria) > 0 {
    _ = hackforger_model.SeedTrackCriteria(ctx, track.ID, criteria)
}
```

- [ ] **Step 2: Verify compilation**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" go build ./services/hackforger/...`

- [ ] **Step 3: Commit**

```bash
git add services/hackforger/hackathon.go
git commit -m "feat(judge): auto-seed track criteria when creating tracks"
```

---

## Chunk 3: Web Routes + API Routes

### Task 10: Update web route registration in web.go

**Files:**
- Modify: `routers/web/web.go:534-549`

**Note:** This task references handler functions defined in Task 11. Do NOT attempt to compile after this task — complete Task 11 first, then compile together.

- [ ] **Step 1: Update manage group routes**

In `routers/web/web.go`, within the `/hackathon/{slug}/manage` group (line 534):

Replace:
```go
m.Post("/finalize", hackforger_web.ManagePhasePost)
```
With:
```go
m.Get("/finalize-preview", hackforger_web.FinalizePreview)
m.Post("/finalize-confirm", hackforger_web.FinalizeConfirm)
```

Add new routes inside the manage group:
```go
m.Post("/criteria", hackforger_web.ManageCriteriaPost)
m.Post("/criteria/{cid}/update", hackforger_web.ManageCriteriaUpdatePost)
m.Post("/criteria/{cid}/delete", hackforger_web.ManageCriteriaDeletePost)
m.Post("/tracks/{tid}/criteria", hackforger_web.ManageTrackCriteriaPost)
```

- [ ] **Step 2: Update judge group routes**

Replace:
```go
m.Post("/{sid}/score", hackforger_web.JudgeScorePost)
```
With:
```go
m.Post("/{sid}/scores", hackforger_web.JudgeScoresPost)
```

- [ ] **Step 3: Commit**

```bash
git add routers/web/web.go
git commit -m "feat(judge): update web route registration for criteria, finalize, scores"
```

### Task 11: Add criteria and finalize web handlers

**Files:**
- Modify: `routers/web/hackforger/hackathon.go`

- [ ] **Step 1: Add criteria management handlers**

Add to `routers/web/hackforger/hackathon.go`:

```go
func ManageCriteriaPost(ctx *context.Context) {
    h := loadHackathon(ctx)
    if h == nil { return }
    name := ctx.FormString("name")
    description := ctx.FormString("description")
    maxScore, _ := strconv.ParseFloat(ctx.FormString("max_score"), 64)
    weight, _ := strconv.ParseFloat(ctx.FormString("weight"), 64)
    sortOrder, _ := strconv.Atoi(ctx.FormString("sort_order"))
    if maxScore <= 0 { maxScore = 10 }
    if weight <= 0 { weight = 25 }
    if err := hackforger_service.AddCriteria(ctx, h.ID, name, description, maxScore, weight, sortOrder); err != nil {
        if hackforger_model.IsErrInvalidHackathonPhase(err) {
            ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.criteria_locked"))
        } else {
            log.Error("AddCriteria: %v", err)
            ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.internal"))
        }
    }
    ctx.Redirect("/hackathon/" + h.Slug + "/manage")
}

func ManageCriteriaUpdatePost(ctx *context.Context) { /* similar pattern */ }
func ManageCriteriaDeletePost(ctx *context.Context) { /* similar pattern */ }
func ManageTrackCriteriaPost(ctx *context.Context) { /* SetTrackCriteriaOverride */ }
```

- [ ] **Step 2: Add finalize preview and confirm handlers**

```go
func FinalizePreview(ctx *context.Context) {
    h := loadHackathon(ctx)
    if h == nil { return }
    rankings, err := hackforger_service.PreviewFinalize(ctx, h.ID)
    if err != nil {
        ctx.Flash.Error(err.Error())
        ctx.Redirect("/hackathon/" + h.Slug + "/manage")
        return
    }
    // Load track names for display
    tracks, _ := hackforger_model.ListTracksByHackathon(ctx, h.ID)
    ctx.Data["Title"] = ctx.Tr("hackforger.hackathon.manage.finalize_preview")
    ctx.Data["Hackathon"] = h
    ctx.Data["Rankings"] = rankings
    ctx.Data["Tracks"] = tracks
    // Load effective rubric per track for column headers
    rubrics := make(map[int64][]*hackforger_service.EffectiveCriteria)
    for _, t := range tracks {
        r, _ := hackforger_service.GetEffectiveRubric(ctx, t.ID)
        rubrics[t.ID] = r
    }
    ctx.Data["Rubrics"] = rubrics
    ctx.HTML(http.StatusOK, tplManage) // reuse manage template with preview section
}

func FinalizeConfirm(ctx *context.Context) {
    h := loadHackathon(ctx)
    if h == nil { return }
    h, _ = hackforger_model.GetHackathonByID(ctx, h.ID) // reload fresh
    if err := hackforger_service.ConfirmFinalize(ctx, ctx.Doer.ID, h); err != nil {
        if hackforger_model.IsErrInvalidHackathonPhase(err) {
            ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.invalid_phase"))
        } else {
            log.Error("ConfirmFinalize: %v", err)
            ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.internal"))
        }
    } else {
        ctx.Flash.Success(ctx.Tr("hackforger.hackathon.manage.finalize_success"))
    }
    ctx.Redirect("/hackathon/" + h.Slug + "/manage")
}
```

- [ ] **Step 3: Remove "finalize" case from ManagePhasePost**

In `ManagePhasePost` (line ~366), remove:
```go
case "finalize":
    err = hackforger_service.FinalizeHackathon(ctx, ctx.Doer.ID, h)
```

Add `IsErrNoCriteria` handling to the error block after the switch:
```go
if hackforger_model.IsErrNoCriteria(err) {
    ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.no_criteria"))
}
```

- [ ] **Step 4: Commit**

```bash
git add routers/web/hackforger/hackathon.go
git commit -m "feat(judge): add criteria CRUD, finalize preview/confirm web handlers"
```

### Task 12: Update judge page and scores handler

**Files:**
- Modify: `routers/web/hackforger/hackathon.go`

- [ ] **Step 1: Rewrite JudgePage with permission check and data loading**

Replace existing `JudgePage` function (line ~456):

```go
func JudgePage(ctx *context.Context) {
    h := loadHackathon(ctx)
    if h == nil { return }
    if h.Status != hackforger_model.HackathonStatusJudging {
        ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.invalid_phase"))
        ctx.Redirect("/hackathon/" + h.Slug)
        return
    }
    // Permission check
    isJudge, _ := hackforger_model.IsJudgeForAnyTrack(ctx, h.ID, ctx.Doer.ID)
    if !isJudge {
        ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.not_judge"))
        ctx.Redirect("/hackathon/" + h.Slug)
        return
    }
    // Load tracks where user is judge
    allJudges, _ := hackforger_model.ListJudges(ctx, h.ID)
    var judgeTracks []int64
    for _, j := range allJudges {
        if j.UserID == ctx.Doer.ID { judgeTracks = append(judgeTracks, j.TrackID) }
    }
    tracks, _ := hackforger_model.ListTracksByHackathon(ctx, h.ID)
    // Filter to judge's tracks, load submissions + rubrics + existing scores per track
    // JSON-encode into data attributes for Vue mount
    // ... (build tracksJSON, subsJSON, rubricsJSON) ...
    ctx.Data["Title"] = ctx.Tr("hackforger.hackathon.judge")
    ctx.Data["Hackathon"] = h
    ctx.Data["TracksJSON"] = tracksJSON
    ctx.Data["SubmissionsJSON"] = subsJSON
    ctx.Data["RubricsJSON"] = rubricsJSON
    ctx.HTML(http.StatusOK, tplJudge)
}
```

- [ ] **Step 2: Replace JudgeScorePost with JudgeScoresPost (JSON handler)**

Replace existing `JudgeScorePost` (line ~473):

```go
func JudgeScoresPost(ctx *context.Context) {
    h := loadHackathon(ctx)
    if h == nil {
        ctx.JSON(http.StatusNotFound, map[string]string{"message": "hackathon not found"})
        return
    }
    sid := ctx.ParamsInt64(":sid")
    // Parse JSON body
    type scoreReq struct {
        Scores []struct {
            CriteriaID int64   `json:"criteria_id"`
            Score      float64 `json:"score"`
            Comment    string  `json:"comment"`
        } `json:"scores"`
    }
    // Note: add "encoding/json" to the import block of this file
    var req scoreReq
    if err := json.NewDecoder(ctx.Req.Body).Decode(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, map[string]string{"message": "invalid request body"})
        return
    }
    scores := make([]hackforger_service.CriteriaScore, len(req.Scores))
    for i, s := range req.Scores {
        scores[i] = hackforger_service.CriteriaScore{CriteriaID: s.CriteriaID, Score: s.Score, Comment: s.Comment}
    }
    if err := hackforger_service.SubmitScores(ctx, ctx.Doer.ID, sid, scores); err != nil {
        msg := "Internal error"
        status := http.StatusInternalServerError
        switch {
        case hackforger_model.IsErrNotJudge(err):
            msg = ctx.Tr("hackforger.hackathon.error.not_judge"); status = http.StatusForbidden
        case hackforger_service.IsErrIncompleteRubric(err):
            msg = ctx.Tr("hackforger.hackathon.error.incomplete_rubric"); status = http.StatusBadRequest
        case hackforger_service.IsErrScoreOutOfRange(err):
            msg = ctx.Tr("hackforger.hackathon.error.score_out_of_range", "10"); status = http.StatusBadRequest
        }
        ctx.JSON(status, map[string]string{"message": msg})
        return
    }
    ctx.JSON(http.StatusOK, map[string]string{"message": "ok"})
}
```

- [ ] **Step 3: Update ManageJudgePost to use username + trackID**

Replace existing `ManageJudgePost` (line ~432):

```go
func ManageJudgePost(ctx *context.Context) {
    h := loadHackathon(ctx)
    if h == nil { return }
    username := ctx.FormString("username")
    trackID, _ := strconv.ParseInt(ctx.FormString("track_id"), 10, 64)
    u, err := user_model.GetUserByName(ctx, username)
    if err != nil {
        ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.user_not_found", username))
        ctx.Redirect("/hackathon/" + h.Slug + "/manage")
        return
    }
    if err := hackforger_model.AddJudge(ctx, h.ID, trackID, u.ID); err != nil {
        if hackforger_model.IsErrDuplicateJudge(err) {
            ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.duplicate_judge"))
        } else {
            log.Error("AddJudge: %v", err)
            ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.internal"))
        }
    }
    ctx.Redirect("/hackathon/" + h.Slug + "/manage")
}
```

Update `ManageJudgeRemovePost`:

```go
func ManageJudgeRemovePost(ctx *context.Context) {
    h := loadHackathon(ctx)
    if h == nil { return }
    userID := ctx.ParamsInt64(":uid")
    trackID, _ := strconv.ParseInt(ctx.FormString("track_id"), 10, 64)
    if err := hackforger_model.RemoveJudge(ctx, h.ID, trackID, userID); err != nil {
        ctx.Flash.Error(err.Error())
    }
    ctx.Redirect("/hackathon/" + h.Slug + "/manage")
}
```

Note: The manage.tmpl judge remove form (Task 15, Step 3) must include a hidden `<input type="hidden" name="track_id" value="{{.TrackID}}">` in each remove button's form.

- [ ] **Step 4: Update ManageHackathon to load criteria data**

In the existing `ManageHackathon` handler, add data loading:

```go
criteria, _ := hackforger_model.ListCriteriaByHackathon(ctx, h.ID)
ctx.Data["Criteria"] = criteria
// Track criteria overrides
trackCriteria := make(map[int64][]*hackforger_model.HackathonTrackCriteria)
for _, t := range tracks {
    tc, _ := hackforger_model.ListTrackCriteria(ctx, t.ID)
    trackCriteria[t.ID] = tc
}
ctx.Data["TrackCriteria"] = trackCriteria
```

- [ ] **Step 5: Update ViewHackathon to use IsJudgeForAnyTrack**

In `ViewHackathon` (line ~207), replace:
```go
isJudge, _ := hackforger_model.IsJudge(ctx, h.ID, ctx.Doer.ID)
```
With:
```go
isJudge, _ := hackforger_model.IsJudgeForAnyTrack(ctx, h.ID, ctx.Doer.ID)
```

- [ ] **Step 6: Update Leaderboard handler**

Enhance `Leaderboard` function to group submissions by track, load effective rubric per track for breakdown columns:

```go
func Leaderboard(ctx *context.Context) {
    h := loadHackathon(ctx)
    if h == nil { return }
    tracks, _ := hackforger_model.ListTracksByHackathon(ctx, h.ID)
    subsByTrack := make(map[int64][]*hackforger_model.HackathonSubmission)
    for _, t := range tracks {
        subs, _, _ := hackforger_model.ListSubmissions(ctx, hackforger_model.ListSubmissionsOptions{
            HackathonID: h.ID, TrackID: t.ID,
        })
        subsByTrack[t.ID] = subs
    }
    // Load effective rubric per track for criteria breakdown columns
    rubrics := make(map[int64][]*hackforger_service.EffectiveCriteria)
    for _, t := range tracks {
        r, _ := hackforger_service.GetEffectiveRubric(ctx, t.ID)
        rubrics[t.ID] = r
    }
    ctx.Data["Title"] = ctx.Tr("hackforger.hackathon.leaderboard")
    ctx.Data["Hackathon"] = h
    ctx.Data["Tracks"] = tracks
    ctx.Data["SubsByTrack"] = subsByTrack
    ctx.Data["Rubrics"] = rubrics
    ctx.Data["ShowBreakdown"] = h.Status == hackforger_model.HackathonStatusFinished
    ctx.HTML(http.StatusOK, tplLeaderboard)
}
```

- [ ] **Step 7: Verify compilation of web routers**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" go build ./routers/web/...`

- [ ] **Step 8: Commit**

```bash
git add routers/web/hackforger/hackathon.go
git commit -m "feat(judge): rewrite judge/manage/leaderboard handlers for multi-criteria"
```

### Task 13: Update API routes and handlers

**Files:**
- Modify: `routers/api/v1/hackforger/hackathon_judge.go`
- Modify: `routers/api/v1/api.go:1788-1793`

- [ ] **Step 1: Update API forms and handlers**

In `routers/api/v1/hackforger/hackathon_judge.go`:

Update `AddJudgeForm`:
```go
type AddJudgeForm struct {
    UserID  int64 `json:"user_id" binding:"Required"`
    TrackID int64 `json:"track_id" binding:"Required"` // NEW
}
```

Update `SubmitScoreForm` → `SubmitScoresForm`:
```go
type SubmitScoresForm struct {
    Scores []struct {
        CriteriaID int64   `json:"criteria_id"`
        Score      float64 `json:"score"`
        Comment    string  `json:"comment"`
    } `json:"scores" binding:"Required"`
}
```

Update `AddJudge` handler to pass `f.TrackID`.
Update `RemoveJudge` to accept `track_id` query param.
Replace `SubmitScore` handler with `SubmitScores`.
Update `GetLeaderboard` to return per-track grouped results.

Add new handlers:
- `ListCriteria`, `AddCriteriaAPI`, `UpdateCriteriaAPI`, `DeleteCriteriaAPI`
- `GetTrackEffectiveRubric`, `SetTrackCriteriaOverrideAPI`
- `FinalizePreviewAPI`, `FinalizeConfirmAPI`

- [ ] **Step 2: Update API route registration**

In `routers/api/v1/api.go`, within the hackathons group (line ~1788):

```go
m.Get("/criteria", hackforger_api.ListCriteria)
m.Post("/criteria", reqToken(), bind(hackforger_api.AddCriteriaForm{}), hackforger_api.AddCriteriaAPI)
m.Put("/criteria/{cid}", reqToken(), bind(hackforger_api.UpdateCriteriaForm{}), hackforger_api.UpdateCriteriaAPI)
m.Delete("/criteria/{cid}", reqToken(), hackforger_api.DeleteCriteriaAPI)
m.Get("/tracks/{tid}/criteria", hackforger_api.GetTrackEffectiveRubric)
m.Put("/tracks/{tid}/criteria/{cid}", reqToken(), bind(hackforger_api.TrackCriteriaOverrideForm{}), hackforger_api.SetTrackCriteriaOverrideAPI)
m.Get("/finalize-preview", reqToken(), hackforger_api.FinalizePreviewAPI)
m.Post("/finalize-confirm", reqToken(), hackforger_api.FinalizeConfirmAPI)
```

Update existing judge/score routes to use new forms.

- [ ] **Step 3: Verify full backend compilation**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make backend`
Expected: Clean compilation.

- [ ] **Step 4: Commit**

```bash
git add routers/api/v1/hackforger/hackathon_judge.go routers/api/v1/api.go
git commit -m "feat(judge): update API routes and handlers for multi-criteria judge system"
```

---

## Chunk 4: Templates + i18n

### Task 14: Add i18n keys

**Files:**
- Modify: `options/locale/locale_en-US.ini`
- Modify: `options/locale/locale_zh-CN.ini`

- [ ] **Step 1: Add English i18n keys**

Add all ~35 new keys from the spec's i18n section to `options/locale/locale_en-US.ini` under the `[hackforger]` section, near the existing `hackathon.judge.*` and `hackathon.manage.*` keys.

Also add these additional keys not in the original spec list (needed by error handlers):
```
hackathon.manage.finalize_success = Hackathon finalized successfully
hackathon.error.user_not_found = User not found: %s
hackathon.error.duplicate_judge = This judge is already assigned to this track.
hackathon.error.internal = An internal error occurred. Please try again.
```

- [ ] **Step 2: Add Chinese i18n keys**

Add corresponding Chinese translations to `options/locale/locale_zh-CN.ini`.

- [ ] **Step 3: Commit**

```bash
git add options/locale/locale_en-US.ini options/locale/locale_zh-CN.ini
git commit -m "feat(judge): add i18n keys for multi-criteria judge system (en-US + zh-CN)"
```

### Task 15: Rewrite manage.tmpl — criteria + track criteria + finalize sections

**Files:**
- Modify: `templates/hackforger/hackathon/manage.tmpl`

- [ ] **Step 1: Add criteria management section**

After the tracks section, add a new `<div class="ui segment">` for criteria management:
- Table listing criteria (name, description, maxScore, weight, sortOrder) with edit/delete forms
- Add criterion form (visible only when status is Draft or Open)
- Warning message when no criteria defined: `{{if and (not .Criteria) (le .Hackathon.Status 2)}}<div class="ui warning message">...</div>{{end}}`

- [ ] **Step 2: Add track criteria overrides section**

After criteria section, add per-track criteria configuration:
- Iterate over tracks, for each track show its criteria overrides
- Checkboxes for enabled/disabled, weight input fields
- POST to `/manage/tracks/{tid}/criteria`

- [ ] **Step 3: Update judge section**

Replace the raw user_id input (line ~31) with:
- Username text input + track dropdown
- Group judges display by track name

- [ ] **Step 4: Add finalize preview section**

When `hackathon.Status == 3` (Judging), replace the old finalize button with:
- "Preview Results" link to `/manage/finalize-preview`
- If `FinalizePreview` data is present, show per-track ranking tables
- "Confirm & Finalize" form with `onclick="return confirm('...')"` confirmation

- [ ] **Step 5: Commit**

```bash
git add templates/hackforger/hackathon/manage.tmpl
git commit -m "feat(judge): add criteria, track overrides, and finalize preview to manage template"
```

### Task 16: Rewrite judge.tmpl — Vue mount point

**Files:**
- Modify: `templates/hackforger/hackathon/judge.tmpl`

- [ ] **Step 1: Replace with Vue mount point**

Replace entire template content with:

```html
{{template "base/head" .}}
<div role="main" aria-label="{{.Title}}" class="page-content hackathon">
    <div class="ui container">
        <h2>{{ctx.Locale.Tr "hackforger.hackathon.judge"}} — {{.Hackathon.Name}}</h2>
        <div id="hackforger-judge-scorecard"
            data-hackathon-slug="{{.Hackathon.Slug}}"
            data-tracks="{{.TracksJSON}}"
            data-submissions="{{.SubmissionsJSON}}"
            data-rubrics="{{.RubricsJSON}}">
        </div>
        <noscript>
            <div class="ui warning message">JavaScript is required for the scoring interface.</div>
        </noscript>
    </div>
</div>
{{template "base/footer" .}}
```

- [ ] **Step 2: Commit**

```bash
git add templates/hackforger/hackathon/judge.tmpl
git commit -m "feat(judge): rewrite judge.tmpl with Vue mount point"
```

### Task 17: Enhance leaderboard.tmpl — per-track tabs

**Files:**
- Modify: `templates/hackforger/hackathon/leaderboard.tmpl`

- [ ] **Step 1: Add per-track tabs and criteria breakdown**

Replace existing leaderboard content with:
- Tab bar iterating over tracks
- Per-track table with Rank, Team/Project, Weighted Total columns
- Criteria breakdown columns (only when `.ShowBreakdown` is true)
- Top 3 highlighting with medal CSS classes

- [ ] **Step 2: Update view.tmpl judge button**

In `templates/hackforger/hackathon/view.tmpl`, the judge button already uses `.IsJudge` (line ~22). The handler now sets this from `IsJudgeForAnyTrack` — no template change needed.

- [ ] **Step 3: Commit**

```bash
git add templates/hackforger/hackathon/leaderboard.tmpl
git commit -m "feat(judge): per-track leaderboard with criteria breakdown"
```

---

## Chunk 5: Vue Component + Frontend

### Task 18: Create JudgeScoreCard.vue

**Files:**
- Create: `web_src/js/components/hackforger/JudgeScoreCard.vue`

- [ ] **Step 1: Create the Vue component**

Create `web_src/js/components/hackforger/JudgeScoreCard.vue` using Options API:

```vue
<script>
import {POST} from '../../modules/fetch.js';

export default {
  props: {
    hackathonSlug: {type: String, required: true},
    tracks: {type: Array, required: true},
    submissions: {type: Object, required: true}, // grouped by trackID
    rubrics: {type: Object, required: true},      // map trackID → criteria[]
  },
  data() {
    return {
      activeTrackId: this.tracks.length ? this.tracks[0].id : 0,
      scores: {},     // {submissionId: {criteriaId: {score, comment}}}
      saving: {},     // {submissionId: bool}
      saved: {},      // {submissionId: bool}
      errors: {},     // {submissionId: string}
    };
  },
  computed: {
    activeSubmissions() {
      return this.submissions[this.activeTrackId] || [];
    },
    activeRubric() {
      return this.rubrics[this.activeTrackId] || [];
    },
    scoredCount() {
      return this.activeSubmissions.filter((s) => this.saved[s.id]).length;
    },
  },
  methods: {
    getScore(subId, cId) {
      return this.scores[subId]?.[cId]?.score ?? '';
    },
    setScore(subId, cId, val) {
      if (!this.scores[subId]) this.scores[subId] = {};
      if (!this.scores[subId][cId]) this.scores[subId][cId] = {score: 0, comment: ''};
      this.scores[subId][cId].score = val;
    },
    getComment(subId, cId) {
      return this.scores[subId]?.[cId]?.comment ?? '';
    },
    setComment(subId, cId, val) {
      if (!this.scores[subId]) this.scores[subId] = {};
      if (!this.scores[subId][cId]) this.scores[subId][cId] = {score: 0, comment: ''};
      this.scores[subId][cId].comment = val;
    },
    async submitScores(submissionId) {
      this.saving[submissionId] = true;
      this.errors[submissionId] = '';
      const subScores = this.scores[submissionId] || {};
      const payload = this.activeRubric.map((c) => ({
        criteria_id: c.criteria_id,
        score: Number(subScores[c.criteria_id]?.score || 0),
        comment: subScores[c.criteria_id]?.comment || '',
      }));
      try {
        const resp = await POST(
          `/hackathon/${this.hackathonSlug}/judge/${submissionId}/scores`,
          {data: {scores: payload}},
        );
        if (resp.ok) {
          this.saved[submissionId] = true;
        } else {
          const data = await resp.json();
          this.errors[submissionId] = data.message || 'Error';
        }
      } catch {
        this.errors[submissionId] = 'Network error';
      }
      this.saving[submissionId] = false;
    },
  },
  created() {
    // Pre-fill existing scores from submissions data
    for (const trackId in this.submissions) {
      for (const sub of this.submissions[trackId]) {
        if (sub.existing_scores) {
          this.scores[sub.id] = {};
          for (const s of sub.existing_scores) {
            this.scores[sub.id][s.criteria_id] = {score: s.score, comment: s.comment};
          }
          // Mark as already scored
          if (sub.existing_scores.length > 0) this.saved[sub.id] = true;
        }
      }
    }
  },
};
</script>

<template>
  <!-- Track tabs -->
  <div class="ui secondary pointing menu" v-if="tracks.length > 1">
    <a v-for="t in tracks" :key="t.id"
       :class="['item', {active: activeTrackId === t.id}]"
       @click="activeTrackId = t.id">
      {{ t.name }}
    </a>
  </div>

  <!-- Progress bar -->
  <div class="ui segment">
    <div class="tw-flex tw-justify-between tw-mb-2">
      <span>{{ scoredCount }} / {{ activeSubmissions.length }} scored</span>
    </div>
    <div class="ui indicating progress" :data-percent="activeSubmissions.length ? (scoredCount / activeSubmissions.length * 100) : 0">
      <div class="bar" :style="{width: (activeSubmissions.length ? scoredCount / activeSubmissions.length * 100 : 0) + '%'}"></div>
    </div>
  </div>

  <!-- Submission cards -->
  <div v-for="sub in activeSubmissions" :key="sub.id" class="ui segment"
       :class="{'positive': saved[sub.id]}">
    <h4>{{ sub.title }}
      <span v-if="saved[sub.id]" class="ui mini green label">Scored</span>
    </h4>
    <p v-if="sub.description">{{ sub.description }}</p>
    <p v-if="sub.demo_url"><a :href="sub.demo_url" target="_blank">Demo</a></p>

    <!-- Criteria form -->
    <div class="ui form">
      <div v-for="c in activeRubric" :key="c.criteria_id" class="field">
        <label>{{ c.name }} (0-{{ c.max_score }}, weight: {{ c.weight }})</label>
        <p v-if="c.description" class="tw-text-sm tw-text-gray">{{ c.description }}</p>
        <div class="two fields">
          <div class="field">
            <input type="number" :min="0" :max="c.max_score" step="0.5"
                   :value="getScore(sub.id, c.criteria_id)"
                   @input="setScore(sub.id, c.criteria_id, $event.target.valueAsNumber)"
                   placeholder="Score">
          </div>
          <div class="field">
            <input type="text"
                   :value="getComment(sub.id, c.criteria_id)"
                   @input="setComment(sub.id, c.criteria_id, $event.target.value)"
                   placeholder="Comment (optional)">
          </div>
        </div>
      </div>
      <button class="ui primary button" :class="{loading: saving[sub.id]}"
              :disabled="saving[sub.id]"
              @click="submitScores(sub.id)">
        Submit Scores
      </button>
      <div v-if="errors[sub.id]" class="ui error message">{{ errors[sub.id] }}</div>
    </div>
  </div>

  <!-- Empty state -->
  <div v-if="!activeSubmissions.length" class="ui placeholder segment">
    <div class="inline"><span>No submissions to judge in this track.</span></div>
  </div>
</template>
```

- [ ] **Step 2: Commit**

```bash
git add web_src/js/components/hackforger/JudgeScoreCard.vue
git commit -m "feat(judge): create JudgeScoreCard.vue interactive scoring component"
```

### Task 19: Register Vue component in init.js

**Files:**
- Modify: `web_src/js/features/hackforger/init.js`

- [ ] **Step 1: Add JudgeScoreCard lazy-load mount**

Add after the BountyPanel block:

```js
// JudgeScoreCard
const judgeEl = document.getElementById('hackforger-judge-scorecard');
if (judgeEl) {
  (async () => {
    const {default: JudgeScoreCard} = await import(
      /* webpackChunkName: "hackforger-judge" */
      '../../components/hackforger/JudgeScoreCard.vue'
    );
    const {createApp} = await import('vue');
    createApp(JudgeScoreCard, {
      hackathonSlug: judgeEl.dataset.hackathonSlug,
      tracks: JSON.parse(judgeEl.dataset.tracks),
      submissions: JSON.parse(judgeEl.dataset.submissions),
      rubrics: JSON.parse(judgeEl.dataset.rubrics),
    }).mount(judgeEl);
  })();
}
```

- [ ] **Step 2: Build frontend**

Run: `make frontend`
Expected: Clean build with new `hackforger-judge` chunk.

- [ ] **Step 3: Commit**

```bash
git add web_src/js/features/hackforger/init.js
git commit -m "feat(judge): register JudgeScoreCard in init.js with lazy-load"
```

---

## Chunk 6: Integration + Build Verification

### Task 20: Full build and smoke test

- [ ] **Step 1: Build backend with embedded assets**

Run: `TAGS="bindata sqlite sqlite_unlock_notify" make build`
Expected: Clean compilation.

- [ ] **Step 2: Build frontend**

Run: `make frontend`
Expected: Clean build.

- [ ] **Step 3: Start server and verify pages load**

```bash
mkdir -p custom/conf
cp /Users/h2oslabs/Workspace/hackforger/custom/conf/app.ini custom/conf/app.ini
rm -f /Users/h2oslabs/Workspace/hackforger/data/queues/common/LOCK
kill $(lsof -t -i :3000) 2>/dev/null; sleep 2
./gitea web
```

Verify:
- `/hackathons/new` loads
- `/explore/hackathons` loads
- No template panics in server logs

- [ ] **Step 4: Commit any fixes**

If any compilation or template issues were found, fix and commit.

### Task 21: Final commit — tag completion

- [ ] **Step 1: Verify all changes are committed**

Run: `git status`
Expected: Clean working tree.

- [ ] **Step 2: Run model tests**

Run: `go test ./models/hackforger/... -v -count=1`
Note: Tests may need test fixture data for new tables. If no test fixtures exist yet, verify compilation only.

- [ ] **Step 3: Run service tests**

Run: `go test ./services/hackforger/... -v -count=1`

- [ ] **Step 4: Final commit if needed**

```bash
git commit -m "feat(judge): Phase 2 judge system — build verified"
```

---

## Post-Implementation

After all tasks are complete:

1. **E2E testing**: Follow `docs/tests/e2e/phase2-judge-e2e-prompt.md` (15 test cases)
2. **Feed integration**: After feed refactor merges, update 2 `PublishHackforgerAction` call sites in `SubmitScores` and `ConfirmFinalize` to use new `HackforgerActionOpts` format
3. **PR creation**: Create PR to `v0.1-dev/hackforger` branch
