# Judge Score Feedback + Rubric Validation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the "Submit Scores no reaction" feedback loop, reject empty-rubric silent no-ops, add publish-time criteria validation, and redirect the scored feed to leaderboard — per `docs/superpowers/specs/2026-04-20-judge-score-feedback-design.md`.

**Architecture:** Backend adds two typed errors (`ErrNoCriteria`, `ErrNoRubricConfigured`) with TDD. Existing `checkCriteriaModifiable` becomes op-aware so Judging-phase Add is allowed but Update/Delete stay locked. Frontend enhances `JudgeScoreCard.vue` with toast/summary/sticky-progress via Forgejo-native primitives (`showInfoToast`, `formatDatetime`, per-key `data-locale-*` attrs). Feed template redirects scored events to public leaderboard. A one-shot SQL script cleans dev-env data that violates new rules.

**Tech Stack:** Go 1.23 / XORM, Fomantic UI + Tailwind, Vue 3 Options API, Forgejo locale `.ini` files, SQLite fixtures via `unittest.PrepareTestDatabase`.

**Spec reference:** `docs/superpowers/specs/2026-04-20-judge-score-feedback-design.md`

---

## Build / Test Commands Cheat Sheet

```bash
# Backend unit tests (no network)
go test -tags 'bindata sqlite sqlite_unlock_notify' ./services/hackforger/... -v -run TestName

# Backend build
TAGS="bindata sqlite sqlite_unlock_notify" make backend

# Frontend build (required after JS/Vue/template changes that touch JSON-embedded strings)
make frontend

# Full HackForger server
./gitea web

# Locale lint — both files must have key
grep -c "hackforger.hackathon.judge.score_saved_at" options/locale/locale_en-US.ini options/locale/locale_zh-CN.ini
```

---

## Task 0: Prep — backup DB + cleanup dev data

**Files:**
- Create: `scripts/cleanup-invalid-hackathons.sql`

- [ ] **Step 1: Back up current DB**

```bash
cp data/forgejo.db data/forgejo.db.bak.$(date +%Y%m%d-%H%M%S)
ls -lh data/forgejo.db.bak.*
```

Expected: new file listed with timestamp.

- [ ] **Step 2: Create `scripts/cleanup-invalid-hackathons.sql`**

```sql
-- One-shot dev-environment cleanup for hackathons that violate the post-#27/#28
-- validation rules. NOT a migration — run manually after backing up data/forgejo.db.
-- See docs/superpowers/specs/2026-04-20-judge-score-feedback-design.md § E.

BEGIN;

-- Published hackathons lacking criteria at time of script authoring (2026-04-20).
-- If re-running later, edit this list or extend with a query.
CREATE TEMP TABLE bad_hackathons AS
  SELECT id FROM hackathon WHERE id IN (87, 72, 71, 76, 79);

CREATE TEMP TABLE bad_tracks AS
  SELECT id FROM hackathon_track WHERE hackathon_id IN (SELECT id FROM bad_hackathons);

-- Delete leaf rows first (SQLite foreign_keys PRAGMA is off by default in Forgejo).
DELETE FROM hackathon_judge_score       WHERE hackathon_id IN (SELECT id FROM bad_hackathons);
DELETE FROM hackathon_track_criteria    WHERE track_id     IN (SELECT id FROM bad_tracks);
DELETE FROM hackathon_judge_criteria    WHERE hackathon_id IN (SELECT id FROM bad_hackathons);
DELETE FROM hackathon_submission        WHERE hackathon_id IN (SELECT id FROM bad_hackathons);
DELETE FROM hackathon_registration      WHERE hackathon_id IN (SELECT id FROM bad_hackathons);
DELETE FROM hackathon_judge             WHERE hackathon_id IN (SELECT id FROM bad_hackathons);
DELETE FROM hackathon_track             WHERE hackathon_id IN (SELECT id FROM bad_hackathons);
DELETE FROM phase                       WHERE activity_kind = 'hackathon'
                                          AND activity_id IN (SELECT id FROM bad_hackathons);
DELETE FROM hackforger_action           WHERE entity_type = 'hackathon'
                                          AND entity_id   IN (SELECT id FROM bad_hackathons);
DELETE FROM hackathon                   WHERE id           IN (SELECT id FROM bad_hackathons);

-- Orphan scored feed events from any hackathon (op_type=33 with zero matching scores).
DELETE FROM hackforger_action
 WHERE op_type = 33
   AND NOT EXISTS (
     SELECT 1 FROM hackathon_judge_score
      WHERE hackathon_id = hackforger_action.entity_id
        AND judge_id     = hackforger_action.user_id
   );

COMMIT;
```

- [ ] **Step 3: Execute the script against the running-server DB**

```bash
# Stop the server if running, so SQLite write locks release cleanly.
# (We don't actually need to, since the script is a single transaction, but
#  it avoids surprising other sessions.)
sqlite3 data/forgejo.db < scripts/cleanup-invalid-hackathons.sql

# Verify: 5 violating hackathons gone, scored feed count = actual score row count distinct.
sqlite3 -header data/forgejo.db "SELECT id, slug FROM hackathon WHERE id IN (87,72,71,76,79);"
sqlite3 -header data/forgejo.db "SELECT COUNT(*) AS scored_feeds FROM hackforger_action WHERE op_type=33;"
sqlite3 -header data/forgejo.db "SELECT COUNT(DISTINCT hackathon_id||'-'||judge_id) AS real_scored_pairs FROM hackathon_judge_score;"
```

Expected: first query returns 0 rows; second and third return the same number.

- [ ] **Step 4: Commit**

```bash
git add scripts/cleanup-invalid-hackathons.sql
git commit -m "scripts: add one-shot cleanup for invalid hackathons (#27 #28 prep)"
```

---

## Task 1: Add `ErrNoCriteria` typed error (TDD)

**Files:**
- Modify: `services/hackforger/hackathon.go:68` (append after `ErrNoDevelopmentPhase`)
- Create: `services/hackforger/hackathon_test.go` (new file)

- [ ] **Step 1: Create failing test file `services/hackforger/hackathon_test.go`**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger_test

import (
	"errors"
	"testing"

	"forgejo.org/modules/util"
	hackforger_service "forgejo.org/services/hackforger"

	"github.com/stretchr/testify/assert"
)

func TestErrNoCriteria_Error_Global(t *testing.T) {
	err := hackforger_service.ErrNoCriteria{HackathonID: 42}
	assert.Equal(t, "hackathon has no scoring criteria [id: 42]", err.Error())
}

func TestErrNoCriteria_Error_TrackScoped(t *testing.T) {
	err := hackforger_service.ErrNoCriteria{HackathonID: 42, TrackID: 7}
	assert.Equal(t, "track has no enabled scoring criteria [hackathon: 42, track: 7]", err.Error())
}

func TestErrNoCriteria_IsInvalidArgument(t *testing.T) {
	err := hackforger_service.ErrNoCriteria{HackathonID: 42}
	assert.True(t, errors.Is(err, util.ErrInvalidArgument),
		"ErrNoCriteria should unwrap to util.ErrInvalidArgument so middleware maps to 400")
}

func TestIsErrNoCriteria(t *testing.T) {
	err := hackforger_service.ErrNoCriteria{HackathonID: 1}
	assert.True(t, hackforger_service.IsErrNoCriteria(err))
	assert.False(t, hackforger_service.IsErrNoCriteria(errors.New("unrelated")))
}
```

- [ ] **Step 2: Run tests — verify they fail**

```bash
go test -tags 'bindata sqlite sqlite_unlock_notify' ./services/hackforger/ -run TestErrNoCriteria -v
go test -tags 'bindata sqlite sqlite_unlock_notify' ./services/hackforger/ -run TestIsErrNoCriteria -v
```

Expected: compile error "undefined: hackforger_service.ErrNoCriteria" — exactly the wrong kind of failure. Good.

- [ ] **Step 3: Add the error type to `services/hackforger/hackathon.go`**

Locate `services/hackforger/hackathon.go:68` (end of `IsErrNoDevelopmentPhase`). Immediately after line 68, insert:

```go

