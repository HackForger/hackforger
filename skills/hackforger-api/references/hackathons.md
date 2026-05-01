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
5. POST `/hackforger/hackathons/{id}/phases` (set schedule)
6. POST `/hackforger/hackathons/{id}/publish`

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
