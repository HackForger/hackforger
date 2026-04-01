# HackForger v0.1 — Product Requirements Document

> **Status:** Draft
> **Author:** Allen Woods
> **Date:** 2026-04-01
> **Branch:** v0.1-dev/hackforger

---

## 1. Executive Summary

HackForger is a Forgejo fork that transforms a self-hosted Git forge into a **complete hackathon, bounty, and grant platform** — where every competition artifact (registration, submission, judging, rewards) maps to native Git objects (orgs, repos, forks, PRs, reviews). By building on Forgejo instead of creating yet another standalone hackathon tool, HackForger eliminates the gap between "where you submit" and "where you build," giving organizers a turnkey platform and hackers a seamless code-to-rewards workflow — all powered by an internal Credits economy.

---

## 2. Problem Statement

### Who has this problem?

Three audiences:

1. **Hackathon organizers** (communities, companies, DAOs) who run coding competitions
2. **Open-source maintainers / project owners** who want to incentivize contributions via bounties
3. **Grant distributors** (foundations, companies) funding open-source projects

### What is the problem?

**For organizers:** Running a hackathon today requires stitching together 3-5 separate tools — Devpost or Devfolio for submissions, GitHub/GitLab for code, Google Forms for registration, Airtable for judging, and Stripe/PayPal for prizes. This causes:
- **Submission integrity gaps** — judges review a Devpost link, not actual code; timestamps are unreliable; plagiarism checking requires manual effort
- **Fragmented participant experience** — hackers register on one platform, code on another, submit on a third
- **No persistent community** — after the event, participants scatter; organizers rebuild from scratch for the next event

**For bounty organizers (repo maintainers):** GitHub Issues lack a native rewards mechanism. Bolt-on tools (Gitcoin, Algora) require participants to leave the forge, create separate accounts, and deal with payment infrastructure that isn't integrated with the codebase.

**For grant distributors:** Grant management is typically spreadsheet-driven with no connection to the funded projects' actual output.

### Why is it painful?

- **Organizers** spend 40%+ of their time on logistics instead of community building
- **Hackers** context-switch between 3+ tools per event, losing flow and motivation
- **Judges** evaluate submissions without easy access to code history, PR discussions, or commit quality
- **The entire ecosystem** lacks a unified incentive layer — there's no way to carry reputation or rewards across hackathons, bounties, and grants

### Evidence

- Devpost hosts 15,000+ hackathons but has zero Git integration — judges see a link, not a repo
- GitCoin Bounties (now deprecated) proved demand for code-native rewards but failed because it sat outside the forge
- Forgejo has 4,800+ stars and growing self-hosted adoption, but zero competition/incentive features
- Feedback from university hackathon organizers: "We spend more time managing submissions than mentoring"

---

## 3. Target Users & Personas

### Roles

HackForger 定义 4 个产品角色。**角色是产品层面的概念**，系统层面通过 capability（能访问什么、能执行哪些操作）来区分权限。一个用户可以同时拥有多个角色。

| Role | 产品定义 | 对应的 Capabilities |
|------|---------|-------------------|
| **Admin** | 平台管理者 | `*`（全部），拥有默认 platform-bot |
| **Organizer** | 组织者 — 创建和管理 Hackathon、Bounty、Grant | `hackathon.*`, `bounty.create`, `bounty.manage`, `grant.*`, `bot.create` |
| **Hacker** | 参与者 — 报名、提交作品、认领 Bounty、申请 Grant | `hackathon.register`, `hackathon.submit`, `bounty.claim`, `bounty.deliver`, `grant.apply`, `credits.redeem`, `bot.create` |
| **Judge** | 评审者 — 对提交物评分、Review PR | `judge.score`, `judge.review` |

> **设计原则：** 跟随 Forgejo 的 capability-based 权限模式。Forgejo 没有 "role" 表 — 它通过 `AccessMode × UnitType` 组合隐式表达角色。HackForger 同样不建 role 表，而是通过扩展 `UnitType`（新增 `TypeHackathon`, `TypeBounty`, `TypeGrant`, `TypeCredits`）复用现有 TeamUnit 机制。产品文档中的"角色"仅作为 capability 集合的语法糖。

### Bot 模型

Bot 不是独立角色，而是附着在用户上的自动化代理，权限继承自创建者（只能是子集）。

