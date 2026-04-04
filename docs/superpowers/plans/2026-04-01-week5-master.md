# Week 5 Master Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete HackForger v0.1 — TDD integration tests, search modal, webhook dispatch, CLI, E2E.

**Architecture:** TDD-first approach. Phase A builds test foundation (fixtures + renames). Phase B writes integration tests that drive feature implementation (search, assistant, webhook). Phase C adds frontend and CLI. Phase D validates via E2E.

**Tech Stack:** Go (XORM, chi router), Forgejo integration test framework, cobra CLI, vanilla JS modal, agent-browser E2E.

**Spec:** `docs/superpowers/specs/2026-04-01-week5-phase5-design.md`

---

## Sub-Plans

This master plan is split into 4 sub-plans matching the spec's execution phases. Each produces working, testable software independently.

| # | Sub-Plan | File | Depends On | Parallelizable |
|---|----------|------|------------|---------------|
| A | TDD Foundation (fixtures + renames + test skeleton) | `2026-04-01-week5-A-tdd-foundation.md` | — | No |
| B | Integration Tests + Feature Implementation | `2026-04-01-week5-B-integration-tests.md` | A | B1-B5 per worktree |
| C | Frontend + Swagger + CLI | `2026-04-01-week5-C-frontend-swagger-cli.md` | B | C2 parallel with C1/C3 |
| D | E2E Acceptance | `2026-04-01-week5-D-e2e.md` | B + C | D1-D2 during B/C |

## Execution Order

```
A (sequential) → B (B1-B5 parallel, B6-B8 sequential TDD) → C (C1→C3 sequential, C2 parallel) → D
```

**Merge conflict warning:** B6-B7 and C5 (i18n tasks) all modify `locale_en-US.ini` and `locale_zh-CN.ini`. These MUST run on the same branch sequentially, not in parallel worktrees.

## Key References

- Spec: `docs/superpowers/specs/2026-04-01-week5-phase5-design.md`
- User Journeys: `docs/product/user-journeys/`
- API Rename Plan: `docs/product/api-rename-plan.md`
- PRD: `docs/product/prd.md`
- Existing integration tests: `tests/integration/hackforger_credits_test.go`, `hackforger_grant_test.go`
- API routes: `routers/api/v1/api.go:1537-1868`
- Notifier: `services/hackforger/notifier.go`
- Webhook types: `modules/webhook/type.go`