// ErrNoCriteria means a hackathon or track has no scoring criteria — either
// no criteria row exists for the hackathon (TrackID == 0) or every criteria
// is disabled via hackathon_track_criteria for the given track (TrackID != 0).
type ErrNoCriteria struct {
	HackathonID int64
	TrackID     int64 // 0 = global; non-zero = specific track has no enabled criteria
}

func (e ErrNoCriteria) Error() string {
	if e.TrackID == 0 {
		return fmt.Sprintf("hackathon has no scoring criteria [id: %d]", e.HackathonID)
	}
	return fmt.Sprintf("track has no enabled scoring criteria [hackathon: %d, track: %d]", e.HackathonID, e.TrackID)
}

// Unwrap lets errors.Is detect invalid-argument and route HTTP 400 — aligns
// with the dominant pattern in services/hackforger/hackathon_criteria.go.
func (e ErrNoCriteria) Unwrap() error { return util.ErrInvalidArgument }

// IsErrNoCriteria checks if err is ErrNoCriteria.
func IsErrNoCriteria(err error) bool { _, ok := err.(ErrNoCriteria); return ok }
```

Also add `"forgejo.org/modules/util"` to the import block at the top (after `"forgejo.org/modules/timeutil"`).

- [ ] **Step 4: Run tests — verify they pass**

```bash
go test -tags 'bindata sqlite sqlite_unlock_notify' ./services/hackforger/ -run TestErrNoCriteria -v
go test -tags 'bindata sqlite sqlite_unlock_notify' ./services/hackforger/ -run TestIsErrNoCriteria -v
```

Expected: all 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add services/hackforger/hackathon.go services/hackforger/hackathon_test.go
git commit -m "feat(hackforger): add ErrNoCriteria typed error"
```

---

## Task 2: `PublishHackathon` rejects hackathon without criteria (TDD)

**Files:**
- Modify: `services/hackforger/hackathon.go:326-372` (`PublishHackathon`)
- Modify: `services/hackforger/hackathon_test.go` (append)
- Modify: `models/fixtures/hackathon.yml`, `models/fixtures/hackathon_track.yml`, `models/fixtures/phase.yml` (add fixture)

- [ ] **Step 1: Check which fixture `phase.yml` exists and its shape**

```bash
ls models/fixtures/phase.yml
cat models/fixtures/phase.yml | head -40
```

Note the schema so Step 3 uses consistent fields.

- [ ] **Step 2: Write the failing test (append to `hackathon_test.go`)**

```go
func TestPublishHackathon_NoCriteria(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Fixture hackathon 7 (see fixtures/hackathon.yml) is Draft, has 1 track,
	// has registration+development phases, but zero criteria.
	h, err := hackforger_model.GetHackathonByID(db.DefaultContext, 7)
	require.NoError(t, err)

	err = hackforger_service.PublishHackathon(db.DefaultContext, 2, h)
	require.Error(t, err)
	assert.True(t, hackforger_service.IsErrNoCriteria(err),
		"expected ErrNoCriteria, got: %T (%v)", err, err)

	typed, ok := err.(hackforger_service.ErrNoCriteria)
	require.True(t, ok)
	assert.Equal(t, int64(7), typed.HackathonID)
	assert.Equal(t, int64(0), typed.TrackID, "global check, not track-scoped")
}
```

Add imports at top if missing:
```go
import (
	// existing
	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"
	"github.com/stretchr/testify/require"
)
```

- [ ] **Step 3: Add fixture hackathon 7 + track 8 + phases**

Append to `models/fixtures/hackathon.yml`:

```yaml
-
  id: 7
  org_id: 3
  owner_id: 2
  name: "Publish Without Criteria"
  slug: "publish-no-criteria"
  description: "Fixture — Draft, 1 track, phases, but zero criteria"
  status_cache: 0
  is_published: false
  max_team_size: 5
  created_unix: 1672578000
  updated_unix: 1672578000
```

Append to `models/fixtures/hackathon_track.yml`:

```yaml
-
  id: 8
  hackathon_id: 7
  name: "Solo Track"
  description: "Only track for publish-no-criteria fixture"
  prize_credits: 0
  prize_dist_mode: "winner_takes_all"
  prize_dist_ratios: ""
  created_unix: 1672578000
```

Append to `models/fixtures/phase.yml` (check existing schema first and mirror it). If the schema has `phase_type_id`, find existing registration + development phase_type_ids for kind `hackathon`:

```bash
grep -B1 -A4 "activity_kind: hackathon" models/fixtures/phase_type.yml | head -30
```

Then add two phases — one registration, one development — for `activity_id: 7`. Use the ids from `phase_type.yml`.

Example (adapt numbers to what phase_type.yml shows):

```yaml
-
  id: 20
  activity_kind: "hackathon"
  activity_id: 7
  phase_type_id: 1     # registration
  start_time: 1672578000
  end_time:   1672664400
  is_locked:  false
-
  id: 21
  activity_kind: "hackathon"
  activity_id: 7
  phase_type_id: 2     # development
  start_time: 1672664400
  end_time:   1672750800
  is_locked:  false
```

- [ ] **Step 4: Run test — verify it fails (call not wired yet)**

```bash
go test -tags 'bindata sqlite sqlite_unlock_notify' ./services/hackforger/ -run TestPublishHackathon_NoCriteria -v
```

Expected: test FAILS with `expected ErrNoCriteria, got: <nil>` — `PublishHackathon` currently succeeds.

- [ ] **Step 5: Add criteria check to `PublishHackathon`**

In `services/hackforger/hackathon.go:362` (right after the `ErrNoDevelopmentPhase` branch, before `SetIsPublished`), insert:

```go
	// Validate scoring criteria — at least one criterion required globally.
	criteriaList, err := hackforger_model.ListCriteriaByHackathon(ctx, h.ID)
	if err != nil {
		return err
	}
	if len(criteriaList) == 0 {
		return ErrNoCriteria{HackathonID: h.ID}
	}

	// Each track must have at least one enabled criterion (via effective rubric).
	tracks, err := hackforger_model.ListTracksByHackathon(ctx, h.ID)
	if err != nil {
		return err
	}
	for _, t := range tracks {
		rubric, err := GetEffectiveRubric(ctx, t.ID)
		if err != nil {
			return err
		}
		if len(rubric) == 0 {
			return ErrNoCriteria{HackathonID: h.ID, TrackID: t.ID}
		}
	}
```

Confirm `hackforger_model.ListCriteriaByHackathon` exists:

```bash
grep -n "func ListCriteriaByHackathon" models/hackforger/
```

If it does not, use the path that the existing `GetEffectiveRubric` uses internally — check `services/hackforger/hackathon_criteria.go` for the correct accessor and substitute.

- [ ] **Step 6: Run test — verify PASS**

```bash
go test -tags 'bindata sqlite sqlite_unlock_notify' ./services/hackforger/ -run TestPublishHackathon_NoCriteria -v
# And make sure we didn't break existing publish tests:
go test -tags 'bindata sqlite sqlite_unlock_notify' ./services/hackforger/ -run TestPublishHackathon -v
```

Expected: new test PASS; any other pre-existing `TestPublishHackathon_*` also PASS.

- [ ] **Step 7: Commit**

```bash
git add services/hackforger/hackathon.go services/hackforger/hackathon_test.go models/fixtures/hackathon.yml models/fixtures/hackathon_track.yml models/fixtures/phase.yml
git commit -m "feat(hackforger): reject publish without scoring criteria (#27 #28)"
```

---

## Task 3: Wire `IsErrNoCriteria` into web publish handler

**Files:**
- Modify: `routers/web/hackforger/hackathon.go` (search `hackforger_service.PublishHackathon`)
- Modify: `options/locale/locale_en-US.ini` (add 2 keys)
- Modify: `options/locale/locale_zh-CN.ini` (add 2 keys)

- [ ] **Step 1: Locate the publish handler**

```bash
grep -n "hackforger_service.PublishHackathon" routers/web/hackforger/hackathon.go
```

