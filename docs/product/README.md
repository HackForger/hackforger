# HackForger v0.1 — Product Documentation

## Documents

| Document | Purpose | Files |
|----------|---------|-------|
| [PRD](prd.md) ([中文](prd.zh-CN.md)) | Strategic context: problem, users, solution, metrics, scope | 1 file |
| [User Journeys](user-journeys/README.md) ([中文](user-journeys/zh-CN/README.md)) | Step-by-step specification: API endpoints, Git operations, Feed events, fixture hints | 6 files by role |
| [User Stories](user-stories/README.md) ([中文](user-stories/zh-CN/README.md)) | Development backlog: Mike Cohn + Gherkin acceptance criteria | 7 files by role |
| [API Rename Plan](api-rename-plan.md) | Endpoint naming cleanup, TDD-driven during E2E phase | 1 file |

## Document Hierarchy

```
PRD (why + what)
 └── User Journeys (how — step by step, per role)
      └── User Stories (dev-ready — acceptance criteria, per role)
           └── Test Fixtures + E2E Tests (generated from above)
```

## File Structure

```
docs/product/
├── README.md                 ← you are here
├── prd.md / prd.zh-CN.md
├── api-rename-plan.md
├── user-journeys/
│   ├── README.md             (index + global preconditions + verification matrix)
│   ├── organizer-hackathon.md  (J1)
│   ├── organizer-bounty.md     (J2a/2b/2c)
│   ├── organizer-grant.md      (J3)
│   ├── hacker-credits.md       (J4)
│   ├── hacker-social.md        (J5)
│   ├── platform-bot.md         (J6)
│   └── zh-CN/                  (中文镜像)
└── user-stories/
    ├── README.md             (index + story index table)
    ├── organizer.md          (H-001~005, H-009~010, G-001, G-003~005, B-001~006)
    ├── hacker.md             (H-006~007, G-002, C-001~002)
    ├── judge.md              (H-008)
    ├── admin.md              (C-003~004)
    ├── agent.md              (A-001~003, platform-bot)
    ├── shared.md             (F-001~004)
    └── zh-CN/                (中文镜像)
```

## Roles

角色是产品层面的概念，系统层面通过 capability (AccessMode × UnitType) 区分权限。见 PRD §3。

| Role | User Journey | User Stories |
|------|-------------|-------------|
| Organizer | J1 (Hackathon), J2 (Bounty), J3 (Grant) | 17 stories |
| Hacker | J4 (Credits), J5 (Social) | 5 stories |
| Judge | J1 (scoring steps) | 1 story |
| Admin | J4 (admin steps) | 2 stories |
| Platform-bot | J6 (admin's bot) | 3 stories |
| Shared | J5 (Feed, Explore) | 4 stories |

## Fixture Users

| Username | Role | Notes |
|----------|------|-------|
| `admin` | Admin | Site admin, owns platform-bot |
| `organizer` | Organizer | Creates hackathons, bounties, grants; owns `organizer/oss-project` |
| `hacker1` | Hacker | Active participant |
| `hacker2` | Hacker | Second participant, teammate/competitor |
| `judge1` | Judge | Technical reviewer |
| `judge2` | Judge | Second reviewer |
| `platform-bot` | Bot (admin's) | PAT auth, API-only, inherits admin capabilities |