| Bot | 创建者 | 用途 |
|-----|--------|------|
| `platform-bot` | admin（平台默认创建） | 辅助平台管理、回复用户查询 |
| organizer 的 bot | organizer（可选） | 自动化 hackathon/bounty 管理 |
| hacker 的 bot | hacker（可选） | 自动发现和参与 bounty/hackathon |

Bot 使用 PAT 认证，`User.Type = UserTypeBot(4)`（Forgejo 原生支持），权限上限 = 创建者的 capability 集合。

### Personas

| Persona | Role | Motivation | Pain Point |
|---------|------|-----------|------------|
| **Organizer Olivia** | Community lead / DevRel | 办好 hackathon、发布 bounty、分配 grant | 需要同时管理 Devpost + GitHub + Google Forms + Airtable |
| **Hacker Hao** | Full-stack developer | 赢得奖金、积累作品集、学习新技术 | 在提交平台和代码平台之间反复切换 |
| **Judge Julia** | Technical reviewer | 给出公正的结构化评价 | 从截图链接评审代码体验极差 |
| **Admin Alex** | Platform operator | 维护平台运行、管理 Credits | 需要运营面板而非直接查数据库 |

### Jobs-to-Be-Done

| Job | Role | Success Criteria |
|-----|------|-----------------|
| 在一个平台完成 hackathon 全流程 | Organizer | 从创建到发奖零外部工具 |
| 在写代码的地方提交和被评审 | Hacker | Fork → code → PR → score，同一个 URL |
| 给 Issue 挂悬赏，完成后付款 | Organizer | Issue → Bounty → Claim → PR → Pay，5 次点击 |
| 带完整上下文评审代码质量 | Judge | 在同一视图看到 PR diff、commit history、评分标准 |
| 分配 grant 并跟踪产出 | Organizer | 申请 → 审批 → 资助 → 跟踪，无需 spreadsheet |
| 跨活动积累声誉 | Hacker | 一个 profile 展示 hackathon 成绩、bounty 完成、grant 获得 |
| Bot 自动参与平台活动 | Any (via bot) | Bot 通过 API 发现、申请、提交，权限受创建者约束 |

---

## 4. Strategic Context

### Vision

**Make every Git forge a hackathon platform.** Just as Forgejo turned self-hosting into a viable GitHub alternative, HackForger turns self-hosted Git into a viable Devpost + Gitcoin + Open Collective alternative — owned by the community, not a SaaS vendor.

### Why Git-Native?

The core architectural insight: **hackathon concepts already have natural Git equivalents**.

| HackForger Concept | Git Primitive | Why This Mapping |
|-------------------|--------------|------------------|
| Hackathon | Organization | Scoping (members, teams, repos), permissions, avatar |
| Track (赛道) | Repository | Code template, README with rules, Issues for Q&A |
| Registration | Org Membership | Already controls repo access |
| Submission | Fork + Pull Request | Timestamped, diffable, reviewable, unforgeable |
| Judging | PR Review + structured score | Inline comments + numerical criteria |
| Team | Org Team | Already handles group permissions |
| Results | Release | Permanent artifact with leaderboard + links |

This is not a metaphor — HackForger literally creates these Git objects. A submission IS a PR. A review IS a code review. This eliminates an entire class of integrity problems (timestamp forgery, plagiarism, "I submitted but the platform lost it").

### Why Now?

1. Forgejo v10+ stabilized the fork, making it safe to build on
2. Self-hosted Git adoption is growing (Forgejo, Gitea, GitLab CE) — communities want sovereignty
3. Devpost has stagnated; no new features for Git integration
4. AI agents are becoming hackathon participants — a Git-native platform naturally supports API-first agents
5. Credits/token economies are proven (GitPOAP, SourceCred) but none integrate with the forge itself

### Competitive Landscape

| Platform | Git Integration | Rewards | Self-Hosted | Weakness |
|----------|----------------|---------|-------------|----------|
| Devpost | None (link only) | External | No | Judges can't see code |
| Devfolio | GitHub OAuth | External | No | SaaS-only, closed source |
| Gitcoin | GitHub issues | Crypto | No | Deprecated bounty product |
| Algora | GitHub issues | USD | No | Bolt-on, not native |
| **HackForger** | **Native (IS the forge)** | **Credits (internal)** | **Yes** | **New, small community** |

---

## 5. Solution Overview

### Module Architecture

