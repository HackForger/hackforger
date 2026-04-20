# Grants API

Base path: `/api/v1/hackforger/grants`. State machine: **Draft → Open → Review → Finalized → Distributed**.

## Round CRUD

| Verb | Path | Purpose |
|------|------|---------|
| GET  | `/hackforger/grants` | List rounds (filter by status, organizer) |
| POST | `/hackforger/grants` | Create round (Draft) |
| GET  | `/hackforger/grants/{slug}` | Get round detail |
| PUT  | `/hackforger/grants/{slug}` | Update (Draft only) |
| DELETE | `/hackforger/grants/{slug}` | Delete (Draft only) |
| GET  | `/hackforger/grants/{slug}/export` | CSV export |

## Round transitions

| Verb | Path | Transition |
|------|------|------------|
| POST | `/hackforger/grants/{slug}/open`       | Draft → Open (now accepting submissions) |
| POST | `/hackforger/grants/{slug}/close`      | Open → Review (no more submissions) |
| POST | `/hackforger/grants/{slug}/finalize`   | Review → Finalized (locks awards) |
| POST | `/hackforger/grants/{slug}/distribute` | Finalized → Distributed (releases credits) |
| POST | `/hackforger/grants/{slug}/cancel`     | Cancel from any non-distributed state (refunds) |

## Project submissions

| Verb | Path | Purpose |
|------|------|---------|
| GET  | `/hackforger/grants/{slug}/projects` | List projects in round |
| POST | `/hackforger/grants/{slug}/projects` | Hacker submits project |
| GET  | `/hackforger/grants/{slug}/projects/{pid}` | Get project |
| PUT  | `/hackforger/grants/{slug}/projects/{pid}` | Approve/reject (`{"action":"approve"}` or `"reject"`) |
| PUT  | `/hackforger/grants/{slug}/projects/{pid}/award` | Allocate award amount (`{"amount":300}`) |
| POST | `/hackforger/grants/{slug}/projects/{pid}/distribute` | Distribute single project (alternative to round-level distribute) |

## Common chains

**Organizer round flow:**
1. POST `/hackforger/grants` (Draft, with budget=500)
2. POST `/hackforger/grants/{slug}/open`
3. (Hackers POST `/projects` — multiple)
4. POST `/hackforger/grants/{slug}/close`
5. PUT `/projects/{pid}` `{"action":"approve"}` for selected
6. PUT `/projects/{pid}/award` `{"amount":300}` (sum must ≤ budget)
7. POST `/hackforger/grants/{slug}/finalize`
8. POST `/hackforger/grants/{slug}/distribute` (releases credits, fires Feed events)

## Constraints

- `award` total across approved projects must not exceed `budget`. Server returns 422 with `ErrAwardExceedsBudget`.
- Cannot reopen after `finalize`.
- `distribute` is idempotent — calling twice is rejected with `ErrAlreadyDistributed`.
