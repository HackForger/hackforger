# HackForger User Journey 文档

> **Purpose:** Executable specification for test fixtures, API integration tests, and E2E test scripts.
> **Companion:** [PRD](../prd.md) · [User Stories](../user-stories/README.md)
> **Date:** 2026-04-01

The full journeys are split by primary role into separate files:

- [J1: Hackathon 全生命周期 (organizer)](organizer-hackathon.md)
- [J2: Bounty 全生命周期 (organizer)](organizer-bounty.md)
- [J3: Grant 全生命周期 (organizer)](organizer-grant.md)
- [J4: Credits 流转 + 兑换 (hacker/admin)](hacker-credits.md)
- [J5: Feed + 社交 + 发现 (hacker)](hacker-social.md)
- [J6: AI Agent 参与 (platform-bot)](platform-bot.md)

---

## Global Preconditions

The following users exist before any journey begins:

| Username | Role | Notes |
|----------|------|-------|
| `admin` | Platform administrator | Site admin privileges, can manage Credits |
| `organizer` | Hackathon/Bounty/Grant organizer | 创建和管理 Hackathon/Bounty/Grant，同时拥有 repo `organizer/oss-project` |
| `hacker1` | Participant | Active developer |
| `hacker2` | Participant | Second developer, teammate/competitor |
| `judge1` | Judge | Technical reviewer |
| `judge2` | Judge | Second reviewer |
| `platform-bot` | AI Agent (admin's bot) | admin 的默认 bot，PAT 认证，辅助平台管理 |

**Credits Ledger** — tracks running balances across journeys:

| User | After J1 | After J2 | After J3 | After J4 |
|------|----------|----------|----------|----------|
| `hacker1` | 500 | 500+300+150=950 | 950+1000=1950 | 1950-200=1750 |
| `hacker2` | 300 | 300+100=400 | 400 | 400+500-50=850 |
| `platform-bot` | 0 | 0+50=50 | 50 | 50 |
| `organizer` | 1000-800=200 | 200+100+100=400 (escrow churn: net 0 from J2a/J2b, +100 each from J2c.1/J2c.2 admin deposits) | 400+2000-1000=1400 | 1400 |

> Note: admin pre-deposits Credits to organizer's account before journeys that require spending.
> Organizer receives 1000 (J1 prize pool) + 600 (J2 bounties) + 100 (J2c.1) + 100 (J2c.2) + 2000 (J3 grant budget).

**Convention**: Each step lists **API** (REST endpoint) and/or **Web** (page route). All HackForger operations have both Web and API routes — Web uses form POST with redirect/flash, API uses JSON. Steps below primarily show API paths for test authoring; the corresponding Web paths follow the pattern:
- Hackathon: `/hackforger/hackathons/{slug}/...`
- Bounty: `/{owner}/{repo}/bounties/{id}/...`
- Grants: `/hackforger/grants/{slug}/...`
- Credits: `/hackforger/credits/...`
- Admin: `/hackforger/admin/credits/...`

**API Naming Deviations**: This document uses **target** endpoint names that align with Forgejo upstream convention (single-verb paths). Some differ from current codebase. These renames will be applied via TDD during the fixture/E2E phase. See [api-rename-plan.md](api-rename-plan.md) for full details.

| Endpoint in this doc | Current codebase | Change |
|---------------------|-----------------|--------|
| `POST /hackathons/{id}/judge` | `/hackathons/{id}/start-judging` | Rename to match `/publish`, `/start`, `/finalize` pattern |
| `POST /grants/rounds/{id}/distribute` | _(does not exist yet)_ | New endpoint for two-phase funding (PRD Q7) |

---

## Cross-Journey Verification Matrix

### All Entity States Covered

| Entity | States Covered | Journey |
|--------|---------------|---------|
| Hackathon | Draft, Open, Hacking, Judging, Finished | J1 |
| Hackathon (Cancelled) | — | Not covered in happy path; add test fixture with status=Cancelled(5) |
| Bounty | Open, Claimed, InReview, Completed, Paid | J2a |
| Bounty | Expired | J2c.1 |
| Bounty | Cancelled | J2c.2 |
| BountyMode | Exclusive(0), Competitive(1) | J2a, J2b |
| BountyApplication | Pending, Accepted, Rejected | J2a, J2c.3 |
| HackathonRegistration | Approved | J1 (auto-approve, skips Pending) |
| HackathonRegistration (Pending) | — | Add fixture for manual-review hackathon where registration stays Pending until organizer approves |
| HackathonRegistration (Rejected) | — | Add fixture for manual rejection case |
| GrantRound | Draft, Open, Review, Finalized, Distributed | J3 |
| GrantRound (Cancelled) | — | Not covered; add test fixture with status=Cancelled(5) |
| GrantProject | Pending, Approved, Funded, Rejected | J3 |
| RedeemOrder | pending, fulfilled | J4 |
| RedeemOrder (cancelled) | — | Add test: hacker1 cancels an order before fulfillment |
| CreditTransaction | deposit, withdraw, redeem, admin_deposit, admin_deduct, escrow, escrow_release, escrow_refund | J1-J4 |

### All Feed Events Covered

| Event | ActionType | Journey | Step |
|-------|-----------|---------|------|
| hackathon_created | 30 | J1 | 1.2 |
| hackathon_registered | 31 | J1 | 1.8, 1.9 |
| hackathon_submitted | 32 | J1 | 1.14, 1.15 |
| hackathon_scored | 33 | J1 | 1.17, 1.18 |
| bounty_created | 34 | J2a | 2a.2; J2b 2b.2 |
| bounty_claimed | 35 | J2a | 2a.5 |
| bounty_delivered | 36 | J2a | 2a.7 |
| bounty_completed | 37 | J2a | 2a.8 |
| bounty_winners_selected | 38 | J2b | 2b.5 |
| grant_round_created | 39 | J3 | 3.1 |
| grant_project_submitted | 40 | J3 | 3.3, 3.4 |
| grant_awarded | 41 | J3 | 3.6 |
| credits_redeemed | 42 | J4 | 4.6 |
| bounty_paid | 43 | J2a | 2a.9; J2b 2b.6 |
| hackathon_phase_changed | 50 | J1 | 1.6, 1.10, 1.16 |
| hackathon_finalized | 51 | J1 | 1.19 |
| bounty_expired | 52 | J2c | 2c.1 |
| bounty_cancelled | 53 | J2c | 2c.2 |
| grant_round_opened | 54 | J3 | 3.2 |
| grant_round_closed | 55 | J3 | 3.5 |
| grant_round_finalized | 56 | J3 | 3.7 |
| grant_round_cancelled | 57 | — | Not covered; add fixture |
| order_fulfilled | 58 | J4 | 4.7 |
| order_cancelled | 59 | — | Not covered; add fixture |

### Uncovered States & Events (Fixture-Only)

The following states and events are not part of the happy-path journeys but must exist in test fixtures:

1. **Hackathon Cancelled(5)** — organizer cancels a Draft hackathon
2. **HackathonRegistration Pending(0)** — hackathon configured for manual review, registration stays Pending until organizer acts
3. **HackathonRegistration Rejected(2)** — organizer manually rejects a registration
4. **GrantRound Cancelled(5)** — organizer cancels an Open grant round
5. **RedeemOrder cancelled** — user cancels pending order before admin fulfills
6. **grant_round_cancelled(57)** — feed event for cancelled round
7. **order_cancelled(59)** — feed event for cancelled order
8. **Account deletion with Credits** — balance is forfeited (zeroed out per PRD Q4)
9. **Track limit enforcement** — attempt to create 11th track returns error (max 10 per hackathon, per PRD Q6)

### Credits Ledger Final State

| User | Balance | Source Breakdown |
|------|---------|-----------------|
| `hacker1` | 1750 | +500 (J1 hackathon) +300 (J2a bounty) +150 (J2b bounty) +1000 (J3 grant) -200 (J4 redeem) |
| `hacker2` | 850 | +300 (J1 hackathon) +100 (J2b bounty) +500 (J4 admin deposit) -50 (J4 admin deduct) |
| `platform-bot` | 50 | +50 (J2b bounty 3rd) |
| `organizer` | 1400 | +1000 (J1 admin deposit) -800 (J1 prizes) +600 (J2 admin deposit) -300 (J2a escrow→released to hacker1) -300 (J2b escrow→released to winners) +100 (J2c.1 admin deposit) -100 (J2c.1 escrow) +100 (J2c.1 refund) +100 (J2c.2 admin deposit) -100 (J2c.2 escrow) +100 (J2c.2 refund) +2000 (J3 admin deposit) -1000 (J3 grant) = 1400 |

### All Roles Used

| Role | Journeys Active |
|------|----------------|
| `admin` | J1 (deposit), J2 (deposit), J3 (deposit), J4 (config, fulfill, deposit, deduct) |
| `organizer` | J1 (full lifecycle), J2 (full lifecycle — create, review, pay, cancel), J3 (full lifecycle) |
| `hacker1` | J1 (register, submit, win), J2 (claim, deliver), J3 (apply, funded), J4 (redeem), J5 (explore, search) |
| `hacker2` | J1 (register, team, submit), J2 (compete), J3 (apply, rejected), J5 (reaction) |
| `judge1` | J1 (score) |
| `judge2` | J1 (score) |
| `platform-bot` | J2b (compete, win 3rd), J6 (full API workflow) |

### Social Interactions Summary

| Interaction | Journey | Step |
|------------|---------|------|
| hacker1 follows organizer | J1 | 1.7 |
| hacker1 ↔ hacker2 mutual follow | J1 | 1.9 |
| Team "DeFi Duo" created | J1 | 1.9 |
| Star on track repo (x3) | J1 | 1.13 |
| hacker1 watches organizer/oss-project | J2b | 2b.3 |
| Reaction on issue comment | J5 | 5.10 |
