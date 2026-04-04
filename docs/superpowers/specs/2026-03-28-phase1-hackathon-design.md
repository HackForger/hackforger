# Phase 1 Hackathon Design Spec

## Overview

Hackathon full-stack implementation: creation, registration, submission, judging, and finalization. Pure Go template SSR frontend, complete REST API, Feed event integration.

This is **Line B** of three parallel development lines. Line A (Bounty) and Line C (Grants+Credits) are handled by other developers on separate feature branches. Merge conflicts on shared files will be resolved at PR merge time.

## Scope

### In Scope
- Hackathon CRUD + slug-based routing
- Status machine (Draft → Open → Hacking → Judging → Finished / Cancelled)
- Track management (CRUD)
- Registration (apply + approve/reject)
- Submission (create + update, linked to track)
- Judge management (add/remove judges)
- Scoring (single score per judge per submission, auto-calculate submission totals + ranks)
- Leaderboard (public, ranked by total score)
- 6 Feed events via notifier
- 26 REST API endpoints with Swagger annotations
- 8 web pages (Go template SSR)
- i18n keys for all template text

### Out of Scope
- Vue components (pure SSR this phase)
- Webhook event emission (Week 5)
- AI review endpoint (Week 5)
- Credits integration (Line C responsibility)
- Reputation calculation (separate module)
- Cron auto-phase-transition (infrastructure exists, logic deferred)

## Architecture

### Layer Responsibilities

```
routers/web/hackforger/hackathon.go     — HTTP handlers, render templates
routers/api/v1/hackforger/hackathon.go  — REST API handlers, JSON responses
    ↓
services/hackforger/hackathon.go        — State machine transitions, business rules
services/hackforger/hackathon_judge.go  — Scoring + rank calculation
    ↓
models/hackforger/hackathon*.go         — CRUD data access (Get/List/Create/Update/Delete)
```

Upper layers call lower layers only. Models never call services.

### Models Layer (CRUD Functions)

5 existing model files already have basic CRUD (GetByID, GetBySlug, Create, Update, Delete, List). Functions to add or modify:

**hackathon.go** (already has: GetByID, GetBySlug, Create, Update, Delete, List):
- Modify `ListHackathons` to use `db.FindAndCount` returning `([]*Hackathon, int64, error)` for pagination
- Add `Keyword string` field to `ListHackathonsOptions` for `?q=` search (LIKE on Name/Description)
- Add `UpdateHackathonStatus(ctx, id, status) error` — targeted status-only update for state machine
- Note: existing code uses `HackathonStatusOpen` (not "Registration") — spec adopts this naming

**hackathon_track.go:**
- `ListTracksByHackathon(ctx, hackathonID) ([]*HackathonTrack, error)`
- `GetTrackByID(ctx, id) (*HackathonTrack, error)`
- `CreateTrack(ctx, t *HackathonTrack) error`
- `UpdateTrack(ctx, t *HackathonTrack) error`
- `DeleteTrack(ctx, id) error`
- `CountTracksByHackathon(ctx, hackathonID) (int64, error)` — needed for Publish precondition

**hackathon_registration.go:**
- `GetRegistration(ctx, hackathonID, userID) (*HackathonRegistration, error)`
- `ListRegistrations(ctx, hackathonID, opts) ([]*HackathonRegistration, int64, error)`
- `CreateRegistration(ctx, r *HackathonRegistration) error`
- `UpdateRegistrationStatus(ctx, id, status) error`
- `CountRegistrations(ctx, hackathonID) (int64, error)`

**hackathon_submission.go:**
- `GetSubmissionByID(ctx, id) (*HackathonSubmission, error)`
- `ListSubmissions(ctx, hackathonID, opts) ([]*HackathonSubmission, int64, error)`
- `CreateSubmission(ctx, s *HackathonSubmission) error`
- `UpdateSubmission(ctx, s *HackathonSubmission) error`
- `UpdateSubmissionRanks(ctx, hackathonID, rankings) error`
- `CountSubmissions(ctx, hackathonID) (int64, error)` — needed for StartJudging precondition

**hackathon_judge.go** (NEW file — judge-to-hackathon assignment):
- `HackathonJudge` struct: `{ID, HackathonID, UserID, CreatedUnix}` with UNIQUE(HackathonID, UserID)
- `AddJudge(ctx, hackathonID, userID) error`
- `RemoveJudge(ctx, hackathonID, userID) error`
- `ListJudges(ctx, hackathonID) ([]*HackathonJudge, error)`
- `IsJudge(ctx, hackathonID, userID) (bool, error)`
- `CountJudges(ctx, hackathonID) (int64, error)`