Expected: one hit, inside a handler like `ManagePublish`. The handler already has an error-type switch.

- [ ] **Step 2: Add the i18n keys first (so handler references resolve)**

Find existing `hackforger.hackathon.error.no_tracks = ...` in `options/locale/locale_en-US.ini` (around line 4228). Just below it, add:

```ini
hackathon.error.publish_requires_criteria = Cannot publish: at least one scoring criterion is required.
hackathon.error.track_requires_criteria = Track "%s" has no enabled scoring criteria.
```

Repeat the same section in `options/locale/locale_zh-CN.ini` with:

```ini
hackathon.error.publish_requires_criteria = 无法发布：至少需要配置一个评分标准。
hackathon.error.track_requires_criteria = 赛道 "%s" 没有启用任何评分项。
```

- [ ] **Step 3: Verify both locales have the keys**

```bash
grep -n "publish_requires_criteria\|track_requires_criteria" options/locale/locale_en-US.ini options/locale/locale_zh-CN.ini
```

Expected: 4 lines (2 per file).

- [ ] **Step 4: Add the handler branch**

In the publish handler, find the switch/ladder that inspects the returned error (it already checks `IsErrNoTracks`, `IsErrNoRegistrationPhase`, etc.). Add a new branch **before** the generic fallback:

```go
		case hackforger_service.IsErrNoCriteria(err):
			typed := err.(hackforger_service.ErrNoCriteria)
			if typed.TrackID == 0 {
				ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.publish_requires_criteria"))
			} else {
				// Resolve track name for the user-facing message.
				var trackName string
				if t, terr := hackforger_model.GetTrackByID(ctx, typed.TrackID); terr == nil && t != nil {
					trackName = t.Name
				} else {
					trackName = fmt.Sprintf("#%d", typed.TrackID)
				}
				ctx.Flash.Error(ctx.Tr("hackforger.hackathon.error.track_requires_criteria", trackName))
			}
```

If `hackforger_model.GetTrackByID` is not imported in this file yet, add it. Confirm the function exists:

```bash
grep -n "func GetTrackByID" models/hackforger/
```

If absent, grep for whatever the existing accessor is (e.g. `GetHackathonTrackByID`) and use that.

- [ ] **Step 5: Build backend**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend
```

Expected: clean build. Any compile error likely means the track accessor name is different — fix and re-run.

- [ ] **Step 6: Manual smoke test**

```bash
./gitea web &
# In a second shell:
agent-browser open http://localhost:3000/user/login
# Log in as hackforger / admin1234
# Visit /hackathon/publish-no-criteria/manage and click "发布"
# Expect red flash: 无法发布：至少需要配置一个评分标准。
pkill -f "./gitea web"
```

(If the fixture hackathon is only loaded for tests, create an equivalent via UI in the running DB for the smoke check — Draft hackathon, 1 track, phases added, no criteria.)

- [ ] **Step 7: Commit**

```bash
git add routers/web/hackforger/hackathon.go options/locale/locale_en-US.ini options/locale/locale_zh-CN.ini
git commit -m "feat(hackforger): surface ErrNoCriteria as flash error on publish"
```

---

## Task 4: Add `ErrNoRubricConfigured` typed error (TDD)

**Files:**
- Modify: `services/hackforger/hackathon_judge.go` (top of file, after `CriteriaScore` struct)
- Create: `services/hackforger/hackathon_judge_test.go` (new file)

- [ ] **Step 1: Create `services/hackforger/hackathon_judge_test.go` with failing tests**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger_test

import (
	"errors"
	"testing"

	"forgejo.org/modules/util"
	hackforger_service "forgejo.org/services/hackforger"

	"github.com/stretchr/testify/assert"
)

func TestErrNoRubricConfigured_Error(t *testing.T) {
	err := hackforger_service.ErrNoRubricConfigured{TrackID: 99}
	assert.Equal(t, "no scoring rubric configured for track [track: 99]", err.Error())
}

func TestErrNoRubricConfigured_IsInvalidArgument(t *testing.T) {
	err := hackforger_service.ErrNoRubricConfigured{TrackID: 1}
	assert.True(t, errors.Is(err, util.ErrInvalidArgument))
}

func TestIsErrNoRubricConfigured(t *testing.T) {
	err := hackforger_service.ErrNoRubricConfigured{TrackID: 1}
	assert.True(t, hackforger_service.IsErrNoRubricConfigured(err))
	assert.False(t, hackforger_service.IsErrNoRubricConfigured(errors.New("x")))
}
```

- [ ] **Step 2: Run — verify fail**

```bash
go test -tags 'bindata sqlite sqlite_unlock_notify' ./services/hackforger/ -run TestErrNoRubricConfigured -v
go test -tags 'bindata sqlite sqlite_unlock_notify' ./services/hackforger/ -run TestIsErrNoRubricConfigured -v
```

Expected: compile errors "undefined: hackforger_service.ErrNoRubricConfigured".

- [ ] **Step 3: Add the type to `services/hackforger/hackathon_judge.go`**

Insert after the `CriteriaScore` struct block (around line 21). First check imports: if `"fmt"` and `"forgejo.org/modules/util"` aren't imported, add them.

```go

// ErrNoRubricConfigured means the track being scored has no active scoring
// criteria — organizer-side configuration error. Judge UI surfaces this as
// "please contact the organizer".
type ErrNoRubricConfigured struct{ TrackID int64 }

func (e ErrNoRubricConfigured) Error() string {
	return fmt.Sprintf("no scoring rubric configured for track [track: %d]", e.TrackID)
}

func (e ErrNoRubricConfigured) Unwrap() error { return util.ErrInvalidArgument }

// IsErrNoRubricConfigured checks if err is ErrNoRubricConfigured.
func IsErrNoRubricConfigured(err error) bool { _, ok := err.(ErrNoRubricConfigured); return ok }
```

- [ ] **Step 4: Run — verify PASS**

```bash
go test -tags 'bindata sqlite sqlite_unlock_notify' ./services/hackforger/ -run TestErrNoRubricConfigured -v
go test -tags 'bindata sqlite sqlite_unlock_notify' ./services/hackforger/ -run TestIsErrNoRubricConfigured -v
```

Expected: 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add services/hackforger/hackathon_judge.go services/hackforger/hackathon_judge_test.go
git commit -m "feat(hackforger): add ErrNoRubricConfigured typed error"
```

---

## Task 5: `SubmitScores` rejects empty rubric early (TDD)

**Files:**
- Modify: `services/hackforger/hackathon_judge.go` (`SubmitScores`, after rubric load)
- Modify: `services/hackforger/hackathon_judge_test.go` (append)

- [ ] **Step 1: Append failing test**

```go
func TestSubmitScores_EmptyRubric(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Fixture needed: a hackathon in Judging with track + judge + submission but zero criteria.
	// We construct it ad-hoc using existing fixtures by removing criteria for the duration of the test.
	// Alternative: add hackathon 8 fixture. Here we add a new hackathon_judge for track 5 (hackathon 4)
	// which has a track but no criteria for that track.
	//
	// (Track 5 belongs to hackathon 4 which is Hacking phase — "score" action needs Judging.
	//  So we use hackathon 1 + track 4 ["Zero Prize Track"] which has NO track_criteria row
	//  AND we delete the hackathon-level criteria for the duration of this test — too fragile.)
	//
	// Decision: write a fixture.

	// Use fixture judge scored attempt on hackathon 8 (Judging, 1 track, 1 judge, 1 submission, zero criteria)
	// judge user 2, submission 100 (from fixtures)
	err := hackforger_service.SubmitScores(db.DefaultContext, 2, 100, []hackforger_service.CriteriaScore{})
	require.Error(t, err)
	assert.True(t, hackforger_service.IsErrNoRubricConfigured(err),
		"expected ErrNoRubricConfigured, got %T (%v)", err, err)
}
```

- [ ] **Step 2: Add fixture hackathon 8 + track 9 + judge + submission**

Append to `models/fixtures/hackathon.yml`:

```yaml
-
  id: 8
  org_id: 3
  owner_id: 2
  name: "Judging Without Rubric"
  slug: "judging-no-rubric"
  description: "Fixture — Judging phase, judges assigned, submission, but zero criteria"
  status_cache: 3
  is_published: true
  max_team_size: 5
  created_unix: 1672578000
  updated_unix: 1672578000