HackForger adds 5 modules to Forgejo, all in isolated `*/hackforger/` directories:

```
┌─────────────────────────────────────────────────────────┐
│                    HackForger Platform                    │
├──────────┬──────────┬──────────┬──────────┬──────────────┤
│ Hackathon│  Bounty  │  Grant   │ Credits  │ Feed +       │
│          │          │          │          │ Reputation   │
├──────────┴──────────┴──────────┴──────────┴──────────────┤
│              Forgejo Core (Git, Users, Orgs, PRs)         │
└─────────────────────────────────────────────────────────┘
```

**Module Dependency Graph:**

```
Hackathon ──→ Credits (prize distribution)
Bounty    ──→ Credits (reward payment)
Grant     ──→ Credits (grant distribution)
Feed      ←── ALL (event emission via PublishHackforgerAction)
Reputation ←── ALL (score aggregation from hackathon wins, bounty completions, grants)
Credits   ──→ (standalone, receives deposits from other modules)
```

### Module Boundaries

#### Hackathon Module
- **Owns:** Hackathon lifecycle (Draft → Open → Hacking → Judging → Finished/Cancelled)
- **Git Integration:** Auto-creates Org + Repos, registration = Org membership, submission = Fork + PR
- **Judging:** Multi-criteria scoring system (per-track criteria, weighted scores)
- **Constraint:** Maximum 10 tracks per hackathon
- **Output:** Leaderboard, Release with results

#### Bounty Module
- **Owns:** Bounty lifecycle (Open → Claimed → InReview → Completed → Paid / Expired / Cancelled)
- **Two Modes:** Exclusive (one claimant) vs. Competitive (multiple submissions, ranked winners)
- **Git Integration:** Mounted on Issue (1:1), delivery via PR, inherits all Issue capabilities
- **Output:** Credits payout to winner(s)

#### Grant Module
- **Owns:** Grant Round lifecycle (Draft → Open → Review → Finalized → Distributed / Cancelled)
- **Model:** Application-based (submit project → review → approve → allocate)
- **Two-Phase Funding:** `Finalized` = projects approved + budget allocated (commitment), `Distributed` = organizer manually releases Credits after reviewing project progress. Platform does not enforce update requirements — the organizer decides based on their own judgment (checking commits, PRs, milestones, etc.)
- **Constraint:** Total allocation ≤ round budget
- **Output:** Credits distribution to approved projects on organizer's manual trigger

#### Credits Module
- **Owns:** Account balances, transaction history, redemption, escrow
- **Operations:** Deposit (from Hackathon/Bounty/Grant wins), Withdraw (internal), Redeem (exchange for rewards), Escrow (platform hold for bounty rewards), Refund (escrow release back to organizer)
- **Transaction Types:** `deposit`, `withdraw`, `redeem`, `admin_deposit`, `admin_deduct`, `escrow`, `escrow_release`, `escrow_refund`
- **Admin:** Manual deposit/deduct, redeem option management, key pool, order fulfillment
- **Invariant:** All balance mutations use `db.WithTx` (ACID)