**hackathon_judge_score.go** (existing struct has: Score, Comment — no CriteriaJSON):
- Scoring uses a single `Score float64` field per judge per submission. No per-criteria breakdown in this phase.
- `GetScore(ctx, judgeID, submissionID) (*HackathonJudgeScore, error)`
- `ListScoresBySubmission(ctx, submissionID) ([]*HackathonJudgeScore, error)`
- `CreateScore(ctx, s *HackathonJudgeScore) error`
- `UpdateScore(ctx, s *HackathonJudgeScore) error`
- `HasAllJudgesScored(ctx, hackathonID, submissionID) (bool, error)` — cross-checks against HackathonJudge table

### Services Layer (Business Logic)

**hackathon.go — State Machine:**

```
Draft → Open           (Publish: requires >= 1 track)
Open → Hacking         (Start: manual or auto at RegistrationEnd)
Hacking → Judging      (StartJudging: requires >= 1 submission)
Judging → Finished     (Finalize: all submissions scored, ranks calculated)
Any non-Finished → Cancelled
```

Note: existing code uses `HackathonStatusOpen` (value 1) for the registration phase.

Each transition:
1. Validates preconditions
2. Updates status via model
3. Publishes Feed event via notifier

**hackathon_judge.go — Scoring:**

- `SubmitScore(ctx, judgeID, submissionID, score, comment)` — validates judge is assigned (via HackathonJudge table), hackathon is in Judging phase
- `CalculateRanks(ctx, hackathonID)` — computes average score per submission, assigns ranks. Called during Finalize.

### Notifier Integration

Add methods to `services/hackforger/notifier.go`:

| Event | ActionType | Audience | Trigger |
|-------|-----------|----------|---------|
| HackathonCreated | 30 | Global | CreateHackathon service |
| HackathonRegistered | 31 | Followers | Registration approved |
| HackathonSubmitted | 32 | Followers | Submission created |
| HackathonScored | 33 | Followers | Score submitted |
| HackathonPhaseChanged | 50 | Org members | Any phase transition |
| HackathonFinalized | 51 | Global | Finalize transition |

**Prerequisite:** The existing `PublishHackforgerAction` in notifier.go currently only writes for actor + global. This phase must extend it to resolve Followers and Org member audiences. This is a shared function so the implementation should be generic enough for all three lines to use.

### API Endpoints

All under `/api/v1/hackforger/hackathons/`:

```
POST   /hackathons                              — Create
GET    /hackathons                              — List (?status=&org_id=&q=&page=&limit=)
GET    /hackathons/{id}                         — Detail
PUT    /hackathons/{id}                         — Update
DELETE /hackathons/{id}                         — Delete (Draft only)
POST   /hackathons/{id}/publish                 — Draft → Registration
POST   /hackathons/{id}/start                   — Registration → Hacking
POST   /hackathons/{id}/start-judging           — Hacking → Judging
POST   /hackathons/{id}/finalize                — Judging → Finished
POST   /hackathons/{id}/cancel                  — Any non-Finished → Cancelled
GET    /hackathons/{id}/tracks                  — List tracks
POST   /hackathons/{id}/tracks                  — Create track
PUT    /hackathons/{id}/tracks/{tid}            — Update track
DELETE /hackathons/{id}/tracks/{tid}            — Delete track
POST   /hackathons/{id}/register                — Register
GET    /hackathons/{id}/registrations           — List registrations
PUT    /hackathons/{id}/registrations/{rid}     — Approve/reject
POST   /hackathons/{id}/submissions             — Submit
GET    /hackathons/{id}/submissions             — List submissions
GET    /hackathons/{id}/submissions/{sid}       — Submission detail
PUT    /hackathons/{id}/submissions/{sid}       — Update submission
GET    /hackathons/{id}/judges                  — List judges
POST   /hackathons/{id}/judges                  — Add judge
DELETE /hackathons/{id}/judges/{uid}            — Remove judge
POST   /hackathons/{id}/submissions/{sid}/score — Submit score
GET    /hackathons/{id}/submissions/{sid}/scores — List scores
GET    /hackathons/{id}/leaderboard             — Leaderboard
```

### Web Routes + Pages