```

Append to `models/fixtures/hackathon_track.yml`:

```yaml
-
  id: 9
  hackathon_id: 8
  name: "Track Without Criteria"
  description: "For SubmitScores empty rubric test"
  prize_credits: 0
  prize_dist_mode: "winner_takes_all"
  prize_dist_ratios: ""
  created_unix: 1672578000
```

Append to `models/fixtures/hackathon_judge.yml`:

```bash
cat models/fixtures/hackathon_judge.yml | tail -5  # check last id used
```

Then append (replace `<next_id>`):

```yaml
-
  id: <next_id>
  hackathon_id: 8
  user_id: 2
  track_id: 9
  created_unix: 1672578000
```

Append to `models/fixtures/hackathon_submission.yml`:

```yaml
-
  id: 100
  hackathon_id: 8
  track_id: 9
  user_id: 4
  title: "Submission For Empty Rubric Test"
  description: ""
  demo_url: ""
  created_unix: 1672578000
  updated_unix: 1672578000
```

Add a Judging phase in `models/fixtures/phase.yml` for `activity_id: 8`:

```yaml
-
  id: 22
  activity_kind: "hackathon"
  activity_id: 8
  phase_type_id: 3    # judging — confirm from phase_type.yml
  start_time: 1672578000
  end_time:   2000000000
  is_locked:  false
```

- [ ] **Step 3: Run — verify fail**

```bash
go test -tags 'bindata sqlite sqlite_unlock_notify' ./services/hackforger/ -run TestSubmitScores_EmptyRubric -v
```

Expected: test runs (fixture loaded) but fails because `SubmitScores` currently no-ops with empty scores; returns nil, not ErrNoRubricConfigured.

- [ ] **Step 4: Add the early check to `SubmitScores`**

In `services/hackforger/hackathon_judge.go`, in the `SubmitScores` function, after step 4 `rubric, err := GetEffectiveRubric(ctx, sub.TrackID)` (around line 55), add:

```go
	if len(rubric) == 0 {
		return ErrNoRubricConfigured{TrackID: sub.TrackID}
	}
```

- [ ] **Step 5: Run — verify PASS + existing tests still pass**

```bash
go test -tags 'bindata sqlite sqlite_unlock_notify' ./services/hackforger/ -run TestSubmitScores -v
```

Expected: all `TestSubmitScores*` PASS (including new one).

- [ ] **Step 6: Commit**

```bash
git add services/hackforger/hackathon_judge.go services/hackforger/hackathon_judge_test.go models/fixtures/*.yml
git commit -m "feat(hackforger): SubmitScores rejects empty rubric early (#27)"
```

---

## Task 6: Wire `ErrNoRubricConfigured` into `JudgeScoresPost` handler

**Files:**
- Modify: `routers/web/hackforger/hackathon.go:1265-1281` (inside `JudgeScoresPost` error switch)
- Modify: `options/locale/locale_en-US.ini` + `locale_zh-CN.ini`

- [ ] **Step 1: Add locale keys**

In `options/locale/locale_en-US.ini` near line 4194 (after `incomplete_rubric`):

```ini
hackathon.error.no_rubric_configured = No scoring criteria configured for this track — please contact the organizer.
```

In `options/locale/locale_zh-CN.ini` matching section:

```ini
hackathon.error.no_rubric_configured = 该赛道尚未配置评分标准，请联系组织者。
```

Verify:

```bash
grep "no_rubric_configured" options/locale/locale_*.ini
```

Expected: 2 lines.

- [ ] **Step 2: Add the handler branch**

In `routers/web/hackforger/hackathon.go` `JudgeScoresPost` (line 1267 — the `switch` block), add a new case before `IsErrInvalidHackathonPhase`:

```go
		case hackforger_service.IsErrNoRubricConfigured(err):
			msg = string(ctx.Tr("hackforger.hackathon.error.no_rubric_configured"))
			status = http.StatusBadRequest
```

- [ ] **Step 3: Build backend**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend
```

Expected: clean build.

- [ ] **Step 4: Commit**

```bash
git add routers/web/hackforger/hackathon.go options/locale/locale_en-US.ini options/locale/locale_zh-CN.ini
git commit -m "feat(hackforger): return 400 + i18n for ErrNoRubricConfigured"
```

---

## Task 7: Refactor `checkCriteriaModifiable` to op-aware form (TDD)

**Files:**
- Modify: `services/hackforger/hackathon_criteria.go:65-83` (function signature + body)
- Modify: `services/hackforger/hackathon_criteria.go:88,125,138` (3 callers)
- Create: `services/hackforger/hackathon_criteria_test.go` (new file)
- Modify: `options/locale/locale_en-US.ini` + `locale_zh-CN.ini` — no new key (reuse existing `criteria_locked`)

- [ ] **Step 1: Create failing tests**

```go
// Copyright 2026 The HackForger Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hackforger_test

import (
	"testing"

	"forgejo.org/models/db"
	hackforger_model "forgejo.org/models/hackforger"
	"forgejo.org/models/unittest"
	hackforger_service "forgejo.org/services/hackforger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Hackathon 1 is Judging (fixture).
// Hackathon 2 is Draft.

func TestAddCriteria_AllowedInJudging(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	err := hackforger_service.AddCriteria(db.DefaultContext, 1,
		"Late Added Criterion", "Added during Judging", 10.0, 50.0, 0)
	require.NoError(t, err, "Judging-phase add should succeed after refactor")
}

func TestAddCriteria_AllowedInDraft(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	err := hackforger_service.AddCriteria(db.DefaultContext, 2,
		"Draft Criterion", "", 10.0, 25.0, 0)
	require.NoError(t, err)
}

func TestAddCriteria_BlockedInFinalized(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Hackathon 5 is Finalized.
	err := hackforger_service.AddCriteria(db.DefaultContext, 5, "x", "", 10.0, 25.0, 0)
	require.Error(t, err)
	assert.True(t, hackforger_model.IsErrInvalidHackathonPhase(err))
}

func TestUpdateCriteria_BlockedInJudging(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Criterion 1 belongs to hackathon 1 (Judging).
	c, err := hackforger_model.GetCriteriaByID(db.DefaultContext, 1)
	require.NoError(t, err)
	c.Name = "Renamed During Judging"

	err = hackforger_service.UpdateCriteria(db.DefaultContext, c)
	require.Error(t, err)
	assert.True(t, hackforger_model.IsErrInvalidHackathonPhase(err),
		"Update during Judging must still be rejected")
}

func TestDeleteCriteria_BlockedInJudging(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	err := hackforger_service.RemoveCriteria(db.DefaultContext, 1)
	require.Error(t, err)
	assert.True(t, hackforger_model.IsErrInvalidHackathonPhase(err))
}
```

- [ ] **Step 2: Run — verify fail**

```bash
go test -tags 'bindata sqlite sqlite_unlock_notify' ./services/hackforger/ -run TestAddCriteria -v
go test -tags 'bindata sqlite sqlite_unlock_notify' ./services/hackforger/ -run TestUpdateCriteria_BlockedInJudging -v
go test -tags 'bindata sqlite sqlite_unlock_notify' ./services/hackforger/ -run TestDeleteCriteria_BlockedInJudging -v
```

Expected: `TestAddCriteria_AllowedInJudging` FAIL (current impl rejects). Update/Delete tests should already PASS.

- [ ] **Step 3: Refactor `checkCriteriaModifiable`**

Replace lines 65-83 of `services/hackforger/hackathon_criteria.go` with:

```go
// criteriaOp enumerates the CRUD operation attempted on criteria, so the gate
// can permit Add during Judging while still blocking Update/Delete.
type criteriaOp int

const (
	criteriaOpAdd criteriaOp = iota
	criteriaOpUpdate
	criteriaOpDelete
)

// checkCriteriaModifiable verifies the hackathon is in a phase that allows
// the requested criteria operation.
//   - Add:             allowed in Draft / Open / Hacking / Judging (rescue path)
//   - Update / Delete: allowed in Draft / Open / Hacking only
//   - Finalized / Cancelled: everything blocked
func checkCriteriaModifiable(ctx context.Context, hackathonID int64, op criteriaOp) error {
	h, err := hackforger_model.GetHackathonByID(ctx, hackathonID)
	if err != nil {
		return err
	}
	// Hard-block once finalized or cancelled — no changes ever.
	if h.StatusCache >= hackforger_model.HackathonStatusFinished {
		return hackforger_model.ErrInvalidHackathonPhase{
			HackathonID: h.ID,
			Current:     h.StatusCache,
			Expected:    hackforger_model.HackathonStatusHacking,
		}
	}
	// Judging: allow only Add (rescue), block Update/Delete.
	if h.StatusCache == hackforger_model.HackathonStatusJudging && op != criteriaOpAdd {
		return hackforger_model.ErrInvalidHackathonPhase{
			HackathonID: h.ID,
			Current:     h.StatusCache,
			Expected:    hackforger_model.HackathonStatusHacking,
		}
	}
	return nil
}
```

- [ ] **Step 4: Update 3 callers to pass op kind**

In the same file:

- Line ~88 (`AddCriteria`): change `checkCriteriaModifiable(ctx, hackathonID)` → `checkCriteriaModifiable(ctx, hackathonID, criteriaOpAdd)`
- Line ~125 (`UpdateCriteria`): change to `checkCriteriaModifiable(ctx, existing.HackathonID, criteriaOpUpdate)`
- Line ~138 (`RemoveCriteria`): change to `checkCriteriaModifiable(ctx, existing.HackathonID, criteriaOpDelete)`

- [ ] **Step 5: Run — verify all tests PASS**

```bash
go test -tags 'bindata sqlite sqlite_unlock_notify' ./services/hackforger/ -v
```

Expected: all tests PASS. Look carefully at any pre-existing criteria test to confirm it still passes with the new signature (it would — we only added a parameter, didn't change semantics in Draft/Open/Hacking).

- [ ] **Step 6: Commit**

```bash
git add services/hackforger/hackathon_criteria.go services/hackforger/hackathon_criteria_test.go
git commit -m "feat(hackforger): allow Add criteria during Judging (rescue path)"
```

---

## Task 8: Relax template gate + add Judging warning

**Files:**
- Modify: `templates/hackforger/hackathon/manage.tmpl:214` (gate condition)
- Modify: `templates/hackforger/hackathon/manage.tmpl` (~line 218, inside criteria card body — add warning)

- [ ] **Step 1: Change the outer gate**

Find line 214:
```
{{if lt .Hackathon.StatusCache 3}}
```

Change to:
```
{{if le .Hackathon.StatusCache 3}}
```

- [ ] **Step 2: Add Judging-phase info message**

Inside the criteria card body (`.hf-card-body`, around line 219), at the very top of the body (before `{{if .Criteria}}`), insert:

```html
				{{if eq .Hackathon.StatusCache 3}}
				<div class="ui info message">
					{{ctx.Locale.Tr "hackforger.hackathon.error.criteria_locked"}}
				</div>
				{{end}}
```

Note: reusing existing key `hackforger.hackathon.error.criteria_locked` ("Scoring criteria cannot be modified after judging has started.") — semantically conveys what's not allowed. (If you'd rather add a more positive "you can add but not edit/delete" key, introduce `hackforger.hackathon.manage.criteria.judging_add_only` in both locales and use that here instead. The spec lists it as optional polish.)

- [ ] **Step 3: Hide Edit / Delete buttons during Judging (Update/Delete are blocked server-side)**

Still in `manage.tmpl`, find the criteria table row actions (around line 243 — the `<td class="tw-flex tw-gap-1">` with Edit + Delete buttons). Wrap them with:

```html
									{{if lt $.Hackathon.StatusCache 3}}
										<button class="hf-btn hf-btn-primary" type="submit">{{ctx.Locale.Tr "hackforger.hackathon.manage.criteria.edit"}}</button>
									</form>
									<form method="post" action="..."/* delete form */>
										<button class="hf-btn hf-btn-danger" type="submit">{{ctx.Locale.Tr "hackforger.hackathon.manage.criteria.delete"}}</button>
									</form>
									{{end}}
