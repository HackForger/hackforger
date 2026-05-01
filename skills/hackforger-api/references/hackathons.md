# Hackathons API

Base path: `/api/v1/hackforger/hackathons`

## Lifecycle (Draft → Open → Hacking → Judging → Finished)

| Verb | Path | Purpose |
|------|------|---------|
| POST | `/hackforger/hackathons` | Create hackathon (Draft) |
| GET  | `/hackforger/hackathons` | List hackathons (filter via query) |
| GET  | `/hackforger/hackathons/{id}` | Get hackathon detail |
| PUT  | `/hackforger/hackathons/{id}` | Update hackathon fields |
| DELETE | `/hackforger/hackathons/{id}` | Delete (Draft only) |
| POST | `/hackforger/hackathons/{id}/publish` | Draft → Open |
| POST | `/hackforger/hackathons/{id}/cancel` | Cancel hackathon |

> **Phase advancement (Open → Hacking → Judging → Finished) is automatic.** A gocron scheduler (`hackforger_hackathon_status`, `@every 5m`) reads each hackathon's phase definitions and fires `onPhaseEvent` at start/end times. There is no API to manually advance — set phase `start_at`/`end_at` correctly via `/phases` and let the scheduler do it.

## Tracks

| Verb | Path |
|------|------|
| GET  | `/hackforger/hackathons/{id}/tracks` |
| POST | `/hackforger/hackathons/{id}/tracks` |
| PUT  | `/hackforger/hackathons/{id}/tracks/{tid}` |
| DELETE | `/hackforger/hackathons/{id}/tracks/{tid}` |

## Review criteria & rubric

| Verb | Path | Purpose |
|------|------|---------|
| GET  | `/hackforger/hackathons/{id}/criteria` | List hackathon-level criteria |
| POST | `/hackforger/hackathons/{id}/criteria` | Add criterion (weight, max score) |
| PUT  | `/hackforger/hackathons/{id}/criteria/{cid}` | Update criterion |
| DELETE | `/hackforger/hackathons/{id}/criteria/{cid}` | Delete criterion |
| GET  | `/hackforger/hackathons/{id}/tracks/{tid}/criteria` | Effective rubric for a track (after overrides) |
| PUT  | `/hackforger/hackathons/{id}/tracks/{tid}/criteria/{cid}` | Override criterion for a track |

## Registrations

| Verb | Path | Purpose |
|------|------|---------|
| POST | `/hackforger/hackathons/{id}/register` | Hacker registers for a track |
| GET  | `/hackforger/hackathons/{id}/registrations` | List registrations |
| PUT  | `/hackforger/hackathons/{id}/registrations/{rid}` | Update (e.g. team, track) |

### Register form (POST `/register`)

```json
{"team_name": "My Team", "track_id": 42}
```

`team_name` is required; `track_id` is optional (omit for hackathons without tracks).

**403 cases:**
- The user is a judge for any track in this hackathon (`JudgeCannotRegister`).
- Judges and participants are mutually exclusive.

**Other failure modes:** 400 (hackathon not in registration phase), 409 (already registered), 404 (hackathon not found).

## Submissions

| Verb | Path |
|------|------|
| POST | `/hackforger/hackathons/{id}/submissions` |
| GET  | `/hackforger/hackathons/{id}/submissions` |
| GET  | `/hackforger/hackathons/{id}/submissions/{sid}` |
| PUT  | `/hackforger/hackathons/{id}/submissions/{sid}` |
| DELETE | `/hackforger/hackathons/{id}/submissions/{sid}` |

## Judges & scoring

| Verb | Path | Purpose |
|------|------|---------|
| GET  | `/hackforger/hackathons/{id}/judges` | List judges |
| POST | `/hackforger/hackathons/{id}/judges` | Add judge |
| DELETE | `/hackforger/hackathons/{id}/judges/{uid}` | Remove judge |
| POST | `/hackforger/hackathons/{id}/submissions/{sid}/score` | Submit scores (judge) |
| GET  | `/hackforger/hackathons/{id}/submissions/{sid}/scores` | List scores for a submission |
| GET  | `/hackforger/hackathons/{id}/leaderboard` | Public leaderboard |

## Finalization

| Verb | Path | Purpose |
|------|------|---------|
| GET  | `/hackforger/hackathons/{id}/finalize-preview` | Preview rankings before settle |
| POST | `/hackforger/hackathons/{id}/finalize` | Confirm settlement (auto-distributes prize credits, advances to Finished) |

## Phase definitions (timeline CRUD)

These configure the *schedule* the cron uses. Editing a phase reschedules the gocron job for that hackathon.

| Verb | Path |
|------|------|
| GET  | `/hackforger/hackathons/{id}/phases` |
| POST | `/hackforger/hackathons/{id}/phases` |
| POST | `/hackforger/hackathons/{id}/phases/reorder` |
| PUT  | `/hackforger/hackathons/{id}/phases/{phase_id}` |
| DELETE | `/hackforger/hackathons/{id}/phases/{phase_id}` |
| GET  | `/hackforger/hackathons/{id}/current-phase` |

## Common chains

