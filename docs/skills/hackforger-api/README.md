# hackforger-api skill — references

The `/hackforger-api` slash command (defined in `.claude/commands/hackforger-api.md`) is a thin curl wrapper. These reference documents catalogue the actual endpoints and how to chain them.

## Files

- **[hackathons.md](hackathons.md)** — Hackathon CRUD, tracks, criteria, registrations, submissions, judging, finalize, phases.
- **[bounties.md](bounties.md)** — Per-repo bounties: create, applications, review, complete, pay, cancel, winners (competitive mode).
- **[grants.md](grants.md)** — Grant rounds (Draft→Open→Review→Finalized→Distributed), project submissions, approve/award/distribute.
- **[credits.md](credits.md)** — Balance, transactions, redeem options + key pool, redeem orders + fulfillment, admin deposit/deduct.
- **[feed-search-reputation.md](feed-search-reputation.md)** — Global feed, unified search, reputation leaderboard, assistant chat.
- **[user-stories.md](user-stories.md)** — E2E full-cycle user journey (Phase 0–10) mapped to API calls. Use this when a task is described in user-story form.
- **[gaps.md](gaps.md)** — Web-only operations that lack API equivalents (backlog for parity).

## Design principle

HackForger holds the principle that **every web-facing user operation must have an API equivalent**. The web (SSR + Vue partial) is a UI form factor, not a capability boundary. Web-only operations are coverage gaps — see `gaps.md` for the current list.

## E2E policy

E2E tests must use the web UI via `agent-browser` (see `docs/tests/e2e/`). The existence of API parity does **not** mean E2E should switch to API — the policies are independent:

- **API parity** ensures developers/scripts/CI/bots can drive the platform.
- **E2E web testing** ensures the UI itself works for real users.

Use this skill for: scripting, CI integration, dev fixtures, gap discovery, third-party automation. Do not use it as an E2E substitute.
