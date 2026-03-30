# Phase 2 Judge System Design Spec

## Overview

Upgrade the hackathon judge system from a single-score MVP to a production-quality multi-criteria scoring system with per-track judge assignment, organizer-defined rubrics, per-track weight overrides, interactive Vue scoring UI, and two-step finalize flow.

This is **Week4-Line A** work. Week3 already built the backend skeleton (models, service, API, basic web routes). This phase upgrades it with structured evaluation, proper permissions, and rich UX.

## Scope

### In Scope
- Multi-criteria scoring rubric (organizer-defined per hackathon)
- Per-track criteria override (enable/disable criteria, custom weights per track)
- Per-track judge assignment (judges assigned to specific tracks)
- JudgeScoreCard.vue interactive component (Options API, lazy-load, web routes)
- Two-step finalize: preview rankings → organizer confirms
- Judge permission middleware (verify user is judge for the track)
- Manage page: add judge by username, track assignment, criteria CRUD, track criteria config
- Leaderboard: per-track tabs, weighted score breakdown
- Typed errors replacing `fmt.Errorf()` in service layer
- Feed event: `ActionHackathonFinalized` with results summary
- i18n keys for all new template text (en-US + zh-CN)

### Out of Scope
- AI review endpoint implementation (Week5 — AI assistant)
- Credits awarding on finalize (Week4-Line C)
- Reputation recalculation trigger (Week4-Line B)
- Webhook event emission (Week5)
- Cron auto-phase-transition

## Architecture

### Layer Responsibilities

```
routers/web/hackforger/hackathon.go          — Web handlers (manage, judge, leaderboard pages)
routers/api/v1/hackforger/hackathon_judge.go — REST API handlers
    ↓
services/hackforger/hackathon_criteria.go    — Criteria + track override management (NEW)
services/hackforger/hackathon_judge.go       — Scoring + rank calculation (MODIFY)
services/hackforger/hackathon.go             — Finalize flow (MODIFY)
    ↓
models/hackforger/hackathon_judge_criteria.go — Criteria model + CRUD (NEW)
models/hackforger/hackathon_track_criteria.go — Track criteria overrides (NEW)
models/hackforger/hackathon_judge.go          — Judge assignment + TrackID (MODIFY)
models/hackforger/hackathon_judge_score.go    — Per-criteria scores (MODIFY)
```

Upper layers call lower layers only. Models never call services.

## Data Model

### New Tables

#### `hackathon_judge_criteria`

Scoring rubric items defined at hackathon level. Shared across all tracks (tracks control inclusion/weights via `hackathon_track_criteria`).

```go
type HackathonJudgeCriteria struct {
    ID          int64              `xorm:"pk autoincr"`
    HackathonID int64             `xorm:"INDEX NOT NULL"`
    Name        string            `xorm:"NOT NULL"`           // e.g., "Innovation"
    Description string            `xorm:"TEXT"`               // guidance for judges
    MaxScore    float64           `xorm:"NOT NULL DEFAULT 10"`
    Weight      float64           `xorm:"NOT NULL DEFAULT 25"` // default weight (percentage)
    SortOrder   int               `xorm:"NOT NULL DEFAULT 0"`
    CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
}
```

CRUD functions:
- `CreateCriteria(ctx, c *HackathonJudgeCriteria) error`
- `UpdateCriteria(ctx, c *HackathonJudgeCriteria) error`
- `DeleteCriteria(ctx, id int64) error`
- `ListCriteriaByHackathon(ctx, hackathonID int64) ([]*HackathonJudgeCriteria, error)`
- `GetCriteriaByID(ctx, id int64) (*HackathonJudgeCriteria, error)`
- `CountCriteriaByHackathon(ctx, hackathonID int64) (int64, error)`

#### `hackathon_track_criteria`

Per-track overrides controlling which criteria are active and their weights.

```go
type HackathonTrackCriteria struct {
    ID         int64   `xorm:"pk autoincr"`
    TrackID    int64   `xorm:"UNIQUE(s) INDEX NOT NULL"`
    CriteriaID int64  `xorm:"UNIQUE(s) INDEX NOT NULL"`
    Enabled    bool    `xorm:"NOT NULL DEFAULT true"`
    Weight     float64 `xorm:"NOT NULL DEFAULT 0"` // 0 means "use criteria default"
}
```

