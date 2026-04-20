# Feed, Search, Reputation, Assistant

## Feed

| Verb | Path | Purpose |
|------|------|---------|
| GET  | `/hackforger/feed` | Activity feed (params: `scope=global|following|user|repo|org`, `actor_id`, `limit`, `cursor`) |

Event types include: hackathon_published, hackathon_phase_changed, registered, submitted, scored, finalized; bounty_created, bounty_claimed, bounty_paid; grant_round_opened, grant_project_submitted, grant_distributed; redeem_fulfilled; followed; etc.

## Unified search

| Verb | Path | Purpose |
|------|------|---------|
| GET  | `/hackforger/search` | Cross-entity search (params: `q`, `kind=hackathon|bounty|grant|repo|user`, `limit`) |

Backed by Bleve indexer. To force reindex (admin):
```
POST /hackforger/admin/reindex
```

## Reputation

| Verb | Path | Purpose |
|------|------|---------|
| GET  | `/hackforger/reputation/users/{username}` | One user's reputation breakdown |
| GET  | `/hackforger/reputation/leaderboard` | Top-N leaderboard (params: `limit`) |
| POST | `/hackforger/reputation/recalculate/{username}` | Force recalculation (admin only) |

### Reputation algorithm settings (admin)

| Verb | Path | Purpose |
|------|------|---------|
| GET | `/hackforger/admin/reputation/settings` | Read raw `weights` and `tiers` JSON |
| PUT | `/hackforger/admin/reputation/settings` | Update one or both fields (partial update; non-empty fields must be valid JSON) |

## Assistant chat

| Verb | Path | Purpose |
|------|------|---------|
| POST | `/hackforger/assistant/chat` | Chat with the platform assistant (`{"message":"..."}`) |