```

Keep the `<form>` for the Edit button closed properly — don't orphan tags.

- [ ] **Step 4: Rebuild + manual smoke**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend
./gitea web &
# In browser: visit /hackathon/judging-hackathon/manage (fixture 1, Judging)
# Expect: criteria card shows, info banner at top, Edit/Delete buttons HIDDEN, Add form visible.
```

- [ ] **Step 5: Commit**

```bash
git add templates/hackforger/hackathon/manage.tmpl
git commit -m "feat(hackforger): show criteria card during Judging with add-only UX"
```

---

## Task 9: Define all new Vue-side locale keys in both files

**Files:**
- Modify: `options/locale/locale_en-US.ini`
- Modify: `options/locale/locale_zh-CN.ini`

- [ ] **Step 1: Add new keys (en-US)**

In `options/locale/locale_en-US.ini`, find the block around `hackathon.judge.*` (line 4169+). Append after `hackathon.judge.existing_score`:

```ini
hackathon.judge.score_saved_at = Saved at %s
hackathon.judge.update_scores = Update Scores
hackathon.judge.next_unscored = Next unscored →
hackathon.judge.rubric_not_configured = This track has no scoring criteria configured. Ask the organizer to add criteria.
hackathon.judge.error_generic = Could not save scores. Please try again.
```

- [ ] **Step 2: Mirror in zh-CN**

Same section of `options/locale/locale_zh-CN.ini`:

```ini
hackathon.judge.score_saved_at = 保存于 %s
hackathon.judge.update_scores = 更新分数
hackathon.judge.next_unscored = 下一个未评 →
hackathon.judge.rubric_not_configured = 该赛道尚未配置评分标准，请组织者补充后再评分。
hackathon.judge.error_generic = 保存分数失败，请重试。
```

- [ ] **Step 3: Verify each key is in both files**

```bash
for k in score_saved_at update_scores next_unscored rubric_not_configured error_generic; do
  echo "=== $k ==="
  grep -c "judge.$k " options/locale/locale_en-US.ini options/locale/locale_zh-CN.ini
done
```

Expected: each produces `1` for both files.

- [ ] **Step 4: Commit**

```bash
git add options/locale/locale_en-US.ini options/locale/locale_zh-CN.ini
git commit -m "i18n: judge score feedback strings (#27)"
```

---

## Task 10: `judge.tmpl` emits per-key `data-locale-*` attributes

**Files:**
- Modify: `templates/hackforger/hackathon/judge.tmpl`

- [ ] **Step 1: Replace the mount div**

In `templates/hackforger/hackathon/judge.tmpl`, replace:

```html
				<div id="hackforger-judge-scorecard"
					data-hackathon-slug="{{.Hackathon.Slug}}"
					data-tracks="{{.TracksJSON}}"
					data-submissions="{{.SubmissionsJSON}}"
					data-rubrics="{{.RubricsJSON}}">
				</div>
```

with:

```html
				<div id="hackforger-judge-scorecard"
					data-hackathon-slug="{{.Hackathon.Slug}}"
					data-tracks="{{.TracksJSON}}"
					data-submissions="{{.SubmissionsJSON}}"
					data-rubrics="{{.RubricsJSON}}"
					data-locale-score-saved="{{ctx.Locale.Tr "hackforger.hackathon.judge.scores_saved"}}"
					data-locale-score-saved-at="{{ctx.Locale.Tr "hackforger.hackathon.judge.score_saved_at"}}"
					data-locale-update-scores="{{ctx.Locale.Tr "hackforger.hackathon.judge.update_scores"}}"
					data-locale-submit-scores="{{ctx.Locale.Tr "hackforger.hackathon.judge.submit_scores"}}"
					data-locale-progress-label="{{ctx.Locale.Tr "hackforger.hackathon.judge.progress_detail"}}"
					data-locale-next-unscored="{{ctx.Locale.Tr "hackforger.hackathon.judge.next_unscored"}}"
					data-locale-rubric-not-configured="{{ctx.Locale.Tr "hackforger.hackathon.judge.rubric_not_configured"}}"
					data-locale-error-generic="{{ctx.Locale.Tr "hackforger.hackathon.judge.error_generic"}}">
				</div>
```