CRUD functions:
- `CreateTrackCriteria(ctx, tc *HackathonTrackCriteria) error`
- `UpdateTrackCriteria(ctx, tc *HackathonTrackCriteria) error`
- `ListTrackCriteria(ctx, trackID int64) ([]*HackathonTrackCriteria, error)`
- `GetTrackCriteria(ctx, trackID, criteriaID int64) (*HackathonTrackCriteria, error)`
- `SeedTrackCriteria(ctx, trackID int64, criteria []*HackathonJudgeCriteria) error` — bulk-insert default overrides for a track

### Modified Tables

#### `hackathon_judge` — add TrackID

```go
type HackathonJudge struct {
    ID          int64              `xorm:"pk autoincr"`
    HackathonID int64             `xorm:"UNIQUE(s) INDEX NOT NULL"`
    TrackID     int64             `xorm:"UNIQUE(s) INDEX NOT NULL"` // NEW
    UserID      int64             `xorm:"UNIQUE(s) INDEX NOT NULL"`
    CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
}
```

UNIQUE constraint changes from `(HackathonID, UserID)` to `(HackathonID, TrackID, UserID)`.

Modified CRUD functions:
- `AddJudge(ctx, hackathonID, trackID, userID int64) error` — adds trackID parameter
- `RemoveJudge(ctx, hackathonID, trackID, userID int64) error` — adds trackID parameter
- `ListJudges(ctx, hackathonID int64, trackID ...int64) ([]*HackathonJudge, error)` — optional trackID filter
- `IsJudge(ctx, hackathonID, trackID, userID int64) (bool, error)` — now track-scoped
- `IsJudgeForAnyTrack(ctx, hackathonID, userID int64) (bool, error)` — NEW, for view page button visibility

#### `hackathon_judge_score` — add CriteriaID

```go
type HackathonJudgeScore struct {
    ID           int64              `xorm:"pk autoincr"`
    HackathonID  int64             `xorm:"INDEX NOT NULL"`
    SubmissionID int64             `xorm:"INDEX NOT NULL"`
    JudgeID      int64             `xorm:"INDEX NOT NULL"`
    CriteriaID   int64             `xorm:"INDEX NOT NULL"` // NEW
    Score        float64           `xorm:"NOT NULL DEFAULT 0"`
    Comment      string            `xorm:"TEXT"`
    CreatedUnix  timeutil.TimeStamp `xorm:"INDEX created"`
    UpdatedUnix  timeutil.TimeStamp `xorm:"updated"`
}
```

New UNIQUE constraint: `(JudgeID, SubmissionID, CriteriaID)`.

Modified CRUD functions:
- `GetScore(ctx, judgeID, submissionID, criteriaID int64)` — adds criteriaID
- `CreateScore` / `UpdateScore` — unchanged signature (struct has CriteriaID)
- `ListScoresBySubmission(ctx, submissionID int64)` — returns all criteria scores
- `ListScoresByJudgeAndSubmission(ctx, judgeID, submissionID int64)` — NEW, returns judge's scores for one submission
- `HasAllJudgesScored` — updated to check all judges have scored all enabled criteria

## Service Layer

### New: `services/hackforger/hackathon_criteria.go`

Criteria lifecycle management:

```go
// AddCriteria creates a criterion and auto-seeds track_criteria for all existing tracks.
// Only allowed when hackathon status is Draft or Open.
func AddCriteria(ctx, hackathonID int64, name, description string, maxScore, weight float64, sortOrder int) error

// UpdateCriteria updates a criterion's name/description/maxScore/weight/sortOrder.
// Only allowed when hackathon status is Draft or Open.
func UpdateCriteria(ctx, criteriaID int64, ...) error

// RemoveCriteria deletes a criterion and cascades to track_criteria + judge_score rows.
// Only allowed when hackathon status is Draft or Open.
func RemoveCriteria(ctx, criteriaID int64) error

// SetTrackCriteriaOverride sets enabled/weight for a specific track+criteria pair.
func SetTrackCriteriaOverride(ctx, trackID, criteriaID int64, enabled bool, weight float64) error

// GetEffectiveRubric returns the effective criteria list for a track:
// joins judge_criteria with track_criteria, filters enabled=true,
// returns with track-level weights.
func GetEffectiveRubric(ctx, trackID int64) ([]*EffectiveCriteria, error)
```

`EffectiveCriteria` is a view struct:

```go
type EffectiveCriteria struct {
    CriteriaID  int64
    Name        string
    Description string
    MaxScore    float64
    Weight      float64  // track-level weight (or default if 0)
    SortOrder   int
}
```

Auto-seed trigger points:
- `CreateTrackWithRepo()` — seed track_criteria for all existing hackathon criteria
- `AddCriteria()` — seed track_criteria for all existing tracks of that hackathon