**Organizer setup → publish:**
1. POST `/hackforger/hackathons` (Draft)
2. POST `/hackforger/hackathons/{id}/tracks` (×N tracks)
3. POST `/hackforger/hackathons/{id}/criteria` (×3+ criteria)
4. POST `/hackforger/hackathons/{id}/judges` (assign judges)
5. POST `/hackforger/hackathons/{id}/phases` (set schedule, ALL 4 phase types: registration → development → judging → results)
6. POST `/hackforger/hackathons/{id}/publish` (requires at least 1 registration phase, otherwise 400)

### Bulk-create request body examples

```jsonc
// 1. POST /hackforger/hackathons — create hackathon
{"slug":"opc-2026-shuzhi-w1", "name":"【初赛W1】数智OPC加速赛", "org_id":1, "max_team_size":10, "description":"..."}
// org_id is REQUIRED by binding but the service overwrites it — pass any non-zero
// (admin user's id=1 is safe). Service auto-creates a dedicated org with the slug as name.

// 2. POST /hackforger/hackathons/{id}/tracks — add a track (×N times for N tracks)
{"name":"数字文化赛道", "slug":"digital-culture", "description":"AI+文娱、AI+教育等",
 "prize_credits":300, "prize_dist_mode":"winner_takes_all"}
// IMPORTANT: slug must be URL-safe (lowercase ASCII + hyphens). Chinese names
// get auto-rejected if used directly as slug. Service auto-creates a Forgejo
// repo `<hackathon-org>/<track-slug>` for the track.

// 3. POST /hackforger/hackathons/{id}/criteria — add a criterion (×N for N criteria)
{"name":"创新性与技术含量", "description":"项目是否具备核心技术或独特的创新理念",
 "max_score":10, "weight":30}
// max_score and weight are REQUIRED. Weights are summed across all criteria for
// the final 100 % normalization (so 30/30/30/10 == 100 %).

// 4. POST /hackforger/hackathons/{id}/judges — assign a judge (×N for N judges)
{"user_id":5, "track_id":3}
// BOTH user_id AND track_id REQUIRED. Judges are per-track, not global.
// (Judge for track A cannot score submissions in track B.)

// 5. POST /hackforger/hackathons/{id}/phases — add a phase (×4 for the 4 lifecycle phases)
{"phase_type_id":1, "start_time":1777583399, "end_time":1778015999}
// phase_type_id is the row id from the `phase_type` table — NOT a string key.
// Look up via GET /hackforger/admin/phase-types or query DB:
//   1=registration, 2=development, 3=judging, 4=results (default seeds)
// start_time/end_time are Unix epoch seconds.

// 6. POST /hackforger/hackathons/{id}/publish — no body
// 200 OK + {"status":"open"} on success
// 400 if no registration phase exists, or if already published
```

### Bulk-create gotchas

- **Track auto-creates a Forgejo repo** under the hackathon's auto-org. Repo name = track slug. So slugs must be URL-safe.
- **Submissions require user-owned repo, not track repo**. Hackers fork the track repo (or create their own), then POST submission with their `repo_id`. Submitting with the track's `repo_id` returns 500 "repo does not belong to user".
- **Phase advancement is automatic** via gocron (`@every 5m`). No API to manually advance — set `start_time`/`end_time` correctly and let the scheduler do it. For E2E acceleration, write status_cache + phase times directly via SQL.
- **Cover images / rich-text images**: upload via `POST /hackforger/attachments` (multipart), get `{uuid}`, embed in description as `![alt](/attachments/<uuid>)`. Public read of these requires the fix in PR #133 (RepoID=-1 attachments now publicly readable).

**Hacker journey:**
1. POST `/hackforger/hackathons/{id}/register` with body `{"team_name": "...", "track_id": N}` (NOT a query param; body fields required)
2. (Wait for Hacking phase — automatic)
3. POST `/hackforger/hackathons/{id}/submissions` (Fork+PR or Link Repo)
4. (Wait for Judging → Finished — automatic)
5. GET `/hackforger/credits/balance` (claim auto-distributed prize)

**Judge journey:**
1. (Wait for Judging phase)
2. GET `/hackforger/hackathons/{id}/submissions` (filter assigned)
3. POST `/hackforger/hackathons/{id}/submissions/{sid}/score` (×N submissions)

## Admin: phase type catalog

The **phase type catalog** is the global library of phase definitions (Registration, Development, Judging, etc.) that organizers pick from when configuring a hackathon's timeline. Per-hackathon `/phases` (above) creates *instances* from these types.

Auth: `reqToken() + reqSiteAdmin()`.

| Verb | Path | Purpose |
|------|------|---------|
| GET | `/hackforger/admin/phase-types` | List all phase types (flat array) |
| POST | `/hackforger/admin/phase-types` | Create (`activity_kind`, `key`, `display_name_i18n`, `is_unique`, `allowed_actions`, `default_order`) |
| PUT | `/hackforger/admin/phase-types/{id}` | Full overwrite |
| DELETE | `/hackforger/admin/phase-types/{id}` | Remove |

`activity_kind` is one of `hackathon`, `bounty`, `grant`. `display_name_i18n` is an i18n **key** (e.g. `hackforger.phase.registration`), not display text — clients translate via locale.