- [ ] **Step 2: Commit (template-only change, no build yet — all consumers updated in Task 11+)**

```bash
git add templates/hackforger/hackathon/judge.tmpl
git commit -m "feat(hackforger): emit data-locale-* on judge scorecard mount"
```

---

## Task 11: `init.js` wires messages prop

**Files:**
- Modify: `web_src/js/features/hackforger/init.js`

- [ ] **Step 1: Update the `JudgeScoreCard` mount block (lines 31-47)**

Replace:

```js
      createApp(JudgeScoreCard, {
        hackathonSlug: judgeEl.dataset.hackathonSlug,
        tracks: JSON.parse(judgeEl.dataset.tracks || '[]'),
        submissions: JSON.parse(judgeEl.dataset.submissions || '{}'),
        rubrics: JSON.parse(judgeEl.dataset.rubrics || '{}'),
      }).mount(judgeEl);
```

with:

```js
      createApp(JudgeScoreCard, {
        hackathonSlug: judgeEl.dataset.hackathonSlug,
        tracks: JSON.parse(judgeEl.dataset.tracks || '[]'),
        submissions: JSON.parse(judgeEl.dataset.submissions || '{}'),
        rubrics: JSON.parse(judgeEl.dataset.rubrics || '{}'),
        messages: {
          scoreSaved:          judgeEl.getAttribute('data-locale-score-saved') || 'Scores saved',
          scoreSavedAt:        judgeEl.getAttribute('data-locale-score-saved-at') || 'Saved at %s',
          updateScores:        judgeEl.getAttribute('data-locale-update-scores') || 'Update Scores',
          submitScores:        judgeEl.getAttribute('data-locale-submit-scores') || 'Submit Scores',
          progressLabel:       judgeEl.getAttribute('data-locale-progress-label') || '%d of %d submissions scored',
          nextUnscored:        judgeEl.getAttribute('data-locale-next-unscored') || 'Next unscored →',
          rubricNotConfigured: judgeEl.getAttribute('data-locale-rubric-not-configured') || 'No rubric configured',
          errorGeneric:        judgeEl.getAttribute('data-locale-error-generic') || 'Could not save scores',
        },
      }).mount(judgeEl);
```

Fallback strings protect against a template update race but should never be hit in practice.

- [ ] **Step 2: Commit**

```bash
git add web_src/js/features/hackforger/init.js
git commit -m "feat(hackforger): pass locale messages prop to JudgeScoreCard"
```

---

## Task 12: `JudgeScoreCard.vue` — imports, props, state (foundation)

**Files:**
- Modify: `web_src/js/components/hackforger/JudgeScoreCard.vue`

- [ ] **Step 1: Update `<script>` header**

Replace the top of the script:

```js
<script>
import {POST} from '../../modules/fetch.js';
```

with:

```js
<script>
import {POST} from '../../modules/fetch.js';
import {showInfoToast, showErrorToast} from '../../modules/toast.js';
import {formatDatetime} from '../../utils/time.js';
// NOTE: Forgejo's `showInfoToast` renders green with octicon-check (toast.js:11-15).
// It is the canonical success toast; no `showSuccessToast` exists. Do not swap.
```

- [ ] **Step 2: Add `messages` prop**

In the `props:` block, after `rubrics`, add:

```js
    messages: {type: Object, default: () => ({})},
```

- [ ] **Step 3: Extend `data()` with new reactive state**

Replace the `data() { return {...} }` block with:

```js
  data() {
    return {
      activeTrackId: this.tracks.length ? this.tracks[0].id : 0,
      scores: {},       // {subId: {criteriaId: {score, comment}}}
      saving: {},       // {subId: bool}
      saved: {},        // {subId: bool}
      errors: {},       // {subId: string}
      lastSavedAt: {},  // {subId: timestamp ms}
      expanded: {},     // {subId: bool} — collapsed summary unless true
      globalError: '',  // top-of-page error banner text
    };
  },
```

- [ ] **Step 4: Extend `created()` to init new per-submission state**

In `created()` after `this.errors[sub.id] = '';`, add:

```js
        this.lastSavedAt[sub.id] = 0;
        this.expanded[sub.id] = false;
```

- [ ] **Step 5: Add new computed + methods**

In `computed: {}`, after `progressPercent()`, append:

```js
    nextUnscoredSubId() {
      const s = this.activeSubmissions.find((s) => !this.saved[s.id]);
      return s ? s.id : null;
    },
```

In `methods: {}`, after `setComment`, before `submitScores`, append:

```js
    scrollToSubmission(subId) {
      const el = document.getElementById('submission-' + subId);
      if (el) el.scrollIntoView({behavior: 'smooth', block: 'center'});
    },
    formatTime(ms) {
      if (!ms) return '';
      // Forgejo locale-aware formatter; respects 12h/24h preference.
      return formatDatetime(new Date(ms), {hour: 'numeric', minute: '2-digit'});
    },
    fillTemplate(tpl, ...args) {
      // Replace %s (or %d) placeholders in order — locale ini uses these.
      let i = 0;
      return (tpl || '').replace(/%[sd]/g, () => (i < args.length ? String(args[i++]) : ''));
    },
```

- [ ] **Step 6: Replace `submitScores` with toast-aware version**

Replace the existing `submitScores` method body with:

```js
    async submitScores(submissionId) {
      this.saving[submissionId] = true;
      this.errors[submissionId] = '';
      this.globalError = '';

      if (this.activeRubric.length === 0) {
        this.globalError = this.messages.rubricNotConfigured;
        showErrorToast(this.globalError);
        this.saving[submissionId] = false;
        return;
      }

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
          this.lastSavedAt[submissionId] = Date.now();
          this.expanded[submissionId] = false;
          showInfoToast(this.messages.scoreSaved);
        } else {
          const data = await resp.json().catch(() => ({}));
          const msg = data.message || this.messages.errorGeneric;
          this.errors[submissionId] = msg;
          this.globalError = msg;
          showErrorToast(msg);
        }
      } catch (e) {
        this.errors[submissionId] = this.messages.errorGeneric;
        this.globalError = this.messages.errorGeneric;
        showErrorToast(this.messages.errorGeneric);
      }
      this.saving[submissionId] = false;
    },
```

- [ ] **Step 7: Build frontend + sanity check**

```bash
make frontend 2>&1 | tail -20
```

Expected: clean Webpack build (no errors). Warnings are OK.

- [ ] **Step 8: Commit**

```bash
git add web_src/js/components/hackforger/JudgeScoreCard.vue
git commit -m "feat(hackforger): wire JudgeScoreCard toasts + i18n prop"
```

---

## Task 13: `JudgeScoreCard.vue` — sticky progress bar + error banner (template)

**Files:**
- Modify: `web_src/js/components/hackforger/JudgeScoreCard.vue` (template block)

- [ ] **Step 1: Add sticky progress bar above existing progress card**

In the `<template>` block, replace the current "Progress bar" section:

```vue
  <!-- Progress bar -->
  <div class="hf-card">
    <div class="hf-card-body">
      <div class="tw-flex tw-justify-between tw-mb-2">
        <span>{{ scoredCount }} / {{ activeSubmissions.length }} submissions scored</span>
      </div>
      <div class="ui indicating progress">
        <div class="bar" :style="{width: progressPercent + '%'}"></div>
      </div>
    </div>
  </div>
```

with:

