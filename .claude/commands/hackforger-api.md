---
description: Call HackForger API endpoints — covers all user-facing operations (hackathons, bounties, grants, credits, submissions, judging, feed)
allowed-tools: Bash, Read
---

Hit a HackForger API endpoint on the internal instance.

## Quick usage

```
/hackforger-api METHOD /path [JSON body]
```

Examples:
- `/hackforger-api GET /hackforger/hackathons`
- `/hackforger-api POST /hackforger/hackathons '{"name":"Web3 Challenge","slug":"web3"}'`
- `/hackforger-api POST /hackforger/hackathons/1/publish`

## Execute

```bash
curl -s -X $METHOD \
  -H "Authorization: token $FORGEJO_TOKEN" \
  -H "Content-Type: application/json" \
  ${BODY:+-d "$BODY"} \
  "https://hackforger.inside.h2os.cloud/api/v1$PATH" | jq .
```

If `FORGEJO_TOKEN` is not set, ask the user to export it first. Default base URL is the internal instance — override via `$FORGEJO_URL` if needed.

## Endpoint reference

This skill covers **every web-facing user operation** in HackForger. Pick the right reference for the task:

| Module | Reference | Covers |
|--------|-----------|--------|
| Hackathons | `docs/skills/hackforger-api/hackathons.md` | create / publish / register / submit / score / finalize / leaderboard / phases / criteria |
| Bounties | `docs/skills/hackforger-api/bounties.md` | create / apply / accept / complete / pay / cancel / winners (competitive) |
| Grants | `docs/skills/hackforger-api/grants.md` | round CRUD / open / close / projects / approve / award / distribute |
| Credits | `docs/skills/hackforger-api/credits.md` | balance / transactions / redeem options / orders / fulfill / admin deposit-deduct |
| Feed/Search/Reputation | `docs/skills/hackforger-api/feed-search-reputation.md` | global feed / unified search / reputation leaderboard / assistant chat |

Story-level mapping (E2E user journey → API call):
→ `docs/skills/hackforger-api/user-stories.md`

Known gaps (web-only operations that should have API parity but don't):
→ `docs/skills/hackforger-api/gaps.md`

## When the task involves more than one call

Read the relevant module reference before chaining. Most state machines (bounty open→claimed→inreview→completed→paid; grant draft→open→review→finalized→distributed; hackathon draft→open→hacking→judging→finished) have ordering constraints that the references document.

**Exception:** hackathon phase transitions (Open→Hacking→Judging→Finished) are **time-driven** by gocron, not API-triggered. Set phase start/end times via the phase CRUD endpoints; the scheduler advances state automatically.

## Authentication

- Token auth via `Authorization: token $FORGEJO_TOKEN` (Forgejo personal access token)
- Most write operations require `reqToken()`; admin operations also require `reqSiteAdmin()`
- For session-cookie web routes (Vue components), use the equivalent `/api/v1/hackforger/...` endpoint instead — they're functionally equivalent and easier to script.