### Modified: `services/hackforger/hackathon_judge.go`

#### SubmitScores (replaces SubmitScore)

```go
// CriteriaScore represents a single criterion's score.
type CriteriaScore struct {
    CriteriaID int64
    Score      float64
    Comment    string
}

// SubmitScores records a judge's scores for all criteria of a submission.
// Validates: hackathon in Judging status, judge assigned to submission's track,
// all enabled criteria scored, each score within [0, maxScore].
// Upsert semantics: creates new or updates existing score records.
func SubmitScores(ctx, judgeID, submissionID int64, scores []CriteriaScore) error
```

Validation steps:
1. Load submission → get hackathon + trackID
2. Verify hackathon status == Judging
3. Verify judge is assigned to submission's track via `IsJudge(ctx, hackathonID, trackID, judgeID)`
4. Load effective rubric for track via `GetEffectiveRubric(ctx, trackID)`
5. Verify all enabled criteria are present in `scores` (no partial submissions)
6. Verify each score is within [0, criterion.MaxScore]
7. Upsert score records (create or update per CriteriaID)
8. Publish `ActionHackathonScored` feed event

#### CalculateRanks (rewrite)

```go
// CalculateRanks computes weighted average scores and assigns ranks per track.
func CalculateRanks(ctx, hackathonID int64) (map[int64][]RankedSubmission, error)
```

Per-track logic:
1. For each track: get effective rubric (criteria + weights)
2. For each submission in that track:
   a. Get all judge scores grouped by criteria
   b. For each criterion: average the scores across all judges
   c. Compute weighted total: `sum(criterion_avg * criterion_weight) / sum(all_weights)`
3. Sort submissions by weighted total descending
4. Assign ranks (1-based, within track)
5. Return `map[trackID][]RankedSubmission` for preview display
6. Optionally persist via `UpdateSubmissionRanks`

```go
type RankedSubmission struct {
    SubmissionID int64
    Title        string
    UserID       int64
    TrackID      int64
    WeightedTotal float64
    CriteriaScores map[int64]float64 // criteriaID → average score
    Rank         int
}
```

### Modified: `services/hackforger/hackathon.go`

#### Two-Step Finalize

```go
// PreviewFinalize calculates rankings for all tracks without changing status.
// Idempotent — organizer can call multiple times to see updated results.
func PreviewFinalize(ctx, hackathonID int64) (map[int64][]RankedSubmission, error)

// ConfirmFinalize locks status to Finished, persists final ranks,
// publishes ActionHackathonFinalized feed event with results summary.
// Only callable from Judging status after PreviewFinalize has been viewed.
func ConfirmFinalize(ctx, hackathonID int64) error
```

### Typed Errors (replace fmt.Errorf in service layer)

```go
type ErrInvalidHackathonPhase struct {
    HackathonID int64
    Current     HackathonStatus
    Expected    HackathonStatus
}

type ErrNotJudge struct {
    UserID      int64
    HackathonID int64
    TrackID     int64
}

type ErrScoreOutOfRange struct {
    CriteriaID int64
    Score      float64
    MaxScore   float64
}

type ErrIncompleteRubric struct {
    Missing []int64 // criteria IDs not scored
}

type ErrCriteriaNotExist struct {
    ID int64
}
```

Each has `IsErr*()` check function and `Unwrap()` returning appropriate `util.Err*`.

## Web Routes

### New Routes

```
POST /hackathon/{slug}/manage/criteria              — add criterion
POST /hackathon/{slug}/manage/criteria/{cid}/update  — update criterion
POST /hackathon/{slug}/manage/criteria/{cid}/delete  — delete criterion
POST /hackathon/{slug}/manage/tracks/{tid}/criteria  — set track criteria override
GET  /hackathon/{slug}/manage/finalize-preview       — preview finalize results
POST /hackathon/{slug}/manage/finalize-confirm       — confirm finalize
POST /hackathon/{slug}/judge/{sid}/scores            — submit all criteria scores (replaces /score)
```

### Modified Routes

```
POST /hackathon/{slug}/manage/judges    — now accepts track_id + username (was user_id)
POST /hackathon/{slug}/manage/judges/{uid}/remove — now accepts track_id
```

### Route Handler Changes

**JudgePage** — add permission check:
1. Check `IsJudgeForAnyTrack(ctx, hackathon.ID, ctx.Doer.ID)` — redirect with error if not
2. Load tracks where user is judge via `ListJudges` filtered by userID
3. For each track: load submissions + effective rubric + judge's existing scores
4. Pass all data to template (or Vue mount point)