```vue
  <!-- Error banner (top) -->
  <div v-if="globalError" class="ui error message visible tw-mb-4" role="alert">
    {{ globalError }}
  </div>

  <!-- Sticky progress bar — top:0 because judge page has no secondary tabbar. -->
  <div class="hf-card" style="position: sticky; top: 0; z-index: 10; background: var(--color-box-body);">
    <div class="hf-card-body">
      <div class="tw-flex tw-justify-between tw-items-center tw-mb-2">
        <span>{{ fillTemplate(messages.progressLabel, scoredCount, activeSubmissions.length) }}</span>
        <button v-if="nextUnscoredSubId"
                class="hf-btn hf-btn-sm hf-btn-outline"
                @click="scrollToSubmission(nextUnscoredSubId)">
          {{ messages.nextUnscored }}
        </button>
      </div>
      <div class="ui indicating progress">
        <div class="bar" :style="{width: progressPercent + '%'}"></div>
      </div>
    </div>
  </div>
```

- [ ] **Step 2: Rebuild + visual smoke**

```bash
make frontend 2>&1 | tail -5
```

Visual check deferred to final E2E (Task 19) — no specific unit test here.

- [ ] **Step 3: Commit**

```bash
git add web_src/js/components/hackforger/JudgeScoreCard.vue
git commit -m "feat(hackforger): sticky progress bar + top error banner in JudgeScoreCard"
```

---

## Task 14: `JudgeScoreCard.vue` — summary card + Update Scores button

**Files:**
- Modify: `web_src/js/components/hackforger/JudgeScoreCard.vue` (submission card block)

- [ ] **Step 1: Add anchor id to submission cards + split into summary + form**

Find the submission card block starting with:

```vue
  <!-- Submission cards -->
  <div v-for="sub in activeSubmissions" :key="sub.id" class="hf-card">
```

Replace with:

```vue
  <!-- Submission cards -->
  <div v-for="sub in activeSubmissions" :key="sub.id"
       :id="'submission-' + sub.id"
       class="hf-card">
    <div class="hf-card-head">
      <span>{{ sub.title }}</span>
      <span v-if="sub.user_name" class="tw-text-sm tw-text-gray tw-ml-2">
        by <a :href="'/' + sub.user_name">{{ sub.user_name }}</a>
      </span>
      <span v-if="saved[sub.id]" class="hf-badge hf-badge-green">Scored</span>
    </div>

    <!-- Collapsed summary when saved and not expanded. -->
    <div v-if="saved[sub.id] && !expanded[sub.id]" class="hf-card-body">
      <div class="tw-flex tw-items-center tw-gap-2 tw-mb-3">
        <svg class="svg octicon-check-circle-fill" width="18" height="18" aria-hidden="true"><use xlink:href="#octicon-check-circle-fill"></use></svg>
        <span>{{ fillTemplate(messages.scoreSavedAt, formatTime(lastSavedAt[sub.id]) || '—') }}</span>
      </div>
      <div class="tw-grid tw-gap-2" style="grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));">
        <div v-for="c in activeRubric" :key="c.criteria_id">
          <div class="tw-text-sm tw-text-gray">{{ c.name }}</div>
          <strong>{{ scores[sub.id] && scores[sub.id][c.criteria_id] ? scores[sub.id][c.criteria_id].score : 0 }} / {{ c.max_score }}</strong>
        </div>
      </div>
      <button class="hf-btn hf-btn-outline tw-mt-3" @click="expanded[sub.id] = true">
        {{ messages.updateScores }}
      </button>
    </div>

    <!-- Editable form (default when unsaved, or expanded for updates). -->
    <div v-else class="hf-card-body">
      <p v-if="sub.track_name" class="tw-text-sm tw-mb-1">
        <span class="hf-badge">{{ sub.track_name }}</span>
      </p>
      <p v-if="sub.repo_full_name" class="tw-text-sm tw-mb-1">
        <svg class="svg octicon-repo" width="16" height="16" aria-hidden="true"><use xlink:href="#octicon-repo"></use></svg>
        <a :href="'/' + sub.repo_full_name">{{ sub.repo_full_name }}</a>
      </p>
      <p v-if="sub.description" class="tw-text-sm tw-text-gray">{{ sub.description }}</p>
      <p v-if="sub.demo_url"><a :href="sub.demo_url" target="_blank">Demo</a></p>

      <!-- Rubric-not-configured warning. -->
      <div v-if="!activeRubric.length" class="ui warning message">
        {{ messages.rubricNotConfigured }}
      </div>

      <!-- Criteria form -->
      <div v-else class="ui form tw-mt-4">
        <div v-for="c in activeRubric" :key="c.criteria_id" class="field">
          <label>{{ c.name }} <span class="tw-text-sm tw-text-gray">(0-{{ c.max_score }}, weight: {{ c.weight }})</span></label>
          <p v-if="c.description" class="tw-text-sm tw-text-gray tw-mb-2">{{ c.description }}</p>
          <div class="two fields">
            <div class="field">
              <input type="number" :min="0" :max="c.max_score" step="0.5"
                     :value="getScore(sub.id, c.criteria_id)"
                     @input="setScore(sub.id, c.criteria_id, $event)"
                     @change="setScore(sub.id, c.criteria_id, $event)"
                     placeholder="Score">
            </div>
            <div class="field">
              <input type="text"
                     :value="getComment(sub.id, c.criteria_id)"
                     @input="setComment(sub.id, c.criteria_id, $event)"
                     @change="setComment(sub.id, c.criteria_id, $event)"
                     placeholder="Comment (optional)">
            </div>
          </div>
        </div>
        <button class="hf-btn hf-btn-primary" :class="{loading: saving[sub.id]}"
                :disabled="saving[sub.id] || !activeRubric.length"
                @click="submitScores(sub.id)">
          {{ saved[sub.id] ? messages.updateScores : messages.submitScores }}
        </button>
        <button v-if="saved[sub.id] && expanded[sub.id]"
                class="hf-btn hf-btn-outline tw-ml-2"
                type="button"
                @click="expanded[sub.id] = false">
          Cancel
        </button>
      </div>
    </div>
  </div>
```

- [ ] **Step 2: Remove the now-duplicate fragment**

After the big block above, there may still be the original `<div class="hf-card-body">` + form markup (the bottom of the original component). Delete that to avoid duplication. The file after this task should have exactly one `v-for="sub in activeSubmissions"` block.

- [ ] **Step 3: Build**

```bash
make frontend 2>&1 | tail -5
```

Expected: clean build.

- [ ] **Step 4: Commit**

```bash
git add web_src/js/components/hackforger/JudgeScoreCard.vue
git commit -m "feat(hackforger): collapsed summary + Update Scores for judged submissions"
```

---

## Task 15: Feed HackathonScored redirect to `/leaderboard`

**Files:**
- Modify: `templates/hackforger/feed/community_feeds.tmpl:15`

- [ ] **Step 1: Change the target URL**

Find line 15:

```go
				{{else if eq .OpType 33}}{{ctx.Locale.Tr "hackforger.feed.hackathon_scored" .EntityName (HackforgerEntityURL .EntityType .EntitySlug)}}
```

Replace with:

```go
				{{else if eq .OpType 33}}{{ctx.Locale.Tr "hackforger.feed.hackathon_scored" .EntityName (printf "%s/leaderboard" (HackforgerEntityURL .EntityType .EntitySlug))}}
```