#### Feed + Reputation Module
- **Feed:** Separate `hackforger_action` table (not Forgejo's `action` table), 24 event types, 4 audience types (global/followers/org/watchers)
- **Reputation:** Weighted score from hackathon placements, bounty completions, grant funding; tiered badges
- **Discovery:** Explore pages for hackathons, bounties, grants; full-text search across entities

### State Machines

**Hackathon:**
```
Draft ──publish──→ Open ──start──→ Hacking ──start_judging──→ Judging ──finalize──→ Finished
  │                  │                │              │                                    
  └───cancel────────→└───cancel──────→└───cancel────→└───cancel──→ Cancelled
```

**Bounty (Exclusive):**
```
Open ──claim──→ Claimed ──submit_pr──→ InReview ──complete──→ Completed ──pay──→ Paid
  │                                                              │
  └──expire──→ Expired (Credits refunded to organizer)           └──cancel──→ Cancelled (Credits refunded)
```
> **Escrow:** When a bounty is created with Credits reward, the platform escrows the amount from the organizer's balance. On completion → pay to claimant. On expiry/cancellation → refund to organizer.

**Bounty (Competitive):**
```
Open ──(multiple submissions)──→ Open ──select_winners──→ Completed ──pay──→ Paid
  │
  └──expire/cancel──→ Expired/Cancelled (Credits refunded to organizer)
```

**Grant Round:**
```
Draft ──open──→ Open ──close──→ Review ──finalize──→ Finalized ──distribute──→ Distributed
  │               │                │                                              
  └──cancel──────→└──cancel───────→└──cancel──→ Cancelled
```

**Redeem Order:**
```
pending ──fulfill──→ fulfilled
  │
  └──cancel──→ cancelled
```

---

## 6. Success Metrics

### Primary Metrics (v0.1 targets — first 3 months after launch)

| Metric | Definition | Target |
|--------|-----------|--------|
| **Hackathon completion rate** | % of hackathons reaching Finished state | ≥ 70% |
| **Bounty completion rate** | % of bounties reaching Paid state | ≥ 50% |
| **Submission-via-PR rate** | % of hackathon submissions that are actual PRs (not external links) | 100% (by design) |

### Secondary Metrics

| Metric | Definition | Target |
|--------|-----------|--------|
| Credits flow-through | Total Credits deposited → redeemed ratio | ≥ 30% redeemed |
| Repeat participation | % of hackers participating in 2+ events | ≥ 25% |
| Judge response time | Average time from Judging phase start to all scores submitted | < 7 days |
| Grant budget utilization | % of round budget allocated to approved projects | ≥ 60% |
| Feed engagement | % of active users checking Feed at least weekly | ≥ 40% |

### Guardrail Metrics

| Metric | Constraint |
|--------|-----------|
| Forgejo upstream compatibility | Zero modifications to Forgejo core files outside 11 defined injection points |
| API response time | p95 < 500ms for all HackForger endpoints |
| Credits transaction integrity | Zero balance inconsistencies (enforced by `db.WithTx`) |

---

## 7. Feature Scope Summary

> **Note:** Detailed user stories will be in a separate `docs/user-stories.md` document.
> Detailed step-by-step journeys with API endpoints will be in `docs/user-journeys.md`.

### Epic 1: Hackathon Lifecycle
Create, publish, register, submit (Fork+PR), judge (multi-criteria), finalize, award Credits.

### Epic 2: Bounty Lifecycle
Create on Issue, two modes (Exclusive/Competitive), apply, claim, deliver (PR), complete, pay Credits.

### Epic 3: Grant Lifecycle
Create round, accept applications, review, allocate budget, finalize, distribute Credits.

### Epic 4: Credits Economy
Account management, deposit/withdraw, redeem options + key pool, order fulfillment, admin operations.

### Epic 5: Feed + Social + Discovery
24 event types, 4 audience types, explore pages, search, reputation scoring + leaderboard.

### Epic 6: AI Agent Participation
Full REST API coverage, PAT authentication, bot-friendly endpoints for automated participation.

### Cross-Cutting Concerns
- **i18n:** All user-facing text in `locale_en-US.ini` + `locale_zh-CN.ini`
- **Typed errors:** Service layer returns typed errors, web layer translates via `ctx.Tr()`
- **Feed events:** Every state change emits `PublishHackforgerAction`
- **ACID Credits:** All balance mutations inside `db.WithTx`

---

## 8. Out of Scope (v0.1)

| Feature | Reason | Future Version |
|---------|--------|---------------|
| Vue SPA frontend | v0.1 uses Go template SSR; Vue enhancement is incremental | v0.2 |
| Webhook event emission | Architecture exists, emission deferred | v0.2 |
| AI-assisted code review | Judge system is human-only for v0.1 | v0.3 |
| Cron-based auto phase transition | Infrastructure ready, logic deferred | v0.2 |
| External payment integration (Stripe, crypto) | Credits are internal-only; external cash-out deferred | v0.3 |
| Mobile-optimized UI | Desktop-first | v0.2 |
| Quadratic funding for Grants | Standard allocation model first | v0.3 |
| Multi-language hackathon content | i18n covers UI chrome, not user-generated content | v0.3 |
| Plagiarism detection | PR-based submission provides git blame/history for manual check | v0.3 |
| Advanced reputation formula tuning via admin UI | Weights configurable in code only | v0.2 |

---

## 9. Dependencies & Risks

### Technical Dependencies

| Dependency | Status | Impact |
|-----------|--------|--------|
| Forgejo v10+ stable API | ✅ Resolved | Foundation for all HackForger features |
| XORM auto-migration | ✅ Available | 16 new tables auto-created on startup |
| Forgejo Actions runner | ✅ Operational | Git operations (branch, commit, push, PR) use Actions workflows |
| LevelDB queue system | ✅ Available | Async event processing |

### Risks & Mitigations

| Risk | Severity | Mitigation |
|------|----------|-----------|
| **Upstream Forgejo merge conflicts** | Medium | Isolated `*/hackforger/` dirs; only 11 injection points touch upstream files |
| **Credits double-spend** | High | All mutations in `db.WithTx`; no concurrent balance updates without lock |
| **State machine invalid transitions** | Medium | Service layer validates current state before transition; typed errors |
| **Feed table growth** | Low | Indexed composite keys; pagination enforced; no full-table scans |
| **Hackathon Org pollution** | Low | Auto-created Orgs are clearly labeled; consider org type flag in v0.2 |
| **Judge bias / fairness** | Medium | Multi-criteria scoring with per-criteria weights; anonymized submission order (future) |

---

## 10. Open Questions

| # | Question | Status | Decision |
|---|----------|--------|----------|
| 1 | Should hackathon Orgs be visible in the global Org list? | **Decided** | Yes, with a "Hackathon" badge — increases discoverability |
| 2 | Can a user be both a judge and a participant in the same hackathon? | **Decided** | No — enforced at registration/judge-assignment time |
| 3 | Should expired bounties auto-refund Credits to the organizer? | **Decided** | Yes — platform escrows Credits when bounty is created; auto-refund to organizer's account on expiry/cancellation. This implies a `platform_escrow` transaction type or a platform hold account. |
| 4 | What happens to Credits when a user account is deleted? | **Decided** | Balance is forfeited (zeroed out). Simple and clean — no orphan funds. |
| 5 | Should competitive bounties allow late submissions after deadline? | **Decided** | No — deadline is hard cutoff |
| 6 | Maximum number of tracks per hackathon? | **Decided** | 10 tracks per hackathon (enforced at creation time). Revisit if organizers need more. |
| 7 | Should grant project updates be required after funding? | **Decided** | Two-phase funding: `Finalized` = approved + budget allocated, `Distributed` = funds actually released. The organizer manually reviews project progress (commits, PRs, milestones) before triggering distribution. Platform does NOT enforce update requirements — the organizer decides based on their own criteria. This maps cleanly to the existing `Finalized → Distributed` state transition. |
| 8 | Feed retention policy — how long to keep events? | **Open** | No cleanup in v0.1; consider archival in v0.2 |

---

## Appendix A: Technical Architecture Reference

### Layer Diagram

```
┌─────────────────────────────────┐
│  Templates (templates/hackforger/)│  ← Go HTML templates (SSR)
├─────────────────────────────────┤
│  Web Routes (routers/web/)       │  ← Session cookie auth
│  API Routes (routers/api/v1/)    │  ← Token/PAT auth
├─────────────────────────────────┤
│  Services (services/hackforger/) │  ← Business logic, state machines
├─────────────────────────────────┤
│  Models (models/hackforger/)     │  ← CRUD, XORM, 16 tables
├─────────────────────────────────┤
│  Forgejo Core                    │  ← Users, Orgs, Repos, PRs, Issues
└─────────────────────────────────┘
```

### Entity Count

- **16 database tables** (all prefixed with `hackforger_` via XORM defaults)
- **24 Feed event types** (ActionType 30–59)
- **26+ REST API endpoints** per module
- **32 HTML templates**
- **4 product roles** (admin, organizer, hacker, judge) mapped to capabilities, not stored as role table
- **7 test fixture users** (6 humans + 1 platform-bot)

### Injection Points into Forgejo

The 11 files where HackForger touches upstream Forgejo code:
1. `routers/web/web.go` — Route registration
2. `routers/api/v1/api.go` — API route registration
3. `models/migrations/migrations.go` — Migration registration
4. `modules/setting/setting.go` — Feature flag (if needed)
5. `templates/base/head_navbar.tmpl` — Navigation link
6. `templates/user/dashboard/feeds.tmpl` — Community tab
7. `templates/explore/navbar.tmpl` — Explore navigation
8. `options/locale/locale_en-US.ini` — i18n keys
9. `options/locale/locale_zh-CN.ini` — i18n keys
10. `cmd/web.go` — Module initialization
11. `models/forgejo_migrations/migrate.go` — HackForger table migrations