**ManageHackathon** — add criteria and track criteria data to template context:
- `ctx.Data["Criteria"]` — all hackathon criteria
- `ctx.Data["TrackCriteria"]` — map[trackID][]TrackCriteria
- `ctx.Data["FinalizePreview"]` — ranking preview (only in Judging status)

**ManageJudgePost** — change from raw user_id input to username lookup:
- Accept `username` + `track_id` form fields
- Resolve username to user via `user_model.GetUserByName`
- Call `AddJudge(ctx, hackathonID, trackID, userID)`

**Leaderboard** — add per-track tabs:
- Load all tracks
- Group submissions by track
- Pass effective criteria per track for score breakdown columns

## API Changes

### New Endpoints

```
POST   /hackathons/{id}/criteria                           — add criterion
GET    /hackathons/{id}/criteria                           — list criteria
PUT    /hackathons/{id}/criteria/{cid}                     — update criterion
DELETE /hackathons/{id}/criteria/{cid}                     — delete criterion
PUT    /hackathons/{id}/tracks/{tid}/criteria/{cid}        — set track criteria override
GET    /hackathons/{id}/tracks/{tid}/criteria               — get effective rubric for track
GET    /hackathons/{id}/finalize-preview                   — preview rankings
POST   /hackathons/{id}/finalize-confirm                   — confirm finalize
```

### Modified Endpoints

```
POST   /hackathons/{id}/judges                    — body now requires track_id
DELETE /hackathons/{id}/judges/{uid}              — body/query now requires track_id
POST   /hackathons/{id}/submissions/{sid}/score   — body changes to scores[] array
GET    /hackathons/{id}/leaderboard               — response grouped by track, includes criteria breakdown
```

## Frontend

### JudgeScoreCard.vue

Interactive Vue component (Options API) for the judge scoring interface.

**Mount point:** `<div id="hackforger-judge-scorecard" data-...>` in `judge.tmpl`

**Props (via data attributes):**
- `hackathonSlug` — for building POST URLs
- `tracks` — JSON array of tracks the judge is assigned to
- `submissions` — JSON array of submissions grouped by track, with existing scores
- `rubrics` — JSON map of trackID → effective criteria list

**Features:**
- Track tab navigation (if judge assigned to multiple tracks)
- Submission cards showing title, description, demo link, PR link
- Per-submission rubric form: criteria name, description tooltip, score input (0-maxScore), comment textarea
- Pre-fill existing scores for editing
- Progress bar: "X of Y submissions scored" per track
- Submit button per submission → POST to `/hackathon/{slug}/judge/{sid}/scores`
- Success/error flash messages
- Visual indicator for completed vs pending submissions

**POST format:**
```json
{
  "scores": [
    {"criteria_id": 1, "score": 8.5, "comment": "Great innovation"},
    {"criteria_id": 2, "score": 7.0, "comment": "Solid implementation"}
  ]
}
```

**Lazy loading:** Registered in `init.js` alongside BountyPanel, same pattern.

### init.js Changes

Add JudgeScoreCard mount block:

```js
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

## Template Changes

### `hackathon/judge.tmpl` — rewrite

Replace bare-bones form with Vue mount point. Template provides:
- Page header with hackathon name + status
- `<div id="hackforger-judge-scorecard">` with data attributes (JSON-encoded)
- Fallback `<noscript>` message

### `hackathon/manage.tmpl` — extend

Add three new sections:

1. **Criteria Management** (visible in Draft/Open status):
   - Table of criteria with name, description, maxScore, weight, sortOrder
   - Add/edit/delete forms
   - Per-track criteria config: table showing each track's enabled/weight overrides

2. **Judge Assignment** (enhanced):
   - Show judges grouped by track with track name
   - Add judge form: username text input + track dropdown (replaces raw user_id)
   - Remove button per judge

3. **Finalize Preview** (visible in Judging status):
   - "Preview Results" button → loads ranking preview
   - Per-track ranking tables with criteria score breakdown
   - "Confirm & Finalize" button with confirmation dialog

### `hackathon/leaderboard.tmpl` — enhance

- Tab navigation per track
- Table columns: Rank, Team/Project, per-criteria average scores, Weighted Total
- Highlight top 3 with medal styling

### `hackathon/view.tmpl` — minor

- "Judge" button visibility: check `IsJudgeForAnyTrack` instead of `IsJudge`
- Show criteria summary section when hackathon has criteria defined

## i18n Keys

New keys needed under `[hackforger]` section (both en-US and zh-CN):

```
hackathon.manage.criteria = Scoring Criteria
hackathon.manage.criteria.add = Add Criterion
hackathon.manage.criteria.name = Criterion Name
hackathon.manage.criteria.description = Description
hackathon.manage.criteria.max_score = Max Score
hackathon.manage.criteria.weight = Weight (%)
hackathon.manage.criteria.sort_order = Display Order
hackathon.manage.criteria.edit = Edit
hackathon.manage.criteria.delete = Delete
hackathon.manage.criteria.delete_confirm = Are you sure you want to delete this criterion?
hackathon.manage.criteria.empty = No scoring criteria defined yet.
hackathon.manage.track_criteria = Track Scoring Overrides
hackathon.manage.track_criteria.enabled = Enabled
hackathon.manage.track_criteria.weight = Weight
hackathon.manage.track_criteria.use_default = Use Default
hackathon.manage.judges.username = Username
hackathon.manage.judges.track = Track
hackathon.manage.judges.select_track = Select Track
hackathon.manage.finalize_preview = Preview Results
hackathon.manage.finalize_confirm = Confirm & Finalize
hackathon.manage.finalize_confirm_dialog = This will lock results and publish them. Continue?
hackathon.manage.finalize_preview.empty = No scored submissions to preview.
hackathon.judge.track = Track
hackathon.judge.criteria = Criteria
hackathon.judge.progress = Scoring Progress
hackathon.judge.progress_detail = %d of %d submissions scored
hackathon.judge.submit_scores = Submit Scores
hackathon.judge.scores_saved = Scores saved successfully
hackathon.judge.no_submissions = No submissions to judge in this track.
hackathon.judge.existing_score = Your existing score
hackathon.leaderboard.criteria_breakdown = Score Breakdown
hackathon.leaderboard.weighted_total = Weighted Total
hackathon.leaderboard.all_tracks = All Tracks
hackathon.error.not_judge = You are not assigned as a judge for this track.
hackathon.error.incomplete_rubric = Please score all criteria before submitting.
hackathon.error.score_out_of_range = Score must be between 0 and %s.
hackathon.error.criteria_locked = Scoring criteria cannot be modified after judging has started.
```

## Feed Events

- `ActionHackathonScored` (existing, type 33) — published per submission when judge submits scores. Content extra: `{"submission_id": N, "track_id": N}`.
- `ActionHackathonFinalized` (existing, type 51) — published on ConfirmFinalize. Content extra: `{"tracks": [{"track_id": N, "winner_submission_id": N, "winner_user_id": N}]}`.

## Error Handling

All service functions return typed errors. Web handlers check error type and use `ctx.Tr()`:

```go
// Example pattern in web handler:
if err := hackforger_service.SubmitScores(ctx, ...); err != nil {
    switch {
    case hackforger_model.IsErrNotJudge(err):
        ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.not_judge"))
    case hackforger_model.IsErrIncompleteRubric(err):
        ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.incomplete_rubric"))
    case hackforger_model.IsErrScoreOutOfRange(err):
        e := err.(hackforger_model.ErrScoreOutOfRange)
        ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.score_out_of_range", fmt.Sprintf("%.0f", e.MaxScore)))
    default:
        ctx.Flash.Error("Internal error")
        log.Error("SubmitScores: %v", err)
    }
    ctx.Redirect(...)
    return
}
```

## Migration Notes

### Schema Migration

The `hackathon_judge` table adds a `TrackID` column. Existing rows (if any from testing) have `TrackID=0` which is invalid. Since this is pre-release, a simple approach works:
- XORM auto-migration adds the columns
- Existing test data with `TrackID=0` is treated as orphaned (ignored in queries that filter by track)
- No formal migration file needed (XORM handles `Sync2` on startup)

### Backward Compatibility

The old single-score `SubmitScore` API endpoint continues to work for one-off scoring (maps to the first criteria), but the primary flow is the new multi-criteria `SubmitScores`. The old endpoint should be deprecated but not removed yet.

## Testing Strategy

### Unit Tests (Go)
- `models/hackforger/`: CRUD for criteria, track_criteria, modified judge/score models
- `services/hackforger/`: SubmitScores validation, CalculateRanks correctness, PreviewFinalize/ConfirmFinalize flow

### E2E Manual Testing
- Full flow: create hackathon → add criteria → create tracks → configure track criteria → add judges per track → register → submit → judge scores all criteria → preview finalize → confirm finalize → verify leaderboard
- Edge cases: judge not assigned to track (blocked), partial rubric submission (blocked), criteria modification after judging started (blocked)