- [ ] **Step 2: Build + smoke**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend
./gitea web &
# Browser: open homepage feed. Click any "评审了 ... 的提交" entry.
# Expect: land on /hackathon/<slug>/leaderboard, not /hackathon/<slug>.
```

- [ ] **Step 3: Commit**

```bash
git add templates/hackforger/feed/community_feeds.tmpl
git commit -m "feat(hackforger): route scored feed to leaderboard (#27)"
```

---

## Task 16: Apply cleanup SQL + restart server with full rebuild

**Files:**
- No code changes — operational step.

- [ ] **Step 1: Verify scripts/cleanup-invalid-hackathons.sql is idempotent**

Re-run it:

```bash
sqlite3 data/forgejo.db < scripts/cleanup-invalid-hackathons.sql
echo "Exit: $?"
```

Expected: exit 0, no errors (hackathons 87/72/71/76/79 already gone, DELETE is a no-op).

- [ ] **Step 2: Full rebuild**

```bash
TAGS="bindata sqlite sqlite_unlock_notify" make backend
make frontend
```

Expected: both clean.

- [ ] **Step 3: Kill old instance, remove LevelDB lock if present, restart**

```bash
pkill -f "./gitea web" || true
rm -f data/queues/common/LOCK
./gitea web &
sleep 3
curl -s http://localhost:3000/ | head -c 200 && echo
```

Expected: HTML response starting with `<!DOCTYPE html>`.

- [ ] **Step 4: No commit (operational only).**

---

## Task 17: Write E2E prompt file

**Files:**
- Create: `docs/tests/e2e/tasks/issue-27-28-judge-score-feedback-e2e.md`

- [ ] **Step 1: Write the file**

Paste the E2E prompt content from `docs/superpowers/specs/2026-04-20-judge-score-feedback-design.md` § "E2E Prompt" into the new file. (The spec body contains the full TC1-TC7 markdown — copy it as-is. Adjust the leading front-matter/heading if needed so the file is self-contained.)

- [ ] **Step 2: Commit**

```bash
git add docs/tests/e2e/tasks/issue-27-28-judge-score-feedback-e2e.md
git commit -m "docs: E2E prompt for #27/#28 judge score feedback"
```

---

## Task 18: Run the E2E with agent-browser + write report

**Files:**
- Create: `docs/tests/e2e/reports/issue-27-28-judge-score-feedback-report.md`
- Create: `tests/screenshots/issue-27-28/*.png` (via agent-browser)

- [ ] **Step 1: Confirm server is running the freshly built binary**

```bash
lsof -iTCP:3000 -sTCP:LISTEN -n | grep gitea
stat -f '%Sm' ./gitea
```

Expected: `gitea` process listening on 3000; binary mtime within the last few minutes.

- [ ] **Step 2: Execute each TC via agent-browser**

Use the prompt's TC1-TC7 verbatim. For each:
1. Navigate, perform action, capture screenshot to `tests/screenshots/issue-27-28/tcN-name.png`
2. Record observed assertion: pass/fail + notes

Key tools:

```bash
agent-browser open <url>
agent-browser snapshot -i          # get element refs
agent-browser find role button "Submit Scores" click
agent-browser screenshot tests/screenshots/issue-27-28/tc3-success-feedback.png
agent-browser console              # after click, inspect console logs
agent-browser network requests --filter scores
```

For fixtures like "Judging phase with submissions" you may need to manipulate the DB (extend phase `end_time`). Record any such manipulation in the report.

- [ ] **Step 3: Write the report**

Create `docs/tests/e2e/reports/issue-27-28-judge-score-feedback-report.md`:

```markdown
# Issue #27 + #28 E2E Report

**Run date**: YYYY-MM-DD HH:MM
**Commit**: <short SHA>
**Server**: http://localhost:3000 (worktree build)

| TC | Description | Result | Screenshot |
|----|-------------|--------|------------|
| TC1 | Publish blocked without criteria | PASS/FAIL | screenshots/issue-27-28/tc1-publish-blocked.png |
| TC2 | Judging add-only | PASS/FAIL | screenshots/issue-27-28/tc2-judging-add-only.png |
| TC3 | Submit success feedback | PASS/FAIL | screenshots/issue-27-28/tc3-success-feedback.png |
| TC4 | Summary persists after refresh | PASS/FAIL | screenshots/issue-27-28/tc4-summary-persist.png |
| TC5 | Error banner on failure | PASS/FAIL | screenshots/issue-27-28/tc5-error-banner.png |
| TC6 | Feed → leaderboard | PASS/FAIL | screenshots/issue-27-28/tc6-feed-to-leaderboard.png |
| TC7 | Preview rankings | PASS/FAIL | screenshots/issue-27-28/tc7-preview-rankings.png |

## Notes
<observations per TC — network failures, console errors, DB manipulations performed>

## Regression check
<did any unrelated page regress? List pages visited.>

## Conclusion
<ready-to-merge / needs-rework + specific blockers>
```

- [ ] **Step 4: Commit**

```bash
git add docs/tests/e2e/reports/issue-27-28-judge-score-feedback-report.md tests/screenshots/issue-27-28/
git commit -m "test(e2e): #27/#28 judge score feedback E2E report"
```

---

## Task 19: Final verification before PR

**Files:**
- None — verification only.

- [ ] **Step 1: Full test suite**

```bash
go test -tags 'bindata sqlite sqlite_unlock_notify' ./services/hackforger/... ./models/hackforger/... -v 2>&1 | tail -50
```

Expected: 0 fails.

- [ ] **Step 2: Grep for forbidden patterns**

```bash
# No raw err.Error() going to Flash.Error without type check
grep -n 'ctx.Flash.Error.*err.Error()' routers/web/hackforger/*.go
# No mock DB in integration tests
grep -rn 'sqlmock\|gomock.*DB' services/hackforger/ tests/integration/hackforger*
# Any i18n key only in one locale?
for k in publish_requires_criteria track_requires_criteria no_rubric_configured score_saved_at update_scores next_unscored rubric_not_configured error_generic; do
  en=$(grep -c "$k" options/locale/locale_en-US.ini)
  zh=$(grep -c "$k" options/locale/locale_zh-CN.ini)
  [ "$en" = "1" ] && [ "$zh" = "1" ] || echo "MISSING: $k en=$en zh=$zh"
done
```

Expected: no output = all clean.

- [ ] **Step 3: Summary commit (if anything was fixed in Step 2)**

```bash
git status
# If there are last-minute fixes:
git add -A
git commit -m "chore: final cleanup pass for #27/#28"
```

- [ ] **Step 4: Ready to PR — do NOT push automatically.**

Human operator reviews the branch + E2E report, then opens PR via `gh pr create` when satisfied.

---

## Self-Review Results (post-authoring)

**Spec coverage:**
- A.1 (ErrNoCriteria) → Task 1, 2, 3 ✓
- A.2 (ErrNoRubricConfigured) → Task 4, 5, 6 ✓
- A.3 (feed only after writes) → Covered by A.2 early-reject (no-writes path returns error before feed). Task 5 ✓
- A.4 (template gate <= 3) → Task 8 ✓
- A.5 (op-aware checkCriteriaModifiable) → Task 7 ✓
- B.1 (Vue toasts + summary + sticky + banner) → Tasks 12, 13, 14 ✓
- B.2 (i18n per-key data-locale-*) → Tasks 9, 10, 11 ✓
- B.2a (computed + scroll/format) → Task 12 ✓
- B.3 (new data state) → Task 12 ✓
- B.4 (post-success interaction) → Covered inside Task 12 submitScores rewrite ✓
- C.1 (AddCriteria in Judging) → Task 7 ✓
- C.2 (manage warning) → Task 8 ✓
- D.1 (community feed → leaderboard) → Task 15 ✓
- D.2 (personal profile) → Deferred per spec "P2 follow-up" — not in this plan ✓
- E (cleanup SQL) → Task 0 ✓
- i18n table (reuse existing keys) → Task 9 adds only NEW keys; reused keys verified present in Task 19 ✓
- E2E → Tasks 17 + 18 ✓

**Placeholder scan:** No TBD/TODO/XXX in task bodies. Two places acknowledge design choices ("adapt to phase_type.yml", "either concrete value or new CSS var") — these are explicit branches with instructions, not placeholders.

**Type consistency:** `criteriaOp` enum (`criteriaOpAdd/Update/Delete`) introduced in Task 7 and referenced only there; `ErrNoCriteria` used in Tasks 1/2/3; `ErrNoRubricConfigured` in Tasks 4/5/6 — consistent. `messages` prop keys (camelCase: scoreSaved, scoreSavedAt, etc.) match between Task 11 (init.js) and Task 12 (Vue).

Plan ready to execute.