| Route | Template | Description |
|-------|----------|-------------|
| `GET /explore/hackathons` | `hackforger/explore_hackathons.tmpl` | Public listing with filters |
| `GET /hackathons/new` | `hackforger/hackathon/new.tmpl` | Create form (auth required) |
| `POST /hackathons/new` | — | Handle creation |
| `GET /hackathon/{slug}` | `hackforger/hackathon/view.tmpl` | Detail page (phase-aware) |
| `POST /hackathon/{slug}/register` | — | Handle registration |
| `GET /hackathon/{slug}/submit` | `hackforger/hackathon/submit.tmpl` | Submission form |
| `POST /hackathon/{slug}/submit` | — | Handle submission |
| `GET /hackathon/{slug}/manage` | `hackforger/hackathon/manage.tmpl` | Organizer panel |
| `GET /hackathon/{slug}/judge` | `hackforger/hackathon/judge.tmpl` | Judge scoring page |
| `POST /hackathon/{slug}/judge/{sid}/score` | — | Handle score submission |
| `GET /hackathon/{slug}/leaderboard` | `hackforger/hackathon/leaderboard.tmpl` | Public leaderboard |

**Middleware:**
- `/explore/hackathons` — inherits `ignExploreSignIn` from existing explore group
- `/hackathons/new`, `/hackathon/{slug}/register`, `/hackathon/{slug}/submit` — `reqSignIn`
- `/hackathon/{slug}/manage/*` — `reqSignIn` + organizer check (OwnerID or OrgID admin)
- `/hackathon/{slug}/judge/*` — `reqSignIn` + judge check (via HackathonJudge table)

### Templates

All in `templates/hackforger/hackathon/`:

- `new.tmpl` — Create/edit form (name, slug, description, dates, org, template repo)
- `view.tmpl` — Detail page, content varies by phase (registration info, submission list, results)
- `submit.tmpl` — Submission form (track selector, demo URL, description)
- `manage.tmpl` — Organizer dashboard (tracks CRUD, registration list, phase control buttons, judge management)
- `judge.tmpl` — Scoring interface (submission list, single score input + comment per submission)
- `leaderboard.tmpl` — Ranked results table

The existing `explore.tmpl` will be kept as the shared explore layout. A new `explore_hackathons.tmpl` partial will render the hackathon-specific listing content, included when `PageIsExploreHackathons` is set.

### i18n Keys

All under `[hackforger]` section in `options/locale/locale_en-US.ini`. Keys needed:

- `hackathon.create_new`, `hackathon.edit`, `hackathon.delete_confirm`
- `hackathon.status.*` (already partially exist)
- `hackathon.field.*` (name, slug, description, dates, etc.)
- `hackathon.register`, `hackathon.register.success`, `hackathon.register.already_registered`
- `hackathon.submit`, `hackathon.submit.success`
- `hackathon.manage.*` (tracks, registrations, phases, judges)
- `hackathon.judge.*` (score, criteria, comment, submit_score)
- `hackathon.leaderboard.*` (rank, team, score, track)
- `hackathon.error.*` (not_found, invalid_phase, no_permission, etc.)

## Shared File Changes

| File | Change | Conflict Risk |
|------|--------|--------------|
| `services/hackforger/notifier.go` | Add hackathon event publish methods | Medium — all 3 lines append methods |
| `options/locale/locale_en-US.ini` | Add `hackathon.*` and `hackathon.judge.*` keys | Low — append-only, different key prefixes |
| `routers/web/web.go` | Register hackathon web route group | Low — each line adds its own group |
| `routers/api/v1/api.go` | Register hackathon API route group | Low — each line adds its own group |
| `templates/hackforger/explore.tmpl` | Add hackathon listing partial include | Low — conditional include |
| `models/forgejo_migrations/` | New migration for `hackforger_hackathon_judge` table | Low — append new migration file |

## Testing Strategy

Per test-plan-draft.md, Hackathon E2E tests cover:
1. Create hackathon as `org_alice`
2. Add tracks, add judges (`judge_carol`, `judge_dave`)
3. Publish → Registration phase
4. Register as `hacker_eve`, `hacker_frank`, `hacker_grace`
5. Approve registrations → Hacking phase
6. Submit projects
7. Start judging → Judging phase
8. Judges score all submissions
9. Finalize → Finished, verify ranks
10. Verify Feed events at each step

Each step validates the corresponding Feed event exists with correct audience.

### Manual E2E Prompt

After implementation, run through this on the internal instance (https://hackforger.inside.h2os.cloud):

1. Log in as `org_alice`, create a hackathon with 2 tracks
2. Publish it, verify it appears on `/explore/hackathons`
3. Log in as `hacker_eve`, register
4. As `org_alice`, approve registration, advance to Hacking
5. As `hacker_eve`, submit a project
6. As `org_alice`, advance to Judging
7. Log in as `judge_carol`, score the submission
8. As `org_alice`, finalize
9. Verify leaderboard shows correct ranking
10. Check Feed API for all 6 event types

Report template: `docs/tests/e2e/phase1-hackathon-e2e-report.md`
