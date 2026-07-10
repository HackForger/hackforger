---
name: hackforger-api
description: Call HackForger platform API endpoints (hackathons, bounties, grants, credits, submissions, judging, feed, search, reputation, orgs, attachments). Use whenever the user asks to query or manage HackForger platform resources via natural language. Auth via FORGEJO_TOKEN env var, base URL via FORGEJO_URL — there is no default, ask the user which instance to target if FORGEJO_URL is not set.
---

# HackForger API skill

This skill lets you call any HackForger platform API endpoint via curl.
Design principle: **every web user action has an API equivalent** — if a user
can do it in the browser, you can do it through this skill.

## Step 0 — Confirm the instance URL (do this first, every time)

HackForger is deployed at multiple domains for different purposes (public
production, private internal, staging, self-hosted by other teams). **The skill
intentionally does not hard-code or default to any specific instance.** Before
making any API call:

1. Check whether `$FORGEJO_URL` is already set in the environment. If it is,
   echo it back to the user once for confirmation ("I'll target
   `$FORGEJO_URL` — confirm or override?") before proceeding.
2. If `$FORGEJO_URL` is **not** set, ask the user which HackForger instance
   they want to target. Examples you can offer to help them answer:
   - `https://hackforger.example.invalid` (example deployment)
   - `http://localhost:3000` (local dev)
   - or their own deployment URL
3. Export `FORGEJO_URL` for the session before issuing any curl call.

Never silently fall back to a default URL — a wrong-instance call can mutate
the wrong database.

## Authentication

Once the instance URL is confirmed, set the token:

```bash
export FORGEJO_TOKEN=<personal-access-token>      # required
export FORGEJO_URL=<the URL the user just confirmed in Step 0>
```

Generate a token at `<FORGEJO_URL>/-/user/settings/applications`. Scope must
include the resources you intend to touch (e.g., `write:repository`,
`admin:org` for org operations).

## Calling an endpoint

```bash
curl -s -X $METHOD \
  -H "Authorization: token $FORGEJO_TOKEN" \
  -H "Content-Type: application/json" \
  ${BODY:+-d "$BODY"} \
  "${FORGEJO_URL:?set FORGEJO_URL first — see Step 0}/api/v1$PATH" | jq .
```

Examples:
- List hackathons: `GET /hackforger/hackathons`
- Create hackathon: `POST /hackforger/hackathons` with `{"name":"...","slug":"..."}`
- Publish hackathon: `POST /hackforger/hackathons/{id}/publish`
- List bounties on a repo: `GET /repos/{owner}/{repo}/bounties`
- Apply for a grant: `POST /hackforger/grants/{round_id}/apply`

For complete endpoint coverage by module, consult the `references/` directory:

| File | Module |
|------|--------|
| `references/README.md` | Skill overview, design principle, E2E policy |
| `references/hackathons.md` | Hackathon CRUD, tracks, criteria, registrations, submissions, judging, finalize, phases, **admin phase type catalog** |
| `references/bounties.md` | Per-repo bounties: create, applications, review, complete, pay, cancel, winners |
| `references/grants.md` | Grant rounds: round CRUD, projects, approve/award/distribute |
| `references/credits.md` | Balance, transactions, redeem options + key pool, orders, admin deposit/deduct |
| `references/feed-search-reputation.md` | Global feed, unified search, reputation leaderboard, **reputation settings (admin)**, assistant chat |
| `references/orgs.md` | HackForger extensions to orgs: join-request notification |
| `references/attachments.md` | Platform-level (RepoID=-1) rich-text uploads |
| `references/user-stories.md` | E2E full-cycle user journey (Phase 0–10) → API call mapping |
| `references/gaps.md` | API ↔ Web parity gaps (currently empty backlog) |

Read the relevant `references/<module>.md` file when planning a multi-step
chain (e.g., "create hackathon + tracks + criteria + phases + publish") so
you don't miss a required step.

## Common pitfalls

1. **Hackathon publishing requires phases** — `POST /hackforger/hackathons/{id}/publish`
   fails with 422 if zero phases exist. Always create at least one phase
   (typically `registration` and `development`) before publishing.

2. **Standard Forgejo operations vs HackForger operations** — fork, follow,
   create-repo, create-org are Forgejo standard endpoints (`/api/v1/repos`,
   `/api/v1/users`, `/api/v1/orgs`). HackForger-specific resources are under
   `/api/v1/hackforger/...`. Use the right namespace.

3. **E2E testing policy** — if the user asks for END-TO-END web testing
   (clicking through the UI), DO NOT translate that into API calls. E2E must
   exercise the actual web flow. API calls are appropriate for setup/teardown
   and feature-level integration tests, not for E2E.

4. **Live instances have real data** — any production or production-like
   HackForger instance shares its DB with daily use. Don't assume Forgejo test
   fixtures (`user2`, `repo1`, `user5`) exist on the instance the user picked
   in Step 0. Query `GET /repos/search` or `GET /users/search` to find real
   targets before mutating anything.

## When the user asks to do something

1. Identify which module the request touches (hackathons / bounties / grants / etc.)
2. Read the corresponding `references/<module>.md` to confirm the endpoint and required body shape
3. Construct the curl call(s)
4. Execute, parse the JSON response, surface meaningful fields back to the user
5. If the request is a chain (e.g., "create hackathon and publish"), execute steps in order and stop on first failure
