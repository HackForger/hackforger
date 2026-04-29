# Issue #77 — prize_dist_mode finalize-no-payout regression test

**Origin**: [issue #77](https://github.com/HackForger/hackforger/issues/77) E2E reported `POST /finalize` returns 200 but participant credits stay at 0.

**What this checks**: API track form now accepts `prize_dist_mode` + defaults to `winner_takes_all`; invalid modes / malformed ratios are rejected with `400` (was `500` or silent skip); `v14k` migration backfilled empty rows.

## Run (against any HackForger instance)

```bash
./issue-77-prize-dist-mode-e2e.sh
```

The script uses `hackforger:admin1234` basic auth against `http://localhost:3000`. Edit `BASE` and `AUTH` at the top of the script if pointing at a different instance.

Exits non-zero on any check failure. Cleans up the test hackathon at the end.

## What each step proves

| Step | API | Asserts |
|------|-----|---------|
| 1 | `GET /hackforger/hackathons/93/tracks` | Migration v14k ran (track 121 mode is `winner_takes_all`, was `''`) |
| 2 | `GET /users/hackforger/orgs` | Admin's org exists for hackathon scoping |
| 3 | `POST /hackforger/hackathons` | Hackathon create still works (no regression) |
| 4 | `POST /hackforger/hackathons/{id}/tracks` body without `prize_dist_mode` | **Original bug**: API now defaults to `winner_takes_all` instead of inserting `''` |
| 5 | Same with `prize_dist_mode: "definitely_not_a_mode"` | New `ErrInvalidPrizeDistMode` → `400` |
| 6 | tiered + ratios string `"not even json"` | `ParsePrizeDistRatios` now returns typed `ErrInvalidDistRatios` → `400` (was `500` before commit `f21a39ffae`) |
| 7 | tiered + ratios sum != 100 | Existing `ValidatePrizeDistRatios` → `400` |
| 8 | `DELETE /hackforger/hackathons/{id}` | Cleanup |

## What this does NOT cover

- **Full finalize → credits payout trip** — requires registering a participant, creating a submission, scoring, finalizing, and asserting credit balance. The integration test `tests/integration/hackforger_track_validation_test.go` covers the API contract; the full payout trip is delegated to the tester via the issue #77 follow-up comment.
- **Web UI flow** — the web track form already had the fallback (`prize_dist_mode = "winner_takes_all"` if empty); the bug was API-only. Web smoke is unchanged.
- **Bug 2 (`current-phase = null` / registration gating)** — separate investigation, not covered here.
